package yapp

import (
	"encoding/json"
	"fmt"
	"os"
)

// Result holds the parsed AST and rendered Markdown.
type Result struct {
	AST      DocumentNode
	Markdown string
}

// ParseFile converts a PDF into a structured AST and Markdown string.
func ParseFile(inputPath string) (Result, error) {
	if inputPath == "" {
		return Result{}, fmt.Errorf("input path is required")
	}

	tokens, err := NewLexer(inputPath).Tokenize()
	if err != nil {
		return Result{}, fmt.Errorf("lexing failed: %w", err)
	}

	ast := NewParser(tokens).Parse()
	markdown := renderMarkdown(ast)
	return Result{AST: ast, Markdown: markdown}, nil
}

// Run converts a PDF to Markdown and writes it to disk. Suitable for CLI use.
func Run(inputPath, outputPath string, enableDebug bool) error {
	if inputPath == "" || outputPath == "" {
		return fmt.Errorf("both input and output paths are required")
	}

	result, err := ParseFile(inputPath)
	if err != nil {
		return err
	}

	if enableDebug {
		pretty, err := json.MarshalIndent(result.AST, "", "  ")
		if err != nil {
			return fmt.Errorf("debug: marshal AST: %w", err)
		}
		fmt.Println(string(pretty))
	}

	if err := writeMarkdown(outputPath, result.Markdown); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	return nil
}
func writeMarkdown(outPath, content string) error {
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write Markdown: %w", err)
	}
	return nil
}
