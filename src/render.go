package main

import (
	"strconv"
	"strings"
	"unicode"
)

func renderMarkdown(doc DocumentNode) string {
	var b strings.Builder

	for pageIdx, page := range doc.Pages {
		if len(doc.Pages) > 1 {
			b.WriteString("## Page ")
			b.WriteString(strings.TrimSpace(fmtInt(page.Number)))
			b.WriteString("\n\n")
		}

		// Flatten blocks into line strings.
		var lines []string
		for _, block := range page.Blocks {
			for _, line := range block.Lines {
				text := strings.TrimSpace(joinSpans(line.Spans))
				if text == "" {
					continue
				}
				lines = append(lines, normalizeSpaces(text))
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

		for i := 0; i < len(lines); i++ {
			trim := strings.TrimSpace(lines[i])

			if trim == "" {
				flushList()
				flushPara()
				continue
			}

			if firstHeading && pageIdx == 0 {
				flushList()
				flushPara()
				b.WriteString("# " + trim + "\n\n")
				firstHeading = false
				continue
			}
			if isHeadingCandidate(trim) {
				flushList()
				flushPara()
				b.WriteString("## " + trim + "\n\n")
				continue
			}

			if text, ok := stripBullet(trim); ok {
				flushPara()
				if text != "" {
					listItems = append(listItems, text)
				}
				continue
			}

			if strings.HasSuffix(trim, ":") && len(trim) < 60 {
				flushPara()
				words := strings.Fields(trim)
				if len(words) > 3 {
					b.WriteString("## " + trim + "\n\n")
					continue
				}
				item := trim
				j := i + 1
				for j < len(lines) {
					next := strings.TrimSpace(lines[j])
					if next == "" || isHeadingCandidate(next) || strings.HasSuffix(next, ":") {
						break
					}
					if _, ok := stripBullet(next); ok {
						break
					}
					item += " " + next
					j++
				}
				listItems = append(listItems, item)
				i = j - 1
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

func stripBullet(s string) (string, bool) {
	trimmed := strings.TrimSpace(s)
	bullets := []string{"•", "-", "*", "• ", "- ", "* "}
	for _, b := range bullets {
		if strings.HasPrefix(trimmed, b) {
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, b))
			return text, true
		}
	}
	return "", false
}

func hasBulletPrefix(s string) bool {
	_, ok := stripBullet(s)
	return ok
}

func normalizeSpaces(s string) string {
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}

func isHeadingCandidate(s string) bool {
	if len(s) < 3 || len(s) > 80 {
		return false
	}
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
		return false
	}
	ratio := float64(upper) / float64(letters)
	if ratio < 0.7 {
		return false
	}
	words := strings.Fields(s)
	if len(words) < 2 && len(s) < 5 {
		return false
	}
	return len(words) <= 8
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
