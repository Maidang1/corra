package render

import (
	"html"
	"strings"

	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/Maidang1/corra/internal/mdext"
)

// GitHub renders a blockquote whose first line is [!NOTE] (or TIP, IMPORTANT,
// WARNING, CAUTION) as a callout. The HTML export draws it as a tinted card
// with the label as its title, which needs two things the default renderer
// will not do: the marker must not be printed, and the blockquote element must
// become a div. Both happen here, before the tree is rendered.

// AlertTitle prints a callout label the way GitHub spells it.
func AlertTitle(label string) string {
	if label == "" {
		return ""
	}
	return strings.ToUpper(label[:1]) + strings.ToLower(label[1:])
}

// alertClass is the CSS class suffix of a callout label.
func alertClass(label string) string {
	return strings.ToLower(label)
}

// alertLabelOf reports the callout label of a blockquote, or "".
func alertLabelOf(quote *gast.Blockquote, source []byte) string {
	first, ok := quote.FirstChild().(*gast.Paragraph)
	if !ok {
		return ""
	}
	return mdext.AlertLabel(firstLineOf(paragraphText(first, source)))
}

// transformAlerts strips the "[!LABEL]" marker from every callout blockquote so
// it is not printed next to the title the renderer adds, and records the label
// of each one in labels. The record is what the renderer reads, because
// stripping can remove the paragraph the label was read from.
func transformAlerts(doc gast.Node, source []byte, labels map[gast.Node]string) {
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		quote, ok := n.(*gast.Blockquote)
		if !entering || !ok {
			return gast.WalkContinue, nil
		}
		label := alertLabelOf(quote, source)
		if label == "" {
			return gast.WalkContinue, nil
		}
		labels[quote] = label
		first, _ := quote.FirstChild().(*gast.Paragraph)
		if stripAlertMarker(first, source) && first != nil && first.Parent() != nil {
			// "> [!NOTE]\n>\n> body" leaves an empty paragraph where the
			// marker was; drop it rather than print an empty <p>.
			quote.RemoveChild(quote, first)
		}
		return gast.WalkContinue, nil
	})
}

// stripAlertMarker blanks the text nodes holding the "[!LABEL]" at the start of
// a paragraph, plus the whitespace that separated it from the body. It reports
// whether the paragraph is left with no text at all.
func stripAlertMarker(p *gast.Paragraph, source []byte) bool {
	if p == nil {
		return false
	}
	label, _ := mdext.AlertLabelPrefix(paragraphText(p, source))
	if label == "" {
		return false
	}
	left := len("[!" + label + "]")
	for c := p.FirstChild(); c != nil; {
		next := c.NextSibling()
		t, ok := c.(*gast.Text)
		if !ok {
			return !paragraphHasText(p, source)
		}
		value := t.Segment.Value(source)
		if len(value) > left {
			// The marker ends inside this node, so the separator that followed
			// it is inside this node too: trim it off the front of what is left.
			t.Segment = text.NewSegment(t.Segment.Start+left, t.Segment.Stop)
			trimSegmentStart(t, source)
			return false
		}
		left -= len(value)
		t.Segment = text.NewSegment(t.Segment.Start, t.Segment.Start)
		// A blanked node that still claims a soft break would print a bare "\n".
		t.SetSoftLineBreak(false)
		t.SetHardLineBreak(false)
		if left == 0 {
			skipAlertSpace(p, c, source)
			return !paragraphHasText(p, source)
		}
		c = next
	}
	return !paragraphHasText(p, source)
}

// trimSegmentStart advances a text node past the spaces and tabs it starts with.
func trimSegmentStart(t *gast.Text, source []byte) {
	value := t.Segment.Value(source)
	trimmed := 0
	for trimmed < len(value) && (value[trimmed] == ' ' || value[trimmed] == '\t') {
		trimmed++
	}
	if trimmed > 0 {
		t.Segment = text.NewSegment(t.Segment.Start+trimmed, t.Segment.Stop)
	}
}

// skipAlertSpace drops the whitespace that followed the marker.
func skipAlertSpace(p *gast.Paragraph, after gast.Node, source []byte) {
	for c := after.NextSibling(); c != nil; {
		next := c.NextSibling()
		t, ok := c.(*gast.Text)
		if !ok {
			return
		}
		value := t.Segment.Value(source)
		trimmed := 0
		for trimmed < len(value) && (value[trimmed] == ' ' || value[trimmed] == '\t') {
			trimmed++
		}
		if trimmed == 0 {
			return
		}
		if trimmed == len(value) {
			t.Segment = text.NewSegment(t.Segment.Start, t.Segment.Start)
			c = next
			continue
		}
		t.Segment = text.NewSegment(t.Segment.Start+trimmed, t.Segment.Stop)
		return
	}
}

// paragraphHasText reports whether any text node of a paragraph still carries
// characters, which is how an emptied marker paragraph is recognised.
func paragraphHasText(p *gast.Paragraph, source []byte) bool {
	if p == nil {
		return false
	}
	for c := p.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*gast.Text); ok && len(t.Segment.Value(source)) > 0 {
			return true
		}
	}
	return false
}

func paragraphText(p *gast.Paragraph, source []byte) string {
	if p == nil {
		return ""
	}
	var out strings.Builder
	lines := p.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		out.Write(seg.Value(source))
	}
	return out.String()
}

func firstLineOf(source string) string {
	if i := strings.IndexByte(source, '\n'); i >= 0 {
		return source[:i]
	}
	return source
}

// blockquoteHTMLRenderer draws a GitHub callout as a tinted card and leaves
// ordinary quotes alone.
type blockquoteHTMLRenderer struct{ labels map[gast.Node]string }

func (r *blockquoteHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(gast.KindBlockquote, r.render)
}

func (r *blockquoteHTMLRenderer) render(w util.BufWriter, source []byte, node gast.Node, entering bool) (gast.WalkStatus, error) {
	label := r.labels[node]
	if entering {
		if label == "" {
			_, err := w.WriteString("<blockquote>\n")
			return gast.WalkContinue, err
		}
		_, err := w.WriteString(`<div class="alert alert-` + alertClass(label) + `">` + "\n" +
			`<p class="alert-title">` + html.EscapeString(AlertTitle(label)) + "</p>\n")
		return gast.WalkContinue, err
	}
	if label == "" {
		_, err := w.WriteString("</blockquote>\n")
		return gast.WalkContinue, err
	}
	_, err := w.WriteString("</div>\n")
	return gast.WalkContinue, err
}
