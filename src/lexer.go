package main

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Lexer walks the PDF and emits tokens akin to lex/flex.
type Lexer struct {
	path string
}

func NewLexer(path string) *Lexer {
	return &Lexer{path: path}
}

func (l *Lexer) Tokenize() ([]Token, error) {
	file, reader, err := pdf.Open(l.path)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer file.Close()

	tokens := make([]Token, 0)
	totalPages := reader.NumPage()

	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		if pageIndex > 1 {
			tokens = append(tokens, Token{
				Type: TokenPageBreak,
				Pos:  Position{Page: pageIndex},
			})
		}

		page := reader.Page(pageIndex)
		if page.V.IsNull() || page.V.Key("Contents").Kind() == pdf.Null {
			continue
		}

		content, err := page.GetPlainText(nil)
		if err != nil {
			return nil, fmt.Errorf("read page %d: %w", pageIndex, err)
		}

		lines := strings.Split(content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				tokens = append(tokens, Token{Type: TokenNewline, Pos: Position{Page: pageIndex}})
				continue
			}
			tokens = append(tokens, Token{
				Type:   TokenWord,
				Lexeme: trimmed,
				Pos: Position{
					Page: pageIndex,
				},
			})
			tokens = append(tokens, Token{Type: TokenNewline, Pos: Position{Page: pageIndex}})
		}
	}

	tokens = append(tokens, Token{
		Type: TokenEOF,
		Pos:  Position{Page: totalPages},
	})

	return tokens, nil
}
