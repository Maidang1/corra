package mgdoc

import (
	"image"
	"image/color"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"github.com/Maidang1/corra/render"
)

// Tags carried by runs for syntax Markdown stores but MyGo has no attribute
// for. They are styled through EditorOptions.TagStyle.
const (
	TagMark = "mark"
	TagSup  = "sup"
	TagSub  = "sub"
)

// rawRef marks an attachment that stands for Markdown source kept verbatim.
type rawRef struct {
	raw   string
	block bool
}

// Codec converts between Markdown and MyGo's text, owning the rendered
// previews (math, diagrams, images) its attachments show.
type Codec struct {
	// Dark tints math and picks chip colors; change it with Retint.
	Dark bool
	// BaseDir resolves relative image paths.
	BaseDir string
	// FontSize is the body size in DIPs; previews scale with it.
	FontSize float32
	// LoadImage reads a local image; nil disables image previews.
	LoadImage func(url, baseDir string) ([]byte, error)
	// Invalidate asks for a repaint after background work completes.
	Invalidate func()

	mu   sync.Mutex
	math []*mathItem
}

type mathItem struct {
	tex     string
	display bool
	at      *richtext.Attachment
	bmp     *ui.Bitmap
	err     error
}

func (c *Codec) size() float32 {
	if c.FontSize > 0 {
		return c.FontSize
	}
	return 16
}

// Load reads Markdown into MyGo's text.
func (c *Codec) Load(source string) richtext.Text {
	c.math = nil
	var paras []richtext.Paragraph
	for _, seg := range Split(source) {
		kind, info := classify(seg)
		if kind == blockNone {
			if ps, ok := c.readText(seg); ok {
				paras = append(paras, ps...)
				continue
			}
			kind = blockOther
		}
		paras = append(paras, c.blockParagraph(kind, info, seg))
	}
	c.renderMath()
	return richtext.TextOf(paras...)
}

func (c *Codec) readText(seg string) ([]richtext.Paragraph, bool) {
	tokenized, tokens := tokenize(seg)
	t := richtext.ParseMarkdown(tokenized)
	if !stable(t) {
		return nil, false
	}
	var out []richtext.Paragraph
	var st tagState
	for _, p := range t.Paragraphs() {
		out = append(out, c.expand(p, tokens, &st))
	}
	if len(out) == 0 {
		return nil, false
	}
	// Code languages ride on the code lines as tags, one fence per block.
	infos := fenceInfos(seg)
	group, prev := -1, false
	for i := range out {
		code := out[i].Style.CodeBlock
		if code && !prev {
			group++
		}
		prev = code
		if code && group < len(infos) && infos[group] != "" {
			for j := range out[i].Runs {
				out[i].Runs[j].Attributes.Tag = langTag + infos[group]
			}
		}
	}
	if group+1 != len(infos) {
		return nil, false
	}
	return out, true
}

const langTag = "lang:"

// stable reports whether text survives a write and re-read unchanged, the
// proof that nothing in it is lost by the model.
func stable(t richtext.Text) bool {
	again := richtext.ParseMarkdown(t.Markdown())
	return reflect.DeepEqual(shape(t), shape(again))
}

type runShape struct {
	Text string
	A    richtext.Attributes
	Name string
	URL  string
}

type paraShape struct {
	Style richtext.ParagraphStyle
	Runs  []runShape
}

func shape(t richtext.Text) []paraShape {
	var out []paraShape
	for _, p := range t.Paragraphs() {
		ps := paraShape{Style: p.Style}
		for _, r := range p.Runs {
			rs := runShape{Text: r.Text, A: r.Attributes}
			if at := r.Attributes.Attachment; at != nil {
				rs.Name, rs.URL = at.Name, at.URL
				rs.A.Attachment = nil
			}
			ps.Runs = append(ps.Runs, rs)
		}
		out = append(out, ps)
	}
	return out
}

type tagState struct{ mark, sup, sub bool }

func (s tagState) tag() string {
	var parts []string
	if s.mark {
		parts = append(parts, TagMark)
	}
	if s.sup {
		parts = append(parts, TagSup)
	}
	if s.sub {
		parts = append(parts, TagSub)
	}
	return strings.Join(parts, "+")
}

// expand turns the stand-in runes of a parsed paragraph into tagged runs and
// attachments.
func (c *Codec) expand(p richtext.Paragraph, tokens []string, st *tagState) richtext.Paragraph {
	var runs []richtext.Run
	add := func(r richtext.Run) {
		if n := len(runs); n > 0 && r.Attributes.Attachment == nil && runs[n-1].Attributes == r.Attributes {
			runs[n-1].Text += r.Text
			return
		}
		runs = append(runs, r)
	}
	for _, r := range p.Runs {
		if r.Attributes.Attachment != nil {
			r.Attributes = c.withImage(r.Attributes)
			add(r)
			continue
		}
		var buf strings.Builder
		flush := func() {
			if buf.Len() > 0 {
				a := r.Attributes
				a.Tag = st.tag()
				add(richtext.Run{Text: buf.String(), Attributes: a})
				buf.Reset()
			}
		}
		for _, ch := range r.Text {
			switch {
			case ch == markOpen || ch == markClose:
				flush()
				st.mark = ch == markOpen
			case ch == supOpen || ch == supClose:
				flush()
				st.sup = ch == supOpen
			case ch == subOpen || ch == subClose:
				flush()
				st.sub = ch == subOpen
			case ch >= rawBase && int(ch-rawBase) < len(tokens):
				flush()
				a := r.Attributes
				a.Attachment = c.inlineRaw(tokens[ch-rawBase])
				add(richtext.Run{Text: string(richtext.ObjectReplacement), Attributes: a})
			default:
				buf.WriteRune(ch)
			}
		}
		flush()
	}
	p.Runs = runs
	return p
}

var imageSize = regexp.MustCompile(`\s+=\d*x?\d*$`)

// withImage gives a parsed image attachment its picture, or a labeled chip.
func (c *Codec) withImage(a richtext.Attributes) richtext.Attributes {
	old := a.Attachment
	at := &richtext.Attachment{Name: old.Name, URL: old.URL}
	a.Attachment = at
	maxW := float32(560)
	var img image.Image
	if c.LoadImage != nil && old.URL != "" && !strings.Contains(old.URL, "://") {
		if data, err := c.LoadImage(imageSize.ReplaceAllString(old.URL, ""), c.BaseDir); err == nil {
			if bmp, err := ui.DecodeBitmap(data); err == nil {
				w, h := bmp.Size()
				fw, fh := float32(w), float32(h)
				if fw > maxW {
					fh, fw = fh*maxW/fw, maxW
				}
				at.Image, at.Width, at.Height = bmp, fw, fh
				return a
			}
		}
	}
	_ = img
	label := old.Name
	if label == "" {
		label = old.URL
	}
	at.Width, at.Height = 160, 24
	at.Draw = func(p *ui.Painter, r ui.Rect) { drawChip(p, r, "图片", label) }
	return a
}

// inlineRaw makes the attachment for an inline math span or footnote mark.
func (c *Codec) inlineRaw(raw string) *richtext.Attachment {
	at := &richtext.Attachment{Name: raw, Data: rawRef{raw: raw}}
	if strings.HasPrefix(raw, "[^") {
		label := strings.TrimSuffix(strings.TrimPrefix(raw, "[^"), "]")
		w := float32(len([]rune(label)))*6 + 10
		at.Width, at.Height = w, c.size()
		sz := c.size() * 0.7
		at.Draw = func(p *ui.Painter, r ui.Rect) {
			p.Text(r.X+2, r.Y, label, sz, p.Theme().Accent)
		}
		return at
	}
	display := strings.HasPrefix(raw, "$$")
	tex := strings.Trim(raw, "$")
	it := &mathItem{tex: tex, display: display, at: at}
	c.math = append(c.math, it)
	at.Width, at.Height = float32(len([]rune(raw)))*c.size()*0.55, c.size()*1.4
	at.Draw = func(p *ui.Painter, r ui.Rect) { c.drawMath(p, r, it, raw) }
	return at
}

func (c *Codec) drawMath(p *ui.Painter, r ui.Rect, it *mathItem, raw string) {
	c.mu.Lock()
	bmp, err := it.bmp, it.err
	c.mu.Unlock()
	if bmp != nil {
		p.Image(bmp, r, ui.Contain)
		return
	}
	col := p.Theme().TextMuted
	if err != nil {
		col = p.Theme().Danger
	}
	p.Text(r.X, r.Y, raw, c.size()*0.9, col)
}

// renderMath typesets every formula of the last Load in one batch.
func (c *Codec) renderMath() {
	if len(c.math) == 0 {
		return
	}
	exprs := make([]render.MathExpression, len(c.math))
	for i, it := range c.math {
		exprs[i] = render.MathExpression{TeX: it.tex, Display: it.display}
	}
	results, err := render.RenderMathBatch(exprs, render.MathOptions{}, float64(c.size()), c.fg())
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, it := range c.math {
		switch {
		case err != nil:
			it.err = err
		case i < len(results) && results[i].Error != nil:
			it.err = results[i].Error
		case i < len(results) && results[i].Image != nil:
			it.err = nil
			b := results[i].Image.Bounds()
			w, h := float32(b.Dx()), float32(b.Dy())
			if it.bmp == nil {
				it.bmp = ui.NewBitmap(results[i].Image)
			} else {
				it.bmp.Update(results[i].Image)
			}
			it.at.Width, it.at.Height = max(w, 1), max(h, 1)
		}
	}
}

func (c *Codec) fg() color.Color {
	if c.Dark {
		return color.RGBA{0xee, 0xee, 0xf0, 0xff}
	}
	return color.RGBA{0x1d, 0x1d, 0x1f, 0xff}
}

// Retint re-typesets math for a changed theme or size; sizes stay as they
// were, so the text layout is not disturbed.
func (c *Codec) Retint(dark bool) {
	if c.Dark == dark {
		return
	}
	c.Dark = dark
	c.renderMath()
}

// Save writes text back as Markdown. Protected blocks and inline source
// come out exactly as they were read.
func (c *Codec) Save(t richtext.Text) string {
	var raws []string
	slot := func(raw string) string {
		raws = append(raws, raw)
		return string(rawBase + rune(len(raws)-1))
	}
	var paras []richtext.Paragraph
	var langs []string
	prevCode := false
	for _, p := range t.Paragraphs() {
		if p.Style.CodeBlock && !prevCode {
			langs = append(langs, "")
		}
		prevCode = p.Style.CodeBlock
		var runs []richtext.Run
		var stack []string
		var prev richtext.Attributes
		marker := func(tag string, open bool, a richtext.Attributes) richtext.Run {
			a.Tag, a.Attachment = "", nil
			o, cl := markOpen, markClose
			switch tag {
			case TagSup:
				o, cl = supOpen, supClose
			case TagSub:
				o, cl = subOpen, subClose
			}
			if open {
				return richtext.Run{Text: string(o), Attributes: a}
			}
			return richtext.Run{Text: string(cl), Attributes: a}
		}
		for _, r := range p.Runs {
			var want []string
			if tag := r.Attributes.Tag; strings.HasPrefix(tag, langTag) {
				if p.Style.CodeBlock && len(langs) > 0 && langs[len(langs)-1] == "" {
					langs[len(langs)-1] = strings.TrimPrefix(tag, langTag)
				}
			} else if tag != "" {
				want = strings.Split(tag, "+")
			}
			// Close tags no longer wanted, reopening the still wanted ones
			// they enclose so the markers nest.
			keep := len(stack)
			for i, tg := range stack {
				if !contains(want, tg) {
					keep = i
					break
				}
			}
			if keep < len(stack) {
				for i := len(stack) - 1; i >= keep; i-- {
					runs = append(runs, marker(stack[i], false, prev))
				}
				reopen := []string{}
				for _, tg := range stack[keep:] {
					if contains(want, tg) {
						reopen = append(reopen, tg)
					}
				}
				stack = stack[:keep]
				for _, tg := range reopen {
					runs = append(runs, marker(tg, true, r.Attributes))
					stack = append(stack, tg)
				}
			}
			for _, tg := range []string{TagMark, TagSup, TagSub} {
				if contains(want, tg) && !contains(stack, tg) {
					runs = append(runs, marker(tg, true, r.Attributes))
					stack = append(stack, tg)
				}
			}
			a := r.Attributes
			a.Tag = ""
			if ref, ok := attachmentRaw(a.Attachment); ok {
				if ref.block {
					// A block owns its paragraph: whatever shares the
					// line is split off so the source stays valid.
					for i := len(stack) - 1; i >= 0; i-- {
						runs = append(runs, marker(stack[i], false, prev))
					}
					stack = nil
					if hasText(runs) {
						paras = append(paras, richtext.Paragraph{Style: p.Style, Runs: runs})
					}
					paras = append(paras, richtext.Paragraph{Runs: []richtext.Run{{Text: slot(ref.raw)}}})
					runs = nil
					continue
				}
				runs = append(runs, richtext.Run{Text: slot(ref.raw)})
			} else {
				runs = append(runs, richtext.Run{Text: r.Text, Attributes: a})
			}
			prev = a
		}
		for i := len(stack) - 1; i >= 0; i-- {
			runs = append(runs, marker(stack[i], false, prev))
		}
		if len(runs) > 0 || len(p.Runs) == 0 {
			p.Runs = runs
			paras = append(paras, p)
		}
	}
	md := withLanguages(richtext.TextOf(paras...).Markdown(), langs)
	var out strings.Builder
	for _, ch := range md {
		switch {
		case ch == markOpen || ch == markClose:
			out.WriteString("==")
		case ch == supOpen || ch == supClose:
			out.WriteByte('^')
		case ch == subOpen || ch == subClose:
			out.WriteByte('~')
		case ch >= rawBase && int(ch-rawBase) < len(raws):
			out.WriteString(raws[ch-rawBase])
		default:
			out.WriteRune(ch)
		}
	}
	return strings.TrimRight(bareLinks(out.String()), "\n") + "\n"
}

// withLanguages puts the info strings back on the fences Markdown wrote.
func withLanguages(md string, langs []string) string {
	lines := strings.Split(md, "\n")
	fence, n := "", -1
	for i, l := range lines {
		body := prefixQuote.ReplaceAllString(l, "")
		if fence == "" {
			if m := fenceOpen.FindStringSubmatch(body); m != nil {
				fence = m[1]
				n++
				if n < len(langs) && langs[n] != "" && strings.TrimSpace(m[2]) == "" {
					lines[i] = l + langs[n]
				}
			}
			continue
		}
		if closesFence(body, fence) {
			fence = ""
		}
	}
	return strings.Join(lines, "\n")
}

var autoLink = regexp.MustCompile(`\[(https?://[^\]\s]+)\]\((https?://[^)\s]+)\)`)

// bareLinks turns [url](url) back into the bare URL the author typed.
func bareLinks(md string) string {
	return autoLink.ReplaceAllStringFunc(md, func(m string) string {
		sub := autoLink.FindStringSubmatch(m)
		if strings.ReplaceAll(sub[1], "\\", "") == sub[2] {
			return sub[2]
		}
		return m
	})
}

func attachmentRaw(at *richtext.Attachment) (rawRef, bool) {
	if at == nil {
		return rawRef{}, false
	}
	ref, ok := at.Data.(rawRef)
	return ref, ok
}

func hasText(runs []richtext.Run) bool {
	for _, r := range runs {
		if strings.TrimSpace(r.Text) != "" {
			return true
		}
	}
	return false
}

// HasBlock reports whether text holds a protected block.
func HasBlock(t richtext.Text) bool {
	for _, p := range t.Paragraphs() {
		for _, r := range p.Runs {
			if ref, ok := attachmentRaw(r.Attributes.Attachment); ok && ref.block {
				return true
			}
		}
	}
	return false
}

// Footnotes counts the footnote marks in text, for numbering a new one.
func Footnotes(t richtext.Text) int {
	n := 0
	for _, p := range t.Paragraphs() {
		for _, r := range p.Runs {
			if ref, ok := attachmentRaw(r.Attributes.Attachment); ok && strings.HasPrefix(ref.raw, "[^") {
				n++
			}
		}
	}
	return n
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
