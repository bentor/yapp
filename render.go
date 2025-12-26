package yapp

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type lineStyle struct {
	text     string
	fontSize float64
	spans    []TextSpan
	xs       []float64
}

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
				xs:       spanStarts(line.Spans),
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

		for i := 0; i < len(lines); i++ {
			line := lines[i]
			trim := strings.TrimSpace(line.text)

			if trim == "" {
				flushList()
				flushPara()
				continue
			}

			if hasBulletPrefix(trim) {
				flushPara()
				if text, ok := stripBullet(trim); ok {
					listItems = append(listItems, text)
				} else if text, ok := stripNumericBullet(trim); ok {
					listItems = append(listItems, text)
				}
				continue
			}

			// Opportunistic table detection: consecutive lines with aligned columns.
			if rows, used := consumeTable(lines[i:]); used > 0 {
				flushList()
				flushPara()
				renderTable(&b, rows)
				i += used - 1
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
			if isHeading {
				flushList()
				flushPara()
				b.WriteString("## " + trim + "\n\n")
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

func spanStarts(spans []TextSpan) []float64 {
	xs := make([]float64, 0, len(spans))
	for _, sp := range spans {
		xs = append(xs, sp.Pos.X)
	}
	return xs
}

func consumeTable(lines []lineStyle) ([][]string, int) {
	if len(lines) == 0 || len(lines[0].spans) < 3 {
		return nil, 0
	}

	colStarts := clusteredStarts(lines[0].spans, 24)
	colStarts = mergeStarts(colStarts, 40)
	if len(colStarts) < 3 || len(colStarts) > 6 {
		return nil, 0
	}
	if medianGap(colStarts) < 18 {
		return nil, 0
	}

	gap := medianGap(colStarts)

	rows := make([][]string, 0, 4)
	firstOK, firstRow := rowFits(colStarts, lines[0].spans, gap)
	if !firstOK {
		return nil, 0
	}
	rows = append(rows, firstRow)
	maxLen := maxCellLen(firstRow)
	seenNumeric := rowHasDigitOutsideFirst(firstRow)
	used := 1

	for used < len(lines) {
		sp := lines[used].spans
		if len(sp) < 2 {
			break
		}
		ok, row := rowFits(colStarts, sp, gap)
		if !ok {
			break
		}
		hasNum := rowHasDigitOutsideFirst(row)
		if seenNumeric && !hasNum {
			break
		}
		if maxCellLen(row) > 40 {
			break
		}
		rows = append(rows, row)
		if hasNum {
			seenNumeric = true
		}
		if l := maxCellLen(row); l > maxLen {
			maxLen = l
		}
		used++
	}

	if len(rows) < 3 {
		return nil, 0
	}
	if idx := firstNumericRow(rows); idx == -1 || idx > 2 {
		return nil, 0
	}
	return rows, used
}

func clusteredStarts(spans []TextSpan, tol float64) []float64 {
	var starts []float64
	for _, sp := range spans {
		x := sp.Pos.X
		placed := false
		for i, v := range starts {
			if math.Abs(v-x) <= tol {
				starts[i] = (v + x) / 2
				placed = true
				break
			}
		}
		if !placed {
			starts = append(starts, x)
		}
	}
	sort.Float64s(starts)
	return starts
}

func rowFits(cols []float64, spans []TextSpan, gap float64) (bool, []string) {
	if len(spans) < 2 {
		return false, nil
	}
	buckets := make([][]TextSpan, len(cols))
	var sumDist float64
	for _, sp := range spans {
		idx := nearest(cols, sp.Pos.X)
		dist := math.Abs(sp.Pos.X - cols[idx])
		sumDist += dist
		buckets[idx] = append(buckets[idx], sp)
	}

	meanDist := sumDist / float64(len(spans))
	if meanDist > gap*0.8 {
		return false, nil
	}

	filled := 0
	for _, bucket := range buckets {
		if len(bucket) == 0 {
			continue
		}
		filled++
		if w := width(bucket); w > gap*1.6+10 {
			return false, nil
		}
	}
	if filled < 2 || len(buckets[0]) == 0 {
		return false, nil
	}

	row := make([]string, len(cols))
	for i, bucket := range buckets {
		var texts []string
		for _, sp := range bucket {
			text := strings.TrimSpace(sp.Text)
			if text != "" {
				texts = append(texts, text)
			}
		}
		row[i] = normalizeSpaces(strings.Join(texts, " "))
	}
	return true, row
}

func medianGap(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	gaps := make([]float64, 0, len(xs)-1)
	for i := 1; i < len(xs); i++ {
		gaps = append(gaps, xs[i]-xs[i-1])
	}
	sort.Float64s(gaps)
	return gaps[len(gaps)/2]
}

func nearest(values []float64, target float64) int {
	closestIdx := 0
	closest := math.MaxFloat64
	for i, v := range values {
		if d := math.Abs(v - target); d < closest {
			closest = d
			closestIdx = i
		}
	}
	return closestIdx
}

func renderTable(b *strings.Builder, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	header := rows[0]
	sep := make([]string, len(header))
	for i := range sep {
		sep[i] = "---"
	}
	b.WriteString("| " + strings.Join(header, " | ") + " |\n")
	b.WriteString("| " + strings.Join(sep, " | ") + " |\n")
	for _, row := range rows[1:] {
		b.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}
	b.WriteString("\n")
}

func mergeStarts(xs []float64, tol float64) []float64 {
	if len(xs) == 0 {
		return xs
	}
	sort.Float64s(xs)
	merged := []float64{xs[0]}
	for _, x := range xs[1:] {
		last := merged[len(merged)-1]
		if math.Abs(x-last) <= tol {
			merged[len(merged)-1] = (last + x) / 2
		} else {
			merged = append(merged, x)
		}
	}
	return merged
}

func maxCellLen(row []string) int {
	max := 0
	for _, c := range row {
		if l := len(c); l > max {
			max = l
		}
	}
	return max
}

func firstNumericRow(rows [][]string) int {
	for i, row := range rows {
		if i == 0 {
			continue // skip header
		}
		for _, c := range row {
			if isNumericLike(c) {
				return i
			}
		}
	}
	return -1
}

func rowHasDigitOutsideFirst(row []string) bool {
	for i, c := range row {
		if i == 0 {
			continue
		}
		if isNumericLike(c) {
			return true
		}
	}
	return false
}

func isNumericLike(s string) bool {
	hasDigit := false
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsLetter(r):
			return false
		}
	}
	return hasDigit
}

func width(spans []TextSpan) float64 {
	if len(spans) == 0 {
		return 0
	}
	minX := spans[0].Pos.X
	maxX := spans[0].Pos.X
	for _, sp := range spans {
		if sp.Pos.X < minX {
			minX = sp.Pos.X
		}
		if sp.Pos.X > maxX {
			maxX = sp.Pos.X
		}
	}
	return maxX - minX
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
