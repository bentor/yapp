package yapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseExamplePDFs(t *testing.T) {
	pdfs := testPDFsFromEnv()
	if len(pdfs) == 0 {
		t.Skip("no TEST_PDFS provided and no defaults found")
	}

	moduleRoot := findModuleRoot(t)

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

			resultDir := filepath.Join(moduleRoot, "test-result")
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

func testPDFsFromEnv() []string {
	env := strings.TrimSpace(os.Getenv("TEST_PDFS"))
	if env != "" {
		return strings.Fields(env)
	}

	// Default to bundled examples if present.
	defaults := []string{
		"examples/test_doc.pdf",
		"examples/read_plain_text/pdf_test.pdf",
		"examples/read_text_with_styles/pdf_test.pdf",
	}
	var available []string
	for _, p := range defaults {
		if _, err := os.Stat(p); err == nil {
			available = append(available, p)
		}
	}
	return available
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

func slugFromPath(p string) string {
	clean := filepath.Clean(p)
	clean = filepath.ToSlash(clean)
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
