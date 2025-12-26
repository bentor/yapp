package yapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExamplePDFs(t *testing.T) {
	moduleRoot := findModuleRoot(t)
	pdfs := testPDFsFromEnv(moduleRoot)
	if len(pdfs) == 0 {
		t.Skip("no TEST_PDFS provided and no defaults found")
	}

	for _, pdfPath := range pdfs {
		if strings.TrimSpace(pdfPath) == "" {
			continue
		}
		pdfPath := pdfPath // capture for subtest
		t.Run(filepath.Base(pdfPath), func(t *testing.T) {
			t.Parallel()

			absPath := pdfPath
			if !filepath.IsAbs(absPath) {
				absPath = filepath.Join(moduleRoot, pdfPath)
			}

			if _, err := os.Stat(absPath); err != nil {
				t.Fatalf("stat %s: %v", absPath, err)
			}

			result, err := ParseFile(absPath)
			if err != nil {
				t.Fatalf("parse %s: %v", absPath, err)
			}
			ast := result.AST
			md := result.Markdown
			if len(ast.Pages) == 0 {
				t.Fatalf("no pages parsed for %s", absPath)
			}
			if strings.TrimSpace(md) == "" {
				t.Fatalf("empty markdown output for %s", absPath)
			}

			pretty, err := json.MarshalIndent(ast, "", "  ")
			if err != nil {
				t.Fatalf("marshal AST: %v", err)
			}

			resultDir := filepath.Join(resultRoot(moduleRoot), "test-result")
			if err := os.MkdirAll(resultDir, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", resultDir, err)
			}

			base := slugFromPath(pdfPath)
			astOut := filepath.Join(resultDir, base+".ast.json")
			mdOut := filepath.Join(resultDir, base+".md")

			if err := os.WriteFile(astOut, pretty, 0o644); err != nil {
				t.Fatalf("write AST %s: %v", astOut, err)
			}
			if err := os.WriteFile(mdOut, []byte(md), 0o644); err != nil {
				t.Fatalf("write markdown %s: %v", mdOut, err)
			}
		})
	}
}

func testPDFsFromEnv(moduleRoot string) []string {
	env := strings.TrimSpace(os.Getenv("TEST_PDFS"))
	if env != "" {
		return normalizePDFPaths(strings.Fields(env), moduleRoot, false)
	}

	// Default to bundled examples if present.
	return normalizePDFPaths([]string{
		"examples/test_doc.pdf",
		"examples/read_plain_text/pdf_test.pdf",
		"examples/read_text_with_styles/pdf_test.pdf",
	}, moduleRoot, true)
}

func findModuleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatalf("could not find go.mod from %s", dir)
		}
		dir = next
	}
}

func normalizePDFPaths(paths []string, moduleRoot string, dropMissing bool) []string {
	repoRoot := filepath.Dir(moduleRoot)
	var resolved []string

	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		if filepath.IsAbs(p) {
			if !dropMissing || pathExists(p) {
				resolved = append(resolved, p)
			}
			continue
		}

		candidates := []string{
			filepath.Join(moduleRoot, p),
			filepath.Join(repoRoot, p),
		}

		chosen := ""
		for _, candidate := range candidates {
			if pathExists(candidate) {
				chosen = candidate
				break
			}
		}

		if chosen == "" && !dropMissing {
			chosen = filepath.Join(moduleRoot, p)
		}

		if chosen != "" {
			resolved = append(resolved, chosen)
		}
	}

	return resolved
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func resultRoot(moduleRoot string) string {
	if filepath.Base(moduleRoot) == "src" {
		return filepath.Dir(moduleRoot)
	}
	return moduleRoot
}

func slugFromPath(p string) string {
	clean := filepath.Clean(p)
	clean = filepath.ToSlash(clean)
	if filepath.IsAbs(clean) {
		clean = filepath.Base(clean)
	}
	clean = strings.TrimPrefix(clean, "/")
	clean = strings.TrimPrefix(clean, "./")

	ext := filepath.Ext(clean)
	base := strings.TrimSuffix(clean, ext)
	base = strings.ReplaceAll(base, "..", "_")
	base = strings.ReplaceAll(base, "/", "_")
	if base == "" {
		base = "pdf"
	}
	return base
}
