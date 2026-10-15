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
func listSources(doc Document) map[int]string {
	out := make(map[int]string)
	for start := 0; start < len(doc.Paragraphs); {
		if !isList(doc.Paragraphs[start]) {
			start++
			continue
		}
		end, structural := start, false
		for end < len(doc.Paragraphs) && isList(doc.Paragraphs[end]) {
			p := doc.Paragraphs[end]
			if p.ListBaseline == "" || p.ListBaseline != listFingerprint(p) {
				structural = true
			}
			if end > start && p.ID != doc.Paragraphs[end-1].NextID {
				structural = true
			}
			if p.Baseline != fingerprint(p) && originalMarkerWidth(p.Raw) != markerWidth(p) {
				structural = true
			}
			end++
		}
		// Each entry is the content column available to the next depth.
		columns := []int{0}
		for i := start; i < end; i++ {
			p := doc.Paragraphs[i]
			depth := max(0, p.Depth)
			// A malformed native depth jump is clamped to the available parent.
			depth = min(depth, len(columns)-1)
			indent := p.ListIndent
			if structural {
				indent = strings.Repeat(" ", columns[depth])
				if p.ListContinuation {
					indent = strings.Repeat(" ", columns[min(depth+1, len(columns)-1)])
				}
			}
			line := p.Raw
			if p.Baseline == "" || p.Baseline != fingerprint(p) || p.ListBaseline != listFingerprint(p) {
				line = serializeParagraph(p)
			} else {
				line = strings.TrimLeft(line, " \t")
			}
			out[i] = indent + line
			if !p.ListContinuation {
				columns = append(columns[:depth+1], columns[depth]+originalMarkerWidth(line))
			}
		}
		start = end
	}
	return out
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
