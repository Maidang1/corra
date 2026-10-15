// HTML export: a standalone HTML page with embedded styles, KaTeX
// auto-render for math and mermaid.js for mermaid diagrams (both from CDN).
package render

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	ghtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/Maidang1/corra/internal/mdext"
)

// ExportHTML renders the document source to a standalone HTML page.
// Math stays in $...$ form with a KaTeX auto-render script; mermaid blocks
// become <pre class="mermaid">. smartPunct applies typographic punctuation.
func ExportHTML(src, title string, smartPunct bool) ([]byte, error) {
	exts := []goldmark.Extender{
		extension.GFM,
		extension.Footnote,
		mdext.InlineExtender,
		emoji.New(),
	}
	if smartPunct {
		exts = append(exts, extension.Typographer)
	}
	md := goldmark.New(
		goldmark.WithExtensions(exts...),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(
			ghtml.WithUnsafe(), // the document is the user's own
			renderer.WithNodeRenderers(
				util.Prioritized(&mdHTMLRenderer{}, 500),
			),
		),
	)
	fm, body := mdext.SplitFrontMatter(src)
	body = rewriteMermaid(body)
	if toc := tocHTML(md, body); toc != "" {
		body = replaceTOC(body, toc)
	}
	var buf bytes.Buffer
	if err := md.Convert([]byte(body), &buf); err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\">\n")
	out.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">` + "\n")
	fmt.Fprintf(&out, "<title>%s</title>\n", html.EscapeString(title))
	if fm != "" {
		fmt.Fprintf(&out, "<!--\n%s\n-->\n", strings.TrimRight(fm, "\n"))
	}
	out.WriteString(htmlStyle)
	out.WriteString(`<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.css">` + "\n")
	out.WriteString("</head><body>\n<main class=\"markdown-body\">\n")
	out.WriteString(buf.String())
	out.WriteString("\n</main>\n")
	out.WriteString(`<script defer src="https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.js"></script>` + "\n")
	out.WriteString(`<script defer src="https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/contrib/auto-render.min.js" onload="renderMathInElement(document.body,{delimiters:[{left:'$$',right:'$$',display:true},{left:'$',right:'$',display:false}]});"></script>` + "\n")
	out.WriteString(`<script type="module">import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs"; mermaid.initialize({startOnLoad:true});</script>` + "\n")
	out.WriteString("</body>\n</html>\n")
	return []byte(out.String()), nil
}

// rewriteMermaid turns ```mermaid fences into <pre class="mermaid"> blocks.
func rewriteMermaid(src string) string {
	var out strings.Builder
	in := false
	for _, line := range strings.SplitAfter(src, "\n") {
		trim := strings.TrimRight(line, "\r\n")
		if !in && strings.EqualFold(strings.TrimSpace(trim), "```mermaid") {
			in = true
			out.WriteString(`<pre class="mermaid">` + "\n")
			continue
		}
		if in && strings.TrimSpace(trim) == "```" {
			in = false
			out.WriteString("</pre>\n")
			continue
		}
		if in {
			out.WriteString(html.EscapeString(strings.TrimRight(line, "\n")))
			out.WriteByte('\n')
			continue
		}
		out.WriteString(line)
	}
	if in {
		out.WriteString("</pre>\n")
	}
	return out.String()
}

// tocHTML builds a nested list of links to the document's headings, or ""
// when there are no headings. Anchor IDs follow goldmark's AutoHeadingID.
func tocHTML(md goldmark.Markdown, body string) string {
	if !hasTOC(body) {
		return ""
	}
	parsed := md.Parser().Parse(text.NewReader([]byte(body)))
	var out strings.Builder
	depth := 0
	open := false
	for n := parsed.FirstChild(); n != nil; n = n.NextSibling() {
		h, ok := n.(*gast.Heading)
		if !ok {
			continue
		}
		level := h.Level
		if level < 1 {
			level = 1
		}
		var text_ string
		for c := h.FirstChild(); c != nil; c = c.NextSibling() {
			text_ += string(c.Text([]byte(body)))
		}
		id := headingAnchor(text_)
		for depth < level {
			out.WriteString("<ul>")
			depth++
		}
		for depth > level {
			out.WriteString("</ul>")
			depth--
		}
		fmt.Fprintf(&out, `<li><a href="#%s">%s</a></li>`, id, html.EscapeString(text_))
		open = true
	}
	for depth > 0 {
		out.WriteString("</ul>")
		depth--
	}
	if !open {
		return ""
	}
	return `<nav class="toc"><strong>目录</strong>` + out.String() + "</nav>"
}

func hasTOC(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		if mdext.IsTOC(line) {
			return true
		}
	}
	return false
}

// replaceTOC swaps the [TOC] paragraph for the rendered list.
func replaceTOC(body, toc string) string {
	var out strings.Builder
	for _, line := range strings.SplitAfter(body, "\n") {
		if mdext.IsTOC(strings.TrimRight(line, "\r\n")) {
			out.WriteString(toc + "\n")
			continue
		}
		out.WriteString(line)
	}
	return out.String()
}

// headingAnchor matches goldmark's AutoHeadingID generation.
func headingAnchor(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var out []rune
	lastDash := false
	for _, r := range s {
		switch {
		case r == ' ' || r == '\t':
			if !lastDash {
				out = append(out, '-')
				lastDash = true
			}
		case util.IsPunctRune(r):
			// dropped
		default:
			out = append(out, r)
			lastDash = false
		}
	}
	return string(out)
}

const htmlStyle = `<style>
body{margin:0;padding:0 16px;font:16px/1.6 -apple-system,"Helvetica Neue","PingFang SC","Microsoft YaHei",sans-serif;color:#24292f}
main{max-width:760px;margin:32px auto}
h1,h2{border-bottom:1px solid #eaecef;padding-bottom:.2em}h1{font-size:1.7em}h2{font-size:1.4em}
code,kbd{font:0.9em ui-monospace,"SF Mono",Menlo,Consolas,monospace;background:#f3f4f6;border-radius:4px;padding:.1em .3em}
pre{background:#f6f8fa;border-radius:8px;padding:12px;overflow:auto}pre code{background:none;padding:0}
blockquote{margin:0;padding:0 1em;color:#57606a;border-left:3px solid #d0d7de}
table{border-collapse:collapse}td,th{border:1px solid #d0d7de;padding:4px 10px}
img{max-width:100%%}
mark{background:#fff1a8}del{color:#57606a}
nav.toc{border:1px solid #eaecef;border-radius:8px;padding:12px 16px;font-size:.92em;margin-bottom:1em}
nav.toc ul{margin:.2em 0;padding-left:1.4em}nav.toc a{color:#0969da;text-decoration:none}
</style>
`

// mdHTMLRenderer renders mdext inline nodes to HTML.
type mdHTMLRenderer struct{ ghtml.Config }

// RegisterFuncs implements renderer.NodeRenderer.
func (r *mdHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(mdext.KindMark, wrapHTML("<mark>", "</mark>"))
	reg.Register(mdext.KindUnderline, wrapHTML("<u>", "</u>"))
	reg.Register(mdext.KindSuperscript, wrapHTML("<sup>", "</sup>"))
	reg.Register(mdext.KindSubscript, wrapHTML("<sub>", "</sub>"))
	reg.Register(mdext.KindKbd, wrapHTML("<kbd>", "</kbd>"))
}

func wrapHTML(open, close string) renderer.NodeRendererFunc {
	return func(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString(open)
		} else {
			_, _ = w.WriteString(close)
		}
		return gast.WalkContinue, nil
	}
}
