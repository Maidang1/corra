// HTMLToMarkdown converts clipboard HTML (as browsers put on the pasteboard)
// into Markdown, giving Typora-style smart paste: copying from a web page or
// a rich document pastes structured Markdown instead of raw markup.
package nativerich

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// HTMLToMarkdown converts an HTML fragment to Markdown. It returns an error
// only when the input is not parseable at all.
func HTMLToMarkdown(src string) (string, error) {
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return "", err
	}
	var body *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if body == nil {
				walk(c)
			}
		}
	}
	walk(doc)
	if body == nil {
		body = doc
	}
	var out strings.Builder
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		writeBlock(&out, c)
	}
	return strings.Trim(out.String(), "\n"), nil
}

// writeBlock appends a block-level node, keeping a blank line between blocks.
func writeBlock(out *strings.Builder, n *html.Node) {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			paragraph(out, text)
		}
		return
	}
	if n.Type != html.ElementNode {
		return
	}
	switch strings.ToLower(n.Data) {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		level := int(n.Data[1] - '0')
		paragraph(out, strings.Repeat("#", level)+" "+inline(n))
	case "p":
		paragraph(out, inline(n))
	case "br":
		// inline-level; ignore at block level
	case "hr":
		paragraph(out, "---")
	case "blockquote":
		var inner strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeBlock(&inner, c)
		}
		for _, line := range strings.Split(strings.Trim(inner.String(), "\n"), "\n") {
			out.WriteString("> " + line + "\n")
		}
		out.WriteByte('\n')
	case "pre":
		code := textOf(n)
		lang := ""
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "code" {
				code = textOf(c)
				lang = attr(c, "class")
				lang = strings.TrimPrefix(strings.TrimPrefix(lang, "language-"), "lang-")
				if i := strings.IndexByte(lang, ' '); i >= 0 {
					lang = lang[:i]
				}
			}
		}
		paragraph(out, "```"+lang+"\n"+strings.TrimSuffix(code, "\n")+"\n```")
	case "ul", "ol":
		writeList(out, n, 0)
		out.WriteByte('\n')
	case "table":
		writeTable(out, n)
	case "div", "section", "article", "figure", "main", "header", "footer":
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeBlock(out, c)
		}
	case "img":
		if md := imgMarkdown(n); md != "" {
			paragraph(out, md)
		}
	default:
		// Unrecognized block: keep its inline content.
		if text := inline(n); strings.TrimSpace(text) != "" {
			paragraph(out, text)
		}
	}
}

func paragraph(out *strings.Builder, text string) {
	out.WriteString(strings.TrimRight(text, "\n"))
	out.WriteString("\n\n")
}

// inline renders a node's children as inline Markdown.
func inline(n *html.Node) string {
	var out strings.Builder
	writeInline(&out, n)
	return strings.TrimSpace(out.String())
}

func writeInline(out *strings.Builder, n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode:
			out.WriteString(escapeInline(c.Data))
		case c.Type != html.ElementNode:
			// comments etc. dropped
		default:
			tag := strings.ToLower(c.Data)
			inner := inline(c)
			switch tag {
			case "strong", "b":
				out.WriteString("**" + inner + "**")
			case "em", "i":
				out.WriteString("*" + inner + "*")
			case "del", "s", "strike":
				out.WriteString("~~" + inner + "~~")
			case "u", "ins":
				out.WriteString("<u>" + inner + "</u>")
			case "mark":
				out.WriteString("==" + inner + "==")
			case "sub":
				out.WriteString("~" + inner + "~")
			case "sup":
				out.WriteString("^" + inner + "^")
			case "code":
				out.WriteString("`" + strings.ReplaceAll(textOf(c), "`", "\\`") + "`")
			case "a":
				href := attr(c, "href")
				if href == "" || href == inner {
					out.WriteString(inner)
				} else {
					out.WriteString("[" + inner + "](" + href + ")")
				}
			case "img":
				out.WriteString(imgMarkdown(c))
			case "br":
				out.WriteString("  \n")
			case "span", "font", "small", "abbr":
				out.WriteString(inner)
			default:
				// Blocks nested inline: recurse as blocks.
				var blockOut strings.Builder
				writeBlock(&blockOut, c)
				text := strings.Trim(blockOut.String(), "\n")
				if text != "" {
					out.WriteString(text)
				}
			}
		}
	}
}

func writeList(out *strings.Builder, list *html.Node, depth int) {
	index := 1
	for item := list.FirstChild; item != nil; item = item.NextSibling {
		if item.Type != html.ElementNode || strings.ToLower(item.Data) != "li" {
			continue
		}
		indent := strings.Repeat("  ", depth)
		marker := "- "
		if list.Data == "ol" {
			marker = fmt.Sprintf("%d. ", index)
			index++
		}
		// Task list items carry a checkbox input.
		task := ""
		var content *strings.Builder
		content = &strings.Builder{}
		for c := item.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && strings.ToLower(c.Data) == "input" &&
				strings.EqualFold(attr(c, "type"), "checkbox") {
				box := " "
				if hasAttr(c, "checked") {
					box = "x"
				}
				task = "[" + box + "] "
				continue
			}
			if c.Type == html.ElementNode && (strings.ToLower(c.Data) == "ul" || strings.ToLower(c.Data) == "ol") {
				continue // nested lists handled below
			}
			if c.Type == html.TextNode {
				content.WriteString(escapeInline(c.Data))
			} else {
				content.WriteString(inline(c))
			}
		}
		out.WriteString(indent + marker + task + strings.TrimSpace(content.String()) + "\n")
		for c := item.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && (strings.ToLower(c.Data) == "ul" || strings.ToLower(c.Data) == "ol") {
				writeList(out, c, depth+1)
			}
		}
	}
}

// writeTable renders a simple HTML table as a GFM pipe table.
func writeTable(out *strings.Builder, table *html.Node) {
	var rows [][]string
	var collect func(n *html.Node, inRow bool)
	collect = func(n *html.Node, inRow bool) {
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "tr":
				var cells []string
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && (strings.ToLower(c.Data) == "td" || strings.ToLower(c.Data) == "th") {
						cells = append(cells, strings.ReplaceAll(inline(c), "|", "\\|"))
					}
				}
				if len(cells) > 0 {
					rows = append(rows, cells)
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collect(c, inRow)
		}
	}
	collect(table, false)
	if len(rows) == 0 {
		return
	}
	cols := 0
	for _, r := range rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	line := func(cells []string) {
		out.WriteString("|")
		for i := 0; i < cols; i++ {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			out.WriteString(" " + cell + " |")
		}
		out.WriteByte('\n')
	}
	line(rows[0])
	out.WriteString("|" + strings.Repeat(" --- |", cols) + "\n")
	for _, r := range rows[1:] {
		line(r)
	}
	out.WriteByte('\n')
}

func imgMarkdown(n *html.Node) string {
	src := attr(n, "src")
	if src == "" || strings.HasPrefix(src, "data:") {
		return ""
	}
	alt := attr(n, "alt")
	size := ""
	if w := attr(n, "width"); w != "" {
		if h := attr(n, "height"); h != "" {
			size = " =" + w + "x" + h
		} else {
			size = " =" + w + "x"
		}
	}
	if size != "" {
		return "![" + alt + "](<" + src + size + ">)"
	}
	return "![" + alt + "](" + src + ")"
}

func escapeInline(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return s
}

func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, name string) bool {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return true
		}
	}
	return false
}

func textOf(n *html.Node) string {
	var out strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			out.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out.String()
}
