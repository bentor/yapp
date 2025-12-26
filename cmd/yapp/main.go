package main

import (
	"flag"
	"fmt"
	"os"

	yapp "github.com/bentor/yapp"
)

func main() {
	var inPath, outPath string
	var debug bool
	flag.StringVar(&inPath, "in", "", "input PDF file")
	flag.StringVar(&outPath, "out", "", "output Markdown file")
	flag.BoolVar(&debug, "debug", false, "pretty-print the AST to stdout")
	flag.Parse()

	if inPath == "" || outPath == "" {
		fmt.Fprintf(os.Stderr, "usage: %s --in input.pdf --out output.md\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if err := yapp.Run(inPath, outPath, debug); err != nil {
		fmt.Fprintf(os.Stderr, "yapp failed: %v\n", err)
		os.Exit(1)
	}
}
