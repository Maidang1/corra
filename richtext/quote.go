package richtext

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// Quote paragraphs retain their own source spans, including the marker-only
// lines between nested blocks. Depth is zero for the outermost quote.
func quoteParagraphs(quote *ast.Blockquote, src []byte, raw string) []Paragraph {
	type part struct {
		node         ast.Node
		start, depth int
		// List items inside the quote (Typora "> - item").
		list                                  bool
		kind, listID, container, marker, item string
		ordinal, listDepth                    int
		continuation                          bool
	}
	var parts []part
	var flatten func(*ast.List, int, int, string) bool
	flatten = func(list *ast.List, quote, depth int, id string) bool {
		ordinal := list.Start
		for item := list.FirstChild(); item != nil; item = item.NextSibling() {
			kind := "bullet"
			if list.IsOrdered() {
				kind = "ordered"
			}
			if isTask(item) {
				kind = "checklist"
			}
			first := true
			if _, nestedFirst := item.FirstChild().(*ast.List); nestedFirst {
				// "> - " holding only a nested list: its own line comes first.
				if item.Pos() < 0 {
					return false
				}
				parts = append(parts, part{node: emptyItem{item}, start: lineStart(string(src), item.Pos()), depth: quote, list: true, kind: kind, listID: id, container: fmt.Sprintf("list%d", list.Pos()), marker: string(list.Marker), item: fmt.Sprintf("item%d", item.Pos()), ordinal: ordinal, listDepth: depth})
				first = false
			}
			for child := item.FirstChild(); child != nil; child = child.NextSibling() {
				if nested, ok := child.(*ast.List); ok {
					if !flatten(nested, quote, depth+1, id) {
						return false
					}
					continue
				}
				if child.Kind() != ast.KindParagraph && child.Kind() != ast.KindTextBlock {
					return false
				}
				parts = append(parts, part{node: child, start: lineStart(string(src), child.Pos()), depth: quote, list: true, kind: kind, listID: id, container: fmt.Sprintf("list%d", list.Pos()), marker: string(list.Marker), item: fmt.Sprintf("item%d", item.Pos()), ordinal: ordinal, listDepth: depth, continuation: !first})
				first = false
			}
			if first {
				// An empty item ("> - ") is an empty line of the list.
				if item.Pos() < 0 {
					return false
				}
				parts = append(parts, part{node: item, start: lineStart(string(src), item.Pos()), depth: quote, list: true, kind: kind, listID: id, container: fmt.Sprintf("list%d", list.Pos()), marker: string(list.Marker), item: fmt.Sprintf("item%d", item.Pos()), ordinal: ordinal, listDepth: depth})
			}
			ordinal++
		}
		return true
	}
	var walk func(*ast.Blockquote, int) bool
	walk = func(q *ast.Blockquote, depth int) bool {
		for child := q.FirstChild(); child != nil; child = child.NextSibling() {
			if nested, ok := child.(*ast.Blockquote); ok {
				if !walk(nested, depth+1) {
					return false
				}
				continue
			}
			if list, ok := child.(*ast.List); ok {
				if !flatten(list, depth+1, 0, fmt.Sprintf("list%d", list.Pos())) {
					return false
				}
				continue
			}
			if child.Kind() != ast.KindParagraph {
				return false
			}
			parts = append(parts, part{node: child, start: lineStart(string(src), child.Pos()), depth: depth})
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
		if part.list {
			p.Kind, p.Depth, p.Quote = part.kind, part.listDepth, part.depth
			if p.Checked || part.kind == "checklist" {
				p.Kind = "checklist"
			}
			if part.marker == "." || part.marker == ")" {
				p.Ordinal = part.ordinal
			}
			p.ListID, p.ListContainer, p.ListMarker = part.listID, part.container, part.marker
			p.ListItem, p.ListContinuation = part.item, part.continuation
			inner := stripQuote(content, part.depth)
			p.ListIndent = inner[:len(inner)-len(strings.TrimLeft(inner, " \t"))]
			p.ListBaseline = listFingerprint(p)
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
