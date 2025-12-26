package yapp

// TokenType mirrors classic lexer phases.
type TokenType string

const (
	TokenWord      TokenType = "WORD"
	TokenNewline   TokenType = "NEWLINE"
	TokenPageBreak TokenType = "PAGE_BREAK"
	TokenEOF       TokenType = "EOF"
)

// Position captures where a piece of text lives on the page.
type Position struct {
	Page     int     `json:"page"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Font     string  `json:"font,omitempty"`
	FontSize float64 `json:"fontSize,omitempty"`
}

// Token is the output of the lexer.
type Token struct {
	Type   TokenType `json:"type"`
	Lexeme string    `json:"lexeme"`
	Pos    Position  `json:"pos"`
}

// DocumentNode is the AST root.
type DocumentNode struct {
	Pages []PageNode `json:"pages"`
}

// PageNode groups blocks on a page.
type PageNode struct {
	Number int         `json:"number"`
	Blocks []BlockNode `json:"blocks"`
}

// BlockNode is a sequence of lines (e.g., a paragraph).
type BlockNode struct {
	Lines []LineNode `json:"lines"`
}

// LineNode is a line of spans.
type LineNode struct {
	Spans []TextSpan `json:"spans"`
}

// TextSpan is a single token on a line.
type TextSpan struct {
	Text string   `json:"text"`
	Pos  Position `json:"pos"`
}
