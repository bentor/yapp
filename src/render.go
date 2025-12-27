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
	italic   bool
	y        float64
}

func renderMarkdown(doc DocumentNode) string {
	var b strings.Builder
	bodySize := medianFontSize(doc)
	if bodySize == 0 {
		bodySize = 12
	}

	var lastTableHeader []string

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
					italic:   spansAreItalic(line.Spans),
					y:        line.Spans[0].Pos.Y,
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
			if line.italic && !strings.HasPrefix(trim, "_") && !strings.HasSuffix(trim, "_") {
				trim = "_" + trim + "_"
			}

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
			if res := consumeTable(lines[i:], bodySize); res.used > 0 {
				flushList()
				flushPara()
				rows := res.rows
				if !res.hasHeader && len(lastTableHeader) > 0 && len(rows) > 0 && len(lastTableHeader) == len(rows[0]) && looksLikeSKU(rows[0][0]) {
					rows = append([][]string{lastTableHeader}, rows...)
				}
				renderTable(&b, rows)
				if res.hasHeader && len(rows) > 0 {
					lastTableHeader = rows[0]
				}
				i += res.used - 1
				continue
			}

			if isAsideCandidate(trim) {
				flushList()
				flushPara()
				if strings.HasPrefix(trim, "_") && strings.HasSuffix(trim, "_") && len(trim) > 2 {
					trim = strings.TrimSuffix(strings.TrimPrefix(trim, "_"), "_")
				}
				b.WriteString("_" + trim + "_\n\n")
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

func isAsideCandidate(s string) bool {
	if len(s) < 6 || len(s) > 140 {
		return false
	}
	if strings.Contains(s, "|") {
		return false
	}
	if hasBulletPrefix(s) {
		return false
	}
	idx := strings.IndexRune(s, ':')
	if idx < 1 || idx > 32 {
		return false
	}
	if strings.HasSuffix(s, ":") {
		return false
	}
	words := strings.Fields(s)
	if len(words) < 4 {
		return false
	}
	if uppercaseRatio(s) > 0.5 {
		return false
	}
	return true
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

func spansAreItalic(spans []TextSpan) bool {
	if len(spans) == 0 {
		return false
	}
	var italic int
	for _, sp := range spans {
		if isItalicFont(sp.Pos.Font) {
			italic++
		}
	}
	return float64(italic) >= float64(len(spans))*0.6
}

func isItalicFont(font string) bool {
	f := strings.ToLower(font)
	return strings.Contains(f, "italic") || strings.Contains(f, "oblique") || strings.Contains(f, "it")
}

type tableResult struct {
	rows      [][]string
	used      int
	hasHeader bool
}

type tableCell struct {
	startX float64
	text   string
}

func consumeTable(lines []lineStyle, bodySize float64) tableResult {
	if len(lines) == 0 {
		return tableResult{}
	}
	if !lineLooksTableStart(lines[0], bodySize) {
		return tableResult{}
	}

	window := len(lines)
	if window > 14 {
		window = 14
	}

	var starts []float64
	tableFontMax := 0.0
	tableLike := 0
	for i := 0; i < window; i++ {
		ln := lines[i]
		if ln.text == "" || len(ln.spans) == 0 {
			continue
		}
		startsForLine := lineCellStarts(ln.spans, ln.fontSize)
		if len(startsForLine) < 2 {
			continue
		}
		if bodySize > 0 && ln.fontSize > bodySize*0.92 && len(startsForLine) < 3 {
			continue
		}
		starts = append(starts, startsForLine...)
		tableLike++
		if ln.fontSize > tableFontMax {
			tableFontMax = ln.fontSize
		}
	}

	if tableLike < 2 || len(starts) < 4 {
		return tableResult{}
	}

	colStarts := clusteredStarts(starts, 24)
	colStarts = mergeStarts(colStarts, 40)
	if len(colStarts) < 3 || len(colStarts) > 6 {
		return tableResult{}
	}
	gap := medianGap(colStarts)
	if gap < 16 {
		return tableResult{}
	}

	var clusters [][]lineStyle
	var current []lineStyle
	used := 0
	var prevY float64
	var prevSize float64
	for used < len(lines) {
		ln := lines[used]
		if ln.text == "" {
			break
		}
		if tableFontMax > 0 && ln.fontSize > tableFontMax*1.25 {
			break
		}
		ok, _ := lineToRow(colStarts, ln.spans, gap, ln.fontSize)
		if !ok {
			break
		}
		if len(current) == 0 {
			current = append(current, ln)
		} else if rowBreak(prevY, ln.y, prevSize, ln.fontSize) {
			clusters = append(clusters, current)
			current = []lineStyle{ln}
		} else {
			current = append(current, ln)
		}
		prevY = ln.y
		prevSize = ln.fontSize
		used++
	}
	if len(current) > 0 {
		clusters = append(clusters, current)
	}

	if len(clusters) < 2 {
		return tableResult{}
	}

	rows := make([][]string, 0, len(clusters))
	for _, group := range clusters {
		row := make([]string, len(colStarts))
		for _, ln := range group {
			ok, cells := lineToRow(colStarts, ln.spans, gap, ln.fontSize)
			if !ok {
				continue
			}
			for i, cell := range cells {
				if cell == "" {
					continue
				}
				if row[i] == "" {
					row[i] = cell
				} else {
					row[i] = strings.TrimSpace(row[i] + " " + cell)
				}
			}
		}
		if countNonEmpty(row) >= 2 {
			rows = append(rows, row)
		}
	}

	if len(rows) < 2 {
		return tableResult{}
	}

	return tableResult{
		rows:      rows,
		used:      used,
		hasHeader: rowLooksLikeHeader(rows[0]),
	}
}

func lineLooksTableStart(line lineStyle, bodySize float64) bool {
	if line.text == "" || len(line.spans) == 0 {
		return false
	}
	starts := lineCellStarts(line.spans, line.fontSize)
	if len(starts) >= 3 {
		return true
	}
	if lineHasSKU(line.spans) {
		return true
	}
	if hasHeaderKeyword(line.text) {
		return true
	}
	if bodySize > 0 && line.fontSize <= bodySize*0.92 && len(starts) >= 2 {
		return true
	}
	return false
}

func lineCellStarts(spans []TextSpan, fontSize float64) []float64 {
	cells := lineCells(spans, fontSize)
	if len(cells) == 0 {
		return nil
	}
	starts := make([]float64, 0, len(cells))
	for _, cell := range cells {
		starts = append(starts, cell.startX)
	}
	return starts
}

func cellGapThreshold(fontSize float64) float64 {
	if fontSize <= 0 {
		return 12
	}
	threshold := fontSize * 1.65
	if threshold < 12 {
		threshold = 12
	}
	return threshold
}

func lineCells(spans []TextSpan, fontSize float64) []tableCell {
	if len(spans) == 0 {
		return nil
	}
	threshold := cellGapThreshold(fontSize)
	var cells []tableCell
	var buf []TextSpan
	startX := spans[0].Pos.X
	prevEnd := spans[0].Pos.X + spans[0].Pos.Width

	flush := func() {
		if len(buf) == 0 {
			return
		}
		text := normalizeSpaces(joinSpans(buf))
		if text != "" {
			cells = append(cells, tableCell{startX: startX, text: text})
		}
		buf = nil
	}

	buf = append(buf, spans[0])
	for _, sp := range spans[1:] {
		gap := sp.Pos.X - prevEnd
		if gap > threshold {
			flush()
			startX = sp.Pos.X
			buf = append(buf, sp)
		} else {
			buf = append(buf, sp)
		}
		end := sp.Pos.X + sp.Pos.Width
		if end > prevEnd {
			prevEnd = end
		}
	}
	flush()
	return cells
}

func lineToRow(cols []float64, spans []TextSpan, gap, fontSize float64) (bool, []string) {
	if len(cols) == 0 || len(spans) == 0 {
		return false, nil
	}
	cells := lineCells(spans, fontSize)
	if len(cells) == 0 {
		return false, nil
	}

	row := make([]string, len(cols))
	var sumDist float64
	for _, cell := range cells {
		idx := nearest(cols, cell.startX)
		dist := math.Abs(cell.startX - cols[idx])
		sumDist += dist
		if row[idx] == "" {
			row[idx] = cell.text
		} else {
			row[idx] = strings.TrimSpace(row[idx] + " " + cell.text)
		}
	}

	meanDist := sumDist / float64(len(cells))
	if gap > 0 && meanDist > gap*0.9 {
		return false, nil
	}

	if countNonEmpty(row) == 0 {
		return false, nil
	}

	return true, row
}

func rowBreak(prevY, y, prevSize, size float64) bool {
	gap := prevY - y
	if gap <= 0 {
		return false
	}
	threshold := math.Max(prevSize, size) * 2.0
	if threshold < 10 {
		threshold = 10
	}
	return gap > threshold
}

func countNonEmpty(row []string) int {
	count := 0
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			count++
		}
	}
	return count
}

func rowLooksLikeHeader(row []string) bool {
	if len(row) == 0 {
		return false
	}
	if looksLikeSKU(row[0]) {
		return false
	}
	text := strings.ToLower(strings.Join(row, " "))
	hits := 0
	for _, keyword := range []string{"sku", "description", "unit", "price", "measure", "notes", "usd"} {
		if strings.Contains(text, keyword) {
			hits++
		}
	}
	return hits >= 2
}

func looksLikeSKU(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 24 {
		return false
	}
	hasLetter := false
	hasDigit := false
	hasDash := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		case r == '-' || r == '–' || r == '—':
			hasDash = true
		default:
			return false
		}
	}
	return hasLetter && hasDigit && hasDash
}

func hasHeaderKeyword(s string) bool {
	lower := strings.ToLower(s)
	for _, keyword := range []string{"sku", "description", "unit", "price", "measure", "notes"} {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func lineHasSKU(spans []TextSpan) bool {
	if len(spans) == 0 {
		return false
	}
	return looksLikeSKU(spans[0].Text)
}

func clusteredStarts(values []float64, tol float64) []float64 {
	if len(values) == 0 {
		return values
	}
	sort.Float64s(values)
	clustered := []float64{values[0]}
	for _, x := range values[1:] {
		last := clustered[len(clustered)-1]
		if math.Abs(x-last) <= tol {
			clustered[len(clustered)-1] = (last + x) / 2
		} else {
			clustered = append(clustered, x)
		}
	}
	return clustered
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
