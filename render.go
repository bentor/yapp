package yapp

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func renderMarkdown(doc DocumentNode) string {
	var b strings.Builder
	bodySize := medianFontSize(doc)
	if bodySize == 0 {
		bodySize = 12
	}

	for pageIdx, page := range doc.Pages {
		if len(doc.Pages) > 1 {
			b.WriteString("## Page ")
			b.WriteString(strings.TrimSpace(fmtInt(page.Number)))
			b.WriteString("\n\n")
		}

		type lineStyle struct {
			text     string
			fontSize float64
			spans    []TextSpan
		}

		// Flatten blocks into line strings while preserving basic style hints.
		var lines []lineStyle
		for _, block := range page.Blocks {
			for _, line := range block.Lines {
				text := strings.TrimSpace(joinSpans(line.Spans))
				if text == "" {
					continue
				}
				lines = append(lines, lineStyle{
					text:     normalizeSpaces(text),
					fontSize: maxSpanSize(line.Spans),
					spans:    line.Spans,
				})
			}
		}

		// Render with generic structure detection.
		firstHeading := true
		var para []string
		var listItems []string
		flushPara := func() {
			if len(para) == 0 {
				return
			}
			b.WriteString(strings.Join(para, " ") + "\n\n")
			para = nil
		}
		flushList := func() {
			if len(listItems) == 0 {
				return
			}
			for _, item := range listItems {
				b.WriteString("- " + item + "\n")
			}
			b.WriteString("\n")
			listItems = nil
		}

		for _, line := range lines {
			trim := strings.TrimSpace(line.text)

			if trim == "" {
				flushList()
				flushPara()
				continue
			}

			isHeading := isHeadingCandidate(trim, line.fontSize, bodySize)

			if firstHeading && pageIdx == 0 && isHeading {
				flushList()
				flushPara()
				b.WriteString("# " + trim + "\n\n")
				firstHeading = false
				continue
			}
			if isHeading && !hasBulletPrefix(trim) {
				flushList()
				flushPara()
				b.WriteString("## " + trim + "\n\n")
				continue
			}

			if text, ok := stripBullet(trim); ok {
				flushPara()
				listItems = append(listItems, text)
				continue
			}

			if text, ok := stripNumericBullet(trim); ok {
				flushPara()
				listItems = append(listItems, text)
				continue
			}

			if strings.HasSuffix(trim, ":") && len(trim) < 60 {
				flushPara()
				b.WriteString("## " + trim + "\n\n")
				continue
			}

			para = append(para, trim)
		}
		flushList()
		flushPara()

		if len(doc.Pages) > 1 && pageIdx != len(doc.Pages)-1 {
			b.WriteString("\n")
		}
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

func joinSpans(spans []TextSpan) string {
	var b strings.Builder
	var lastText string
	var haveLast bool

	for _, span := range spans {
		text := strings.TrimSpace(span.Text)
		if text == "" {
			continue
		}

		if haveLast && !isPunctuation(text) && !strings.HasSuffix(lastText, "-") {
			b.WriteString(" ")
		}

		b.WriteString(text)
		lastText = text
		haveLast = true
	}

	return b.String()
}

var bulletPrefixes = []string{"•", "-", "*", "‣", "▪", "◦", "●", "–", "—", "·", "→", "»", "›"}

func stripBullet(s string) (string, bool) {
	trimmed := strings.TrimSpace(s)
	for _, marker := range bulletPrefixes {
		if strings.HasPrefix(trimmed, marker) {
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			return text, text != ""
		}
	}
	return "", false
}

func stripNumericBullet(s string) (string, bool) {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) < 2 {
		return "", false
	}
	i := 0
	for i < len(trimmed) && unicode.IsDigit(rune(trimmed[i])) {
		i++
	}
	if i > 0 && i < len(trimmed) && (trimmed[i] == '.' || trimmed[i] == ')') {
		body := strings.TrimSpace(trimmed[i+1:])
		if body != "" {
			return body, true
		}
	}
	if len(trimmed) >= 3 && unicode.IsLetter(rune(trimmed[0])) && (trimmed[1] == '.' || trimmed[1] == ')') {
		body := strings.TrimSpace(trimmed[2:])
		if body != "" {
			return body, true
		}
	}
	return "", false
}

func hasBulletPrefix(s string) bool {
	if _, ok := stripBullet(s); ok {
		return true
	}
	_, ok := stripNumericBullet(s)
	return ok
}

func normalizeSpaces(s string) string {
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}

func isHeadingCandidate(s string, fontSize, bodySize float64) bool {
	if len(s) < 3 || len(s) > 120 {
		return false
	}
	words := strings.Fields(s)
	ratio := uppercaseRatio(s)

	if bodySize > 0 && fontSize >= bodySize*1.35 && len(words) <= 14 {
		return true
	}
	if ratio > 0.65 && len(words) <= 10 && (bodySize == 0 || fontSize >= bodySize*1.05) {
		return true
	}
	if strings.HasSuffix(s, ":") && bodySize > 0 && fontSize >= bodySize*1.1 {
		return true
	}
	return false
}

func uppercaseRatio(s string) float64 {
	letters, upper := 0, 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	if letters == 0 {
		return 0
	}
	return float64(upper) / float64(letters)
}

func maxSpanSize(spans []TextSpan) float64 {
	var max float64
	for _, span := range spans {
		if span.Pos.FontSize > max {
			max = span.Pos.FontSize
		}
	}
	return max
}

func medianFontSize(doc DocumentNode) float64 {
	var sizes []float64
	for _, page := range doc.Pages {
		for _, block := range page.Blocks {
			for _, line := range block.Lines {
				for _, span := range line.Spans {
					if span.Pos.FontSize > 0 {
						sizes = append(sizes, span.Pos.FontSize)
					}
				}
			}
		}
	}
	if len(sizes) == 0 {
		return 0
	}
	sort.Float64s(sizes)
	return sizes[len(sizes)/2]
}

func isPunctuation(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune(",.;:!?\"'()-", r) {
			return false
		}
	}
	return true
}

func fmtInt(v int) string {
	return strconv.Itoa(v)
}
