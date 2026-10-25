// HTML export embeds document math as self-contained SVG images.
package render

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"image/color"
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
// Mermaid blocks become <pre class="mermaid">. Math needs no runtime scripts.
func ExportHTML(src, title string, smartPunct bool) ([]byte, error) {
	return ExportHTMLOpts(context.Background(), src, title, smartPunct, MathOptions{})
}

func ExportHTMLOpts(ctx context.Context, src, title string, smartPunct bool, mathOptions MathOptions) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	mathRenderer := &mathHTMLRenderer{results: make(map[gast.Node]MathResult)}
	alertLabels := map[gast.Node]string{}
	exts := []goldmark.Extender{
		extension.GFM,
		extension.Footnote,
		MathExtender,
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
				util.Prioritized(&blockquoteHTMLRenderer{labels: alertLabels}, 500),
				util.Prioritized(mathRenderer, 500),
			),
		),
	)
	fm, body := mdext.SplitFrontMatter(src)
	body = rewriteMermaid(body)
	if toc := tocHTML(md, body); toc != "" {
		body = replaceTOC(body, toc)
	}
	var buf bytes.Buffer
	parsed := md.Parser().Parse(text.NewReader([]byte(body)))
	mdext.TransformHTML(parsed, []byte(body))
	mdext.TransformSizedImages(parsed, []byte(body))
	transformAlerts(parsed, []byte(body), alertLabels)
	plan := collectMath(parsed)
	if len(plan.Expressions) > 0 {
		results, err := RenderMathBatchContext(ctx, plan.Expressions, mathOptions, 16, color.Black)
		if err != nil {
			return nil, err
		}
		for node, index := range plan.Nodes {
			mathRenderer.results[node] = results[index]
		}
	}
	if err := md.Renderer().Render(&buf, []byte(body), parsed); err != nil {
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
	out.WriteString("</head><body>\n<main class=\"markdown-body\">\n")
	out.WriteString(buf.String())
	out.WriteString("\n</main>\n")
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
.alert{margin:1em 0;padding:.6em 1em;border-left:4px solid;border-radius:6px}
.alert .alert-title{font-weight:600;margin:0 0 .3em}
.alert-note{background:#ddf4ff;border-color:#54aeff}.alert-note .alert-title{color:#0a3069}
.alert-tip{background:#dafbe1;border-color:#4ac26b}.alert-tip .alert-title{color:#0f5323}
.alert-important{background:#fbefff;border-color:#b062f5}.alert-important .alert-title{color:#512a97}
.alert-warning{background:#fff8c5;border-color:#d4a72c}.alert-warning .alert-title{color:#7d4e00}
.alert-caution{background:#ffebe9;border-color:#ff8182}.alert-caution .alert-title{color:#a40e26}
table{border-collapse:collapse}td,th{border:1px solid #d0d7de;padding:4px 10px}
img{max-width:100%}
.math-display{display:block;text-align:center;margin:1em 0;overflow-x:auto}.math-inline{vertical-align:middle}.math-error{white-space:pre-wrap;color:#a12d24}
mark{background:#fff1a8}del{color:#57606a}
nav.toc{border:1px solid #eaecef;border-radius:8px;padding:12px 16px;font-size:.92em;margin-bottom:1em}
nav.toc ul{margin:.2em 0;padding-left:1.4em}nav.toc a{color:#0969da;text-decoration:none}
</style>
`

type mathHTMLRenderer struct{ results map[gast.Node]MathResult }

func (r *mathHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindMathInlineNode, r.render)
	reg.Register(KindMathBlockNode, r.render)
}

func (r *mathHTMLRenderer) render(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	if !entering {
		return gast.WalkContinue, nil
	}
	tex, display := "", false
	raw := ""
	container := "div"
	switch n := node.(type) {
	case *mathInlineNode:
		tex, display, raw = n.TeX, n.Display, n.Raw
		container = "span"
	case *mathBlockNode:
		tex, display = n.TeX, true
		raw = blockSource(node, source)
	}
	result, ok := r.results[node]
	if !ok || result.Error != nil {
		message := "Invalid math"
		if result.Error != nil {
			message = result.Error.Error()
		}
		_, err := fmt.Fprintf(w, `<code class="math-error" title="%s">%s</code>`, html.EscapeString(message), html.EscapeString(raw))
		return gast.WalkSkipChildren, err
	}
	if display {
		_, _ = fmt.Fprintf(w, `<%s class="math-display">`, container)
	}
	class := "math-inline"
	if display {
		class = "math-block"
	}
	_, err := fmt.Fprintf(w, `<img class="%s" role="math" alt="%s" data-equation-number="%s" width="%.2f" height="%.2f" src="data:image/svg+xml;base64,%s">`, class, html.EscapeString(tex), html.EscapeString(result.Number), float64(result.Image.Bounds().Dx())/2, float64(result.Image.Bounds().Dy())/2, base64.StdEncoding.EncodeToString(result.SVG))
	if display {
		_, _ = fmt.Fprintf(w, "</%s>", container)
	}
	return gast.WalkSkipChildren, err
}

// mdHTMLRenderer renders mdext inline nodes to HTML.
type mdHTMLRenderer struct{ ghtml.Config }

// RegisterFuncs implements renderer.NodeRenderer.
func (r *mdHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(mdext.KindMark, wrapHTML("<mark>", "</mark>"))
	reg.Register(mdext.KindUnderline, wrapHTML("<u>", "</u>"))
	reg.Register(mdext.KindSuperscript, wrapHTML("<sup>", "</sup>"))
	reg.Register(mdext.KindSubscript, wrapHTML("<sub>", "</sub>"))
	reg.Register(mdext.KindKbd, wrapHTML("<kbd>", "</kbd>"))
	reg.Register(mdext.KindSpan, func(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering {
			span := n.(*mdext.Span)
			_, _ = w.WriteString(mdext.SpanTag(span.Color, span.Background))
		} else {
			_, _ = w.WriteString("</span>")
		}
		return gast.WalkContinue, nil
	})
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
