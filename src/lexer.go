package yapp

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

const (
	lineTolerance     = 2.5
	wordGapFloor      = 1.5
	wordGapScale      = 0.38
	trackingGapScale  = 1.6
	missingWidthScale = 0.6
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

		glyphs := page.Content().Text
		if len(glyphs) == 0 {
			continue
		}

		sort.Sort(pdf.TextVertical(glyphs))
		lines := groupLines(glyphs)

		var prevY, prevHeight float64
		var havePrev bool

		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			lineY := line[0].Y
			lineHeight := maxFontSize(line)

			if havePrev {
				gap := prevY - lineY
				if gap > math.Max(prevHeight, lineHeight)*1.35 {
					tokens = append(tokens, Token{Type: TokenNewline, Pos: Position{Page: pageIndex, Y: lineY}})
				}
			}

			words := buildWords(line, pageIndex)
			tokens = append(tokens, words...)
			if len(words) > 0 {
				tokens = append(tokens, Token{Type: TokenNewline, Pos: Position{Page: pageIndex, Y: lineY}})
			}

			prevY = lineY
			prevHeight = lineHeight
			havePrev = true
		}
	}

	tokens = append(tokens, Token{
		Type: TokenEOF,
		Pos:  Position{Page: totalPages},
	})

	return tokens, nil
}

func groupLines(glyphs []pdf.Text) [][]pdf.Text {
	var lines [][]pdf.Text
	var line []pdf.Text
	var anchorY float64

	for _, g := range glyphs {
		if g.S == "" {
			continue
		}
		if len(line) == 0 {
			anchorY = g.Y
			line = append(line, g)
			continue
		}
		if math.Abs(g.Y-anchorY) <= math.Max(lineTolerance, g.FontSize*0.35) {
			line = append(line, g)
			continue
		}
		sort.Sort(pdf.TextHorizontal(line))
		lines = append(lines, line)
		line = []pdf.Text{g}
		anchorY = g.Y
	}

	if len(line) > 0 {
		sort.Sort(pdf.TextHorizontal(line))
		lines = append(lines, line)
	}

	return lines
}

func buildWords(line []pdf.Text, page int) []Token {
	tokens := make([]Token, 0, len(line))
	var buf strings.Builder
	var start pdf.Text
	var last pdf.Text
	var haveWord bool

	flush := func() {
		word := cleanGlyphText(buf.String())
		if word == "" || !haveWord {
			buf.Reset()
			haveWord = false
			return
		}
		width := (last.X + last.W) - start.X
		tokens = append(tokens, Token{
			Type:   TokenWord,
			Lexeme: word,
			Pos: Position{
				Page:     page,
				X:        start.X,
				Y:        start.Y,
				Width:    width,
				Font:     start.Font,
				FontSize: start.FontSize,
			},
		})
		buf.Reset()
		haveWord = false
	}

	for _, g := range line {
		raw := g.S
		if raw == "\uFFFD" {
			continue
		}
		ch := strings.TrimSpace(raw)
		if ch == "" {
			// Preserve explicit spacing by flushing any buffered word.
			flush()
			continue
		}

		if !haveWord {
			start = g
			last = g
			buf.WriteString(ch)
			haveWord = true
			continue
		}

		gap := g.X - (last.X + glyphAdvance(last))
		threshold := math.Max(wordGapFloor, math.Max(last.FontSize, g.FontSize)*wordGapScale)
		if gap > threshold && !shouldJoinTracked(last, g, gap, threshold) {
			flush()
			start = g
		}
		buf.WriteString(ch)
		last = g
	}
	flush()
	return tokens
}

func cleanGlyphText(s string) string {
	s = strings.ReplaceAll(s, "\uFFFD", "")
	return strings.TrimSpace(s)
}

func glyphAdvance(g pdf.Text) float64 {
	if g.W > 0 {
		return g.W
	}
	text := strings.TrimSpace(g.S)
	if text == "" || g.FontSize <= 0 {
		return 0
	}
	runes := utf8.RuneCountInString(text)
	if runes == 0 {
		return 0
	}
	return float64(runes) * g.FontSize * missingWidthScale
}

func shouldJoinTracked(last, current pdf.Text, gap, threshold float64) bool {
	if last.W > 0 && current.W > 0 {
		return false
	}
	if gap > threshold*trackingGapScale {
		return false
	}
	if last.Font != current.Font || math.Abs(last.FontSize-current.FontSize) > 0.1 {
		return false
	}
	return isTrackingToken(strings.TrimSpace(last.S)) && isTrackingToken(strings.TrimSpace(current.S))
}

func isTrackingToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '.', '-', '/', '%', '°':
			continue
		default:
			return false
		}
	}
	return true
}

func maxFontSize(line []pdf.Text) float64 {
	var max float64
	for _, g := range line {
		if g.FontSize > max {
			max = g.FontSize
		}
	}
	return max
}
