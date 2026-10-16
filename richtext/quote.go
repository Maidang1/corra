package richtext

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// Quote paragraphs retain their own source spans, including the marker-only
// lines between nested blocks. Depth is zero for the outermost quote.
func quoteParagraphs(quote *ast.Blockquote, src []byte, raw string) []Paragraph {
	type part struct {
		node         ast.Node
		start, depth int
	}
	var parts []part
	var walk func(*ast.Blockquote, int) bool
	walk = func(q *ast.Blockquote, depth int) bool {
		for child := q.FirstChild(); child != nil; child = child.NextSibling() {
			if nested, ok := child.(*ast.Blockquote); ok {
				if !walk(nested, depth+1) {
					return false
				}
				continue
			}
			if child.Kind() != ast.KindParagraph {
				return false
			}
			parts = append(parts, part{child, lineStart(string(src), child.Pos()), depth})
		}
		return true
	}
	if !walk(quote, 0) || len(parts) == 0 {
		return sourceLines(raw)
	}
	base := lineStart(string(src), quote.Pos())
	out := make([]Paragraph, 0, len(parts))
	for i, part := range parts {
		end := base + len(raw)
		if i+1 < len(parts) {
			end = parts[i+1].start
		}
		if part.start < base || end < part.start || end > base+len(raw) {
			return sourceLines(raw)
		}
		fragment := string(src[part.start:end])
		content, separator := splitQuoteSeparator(fragment)
		p := Paragraph{Kind: "quote", Depth: part.depth, Raw: content, Separator: separator, Runs: []Run{}}
		if !collectRuns(part.node, src, Run{}, &p) {
			return sourceLines(raw)
		}
		out = append(out, p)
	}
	return applyAlerts(out)
}

func splitQuoteSeparator(source string) (string, string) {
	lines := strings.SplitAfter(source, "\n")
	end := len(source)
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if strings.Trim(line, " >\t\r\n") == "" {
			end -= len(line)
			continue
		}
		contentEnd := end - len(line) + len(strings.TrimRight(line, "\r\n"))
		return source[:contentEnd], source[contentEnd:]
	}
	return source, ""
}
