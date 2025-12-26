package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	inPath  string
	outPath string
	debug   bool
)

func init() {
	flag.StringVar(&inPath, "in", "", "input PDF file")
	flag.StringVar(&outPath, "out", "", "output Markdown file")
	flag.BoolVar(&debug, "debug", false, "pretty-print the AST to stdout")
}

func main() {
	flag.Parse()

	if inPath == "" || outPath == "" {
		fmt.Fprintf(os.Stderr, "usage: %s --in input.pdf --out output.md\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	tokens, err := NewLexer(inPath).Tokenize()
	if err != nil {
		log.Fatalf("lexing failed: %v", err)
	}

	ast := NewParser(tokens).Parse()

	if debug {
		pretty, err := json.MarshalIndent(ast, "", "  ")
		if err != nil {
			log.Fatalf("debug: marshal AST: %v", err)
		}
		fmt.Println(string(pretty))
	}

	markdown := renderMarkdown(ast)

	if err := writeMarkdown(outPath, markdown); err != nil {
		log.Fatalf("write failed: %v", err)
	}

	log.Printf("wrote Markdown for %d pages to %s", len(ast.Pages), outPath)
}
func writeMarkdown(outPath, content string) error {
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write Markdown: %w", err)
	}
	return nil
}
