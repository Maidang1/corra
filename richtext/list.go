package richtext

import (
	"fmt"
	"strings"
)

func isList(p Paragraph) bool {
	return p.Kind == "bullet" || p.Kind == "ordered" || p.Kind == "checklist"
}

func listFingerprint(p Paragraph) string {
	return fmt.Sprintf("%s:%d:%d:%s:%s:%t", p.Kind, p.Depth, p.Ordinal, p.ListContainer, p.ListMarker, p.ListContinuation)
}

func listMarker(p Paragraph) string {
	ordered := p.Kind == "ordered" || (p.Kind == "checklist" && p.Ordinal > 0)
	if ordered {
		if p.ListMarker == ")" {
			return ")"
		}
		return "."
	}
	if p.ListMarker == "+" || p.ListMarker == "*" {
		return p.ListMarker
	}
	return "-"
}

// listSources preserves source indentation for text edits. A structural edit
// recalculates the whole adjacent tree, because an ordered parent's marker can
// require more columns than a bullet's marker.
func listSources(doc Document) (map[int]string, map[int]int) {
	out := make(map[int]string)
	// itemColumns maps each list paragraph to the content column of the item
	// it belongs to, which is where code inside that item is indented.
	itemColumns := make(map[int]int)
	column := 0
	for start := 0; start < len(doc.Paragraphs); {
		if !isList(doc.Paragraphs[start]) {
			start++
			continue
		}
		end, structural := start, false
		quote := doc.Paragraphs[start].Quote
		// An empty paragraph (an extra blank line) stays inside the list when
		// the item goes on after it.
		inside := func(i int) bool {
			for i < len(doc.Paragraphs) && emptyParagraph(doc.Paragraphs[i]) && doc.Paragraphs[i].Kind == "body" {
				i++
			}
			return i < len(doc.Paragraphs) && doc.Paragraphs[i].Quote == quote && (listCode(doc.Paragraphs[i]) || (isList(doc.Paragraphs[i]) && doc.Paragraphs[i].ListContinuation))
		}
		for end < len(doc.Paragraphs) && (isList(doc.Paragraphs[end]) || listCode(doc.Paragraphs[end]) || (end > start && emptyParagraph(doc.Paragraphs[end]) && doc.Paragraphs[end].Kind == "body" && inside(end))) && doc.Paragraphs[end].Quote == quote {
			p := doc.Paragraphs[end]
			if !isList(p) {
				end++ // code lines inside an item are written by the code writer
				continue
			}
			if p.ListBaseline == "" || p.ListBaseline != listFingerprint(p) {
				structural = true
			}
			if end > start && p.ID != doc.Paragraphs[end-1].NextID {
				structural = true
			}
			if p.Baseline != fingerprint(p) && originalMarkerWidth(stripQuote(p.Raw, quote)) != markerWidth(p) {
				structural = true
			}
			end++
		}
		// Each entry is the content column available to the next depth. The
		// list keeps the indent it was written with (up to three spaces):
		// lines after it may rely on falling outside its items.
		base := 0
		if first := doc.Paragraphs[start]; quote == 0 && len(first.ListIndent) <= 3 && strings.Trim(first.ListIndent, " ") == "" {
			base = len(first.ListIndent)
		}
		columns := []int{base}
		// levels holds the stored depth of each open item. A malformed native
		// depth jump (a first item at depth 1) maps to the nearest open
		// parent, and its siblings map with it.
		var levels []int
		for i := start; i < end; i++ {
			p := doc.Paragraphs[i]
			if !isList(p) {
				continue
			}
			stored := max(0, p.Depth)
			for len(levels) > 0 && (levels[len(levels)-1] > stored || (!p.ListContinuation && levels[len(levels)-1] == stored)) {
				levels = levels[:len(levels)-1]
			}
			depth := len(levels)
			if p.ListContinuation {
				depth = max(0, len(levels)-1)
			} else {
				levels = append(levels, stored)
			}
			depth = min(depth, len(columns)-1)
			indent := p.ListIndent
			if structural {
				indent = strings.Repeat(" ", columns[depth])
				if p.ListContinuation {
					indent = strings.Repeat(" ", columns[min(depth+1, len(columns)-1)])
				}
			}
			line := stripQuote(p.Raw, quote)
			unchanged := !(p.Baseline == "" || p.Baseline != fingerprint(p) || p.ListBaseline != listFingerprint(p))
			if !unchanged {
				line = serializeParagraph(p)
				// Wrapped lines of the item's text align under its content.
				content := len(indent)
				if !p.ListContinuation {
					content += originalMarkerWidth(line)
				}
				line = strings.ReplaceAll(line, "\n", "\n"+strings.Repeat(" ", content))
			} else {
				line = strings.TrimLeft(line, " \t")
			}
			out[i] = indent + line
			if strings.TrimSpace(line) == "" {
				out[i] = "" // an empty item paragraph needs no indentation
			}
			if quote > 0 {
				// Every line of the item sits inside the quote.
				prefix := strings.Repeat("> ", quote)
				if unchanged && !structural {
					out[i] = p.Raw
				} else {
					out[i] = prefix + strings.ReplaceAll(out[i], "\n", "\n"+prefix)
				}
			}
			if !p.ListContinuation {
				columns = append(columns[:depth+1], columns[depth]+originalMarkerWidth(line))
				column = len(indent) + originalMarkerWidth(line)
			} else {
				column = len(indent) // its own item's text column, not a nested item's
			}
			if p.Quote == 0 {
				itemColumns[i] = column
			}
		}
		start = end
	}
	return out, itemColumns
}

func markerWidth(p Paragraph) int {
	if p.Kind == "ordered" || (p.Kind == "checklist" && p.Ordinal > 0) {
		return len(fmt.Sprintf("%d. ", max(1, p.Ordinal)))
	}
	return 2
}

func originalMarkerWidth(raw string) int {
	line := strings.TrimLeft(raw, " \t")
	end := 0
	for end < len(line) && line[end] >= '0' && line[end] <= '9' {
		end++
	}
	markerEnd := 1
	if end > 0 && end < len(line) && (line[end] == '.' || line[end] == ')') {
		markerEnd = end + 1
	}
	column := markerEnd
	for i := markerEnd; i < len(line); i++ {
		if line[i] == ' ' {
			column++
		} else if line[i] == '\t' {
			column += 4 - column%4
		} else {
			break
		}
	}
	return max(markerEnd+1, column)
}

// stripQuote removes quote markers ("> ") from every line of a list item that
// lives inside quote blockquotes.
func stripQuote(raw string, quote int) string {
	if quote == 0 {
		return raw
	}
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		rest := line
		for level := 0; level < quote; level++ {
			trimmed := strings.TrimLeft(rest, " ")
			if !strings.HasPrefix(trimmed, ">") {
				break
			}
			rest = strings.TrimPrefix(trimmed[1:], " ")
		}
		lines[i] = rest
	}
	return strings.Join(lines, "\n")
}

// listCode reports a code line or quote that belongs to a list item.
func listCode(p Paragraph) bool {
	return (p.Kind == "code" || p.Kind == "quote") && p.CodeIndent != ""
}
