package editor

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/render"
)

// headingScale is a heading's size relative to the body, by level;
// headingBefore and headingAfter its room above and below, in body sizes:
// more above than below, so a heading belongs to the section it opens.
var (
	headingScale  = [7]float32{1, 1.75, 1.45, 1.25, 1.1, 1, 0.95}
	headingBefore = [7]float32{0, 1.6, 1.6, 1.3, 1.1, 1, 1}
	headingAfter  = [7]float32{0, 0.6, 0.5, 0.4, 0.3, 0.3, 0.3}
)

func bodyLook(env *Env) TextLook {
	f := env.Font
	if f.LineHeight == 0 {
		f.LineHeight = 1.6
	}
	return TextLook{Font: f, Color: env.Theme.Text, After: f.Size * 0.6}
}

func monoFont(env *Env, scale float32) ui.Font {
	f := env.Font
	f.Family = env.Mono
	f.Size *= scale
	f.LineHeight = 1.5
	return f
}

// DefaultViews returns views for the built-in node types and marks.
func DefaultViews() map[string]ViewSpec {
	return map[string]ViewSpec{
		"paragraph": {Text: func(n *doc.Node, env *Env) TextLook { return bodyLook(env) }},
		"heading": {Text: func(n *doc.Node, env *Env) TextLook {
			l := bodyLook(env)
			level := min(max(n.Attrs().Int("level"), 1), 6)
			l.Font.Size *= headingScale[level]
			l.Font.Weight = 600
			if level <= 2 {
				l.Font.Weight = 700
			}
			l.Font.LineHeight = 1.3
			l.Before = env.Font.Size * headingBefore[level]
			l.After = env.Font.Size * headingAfter[level]
			return l
		}},
		"code_block": {Text: func(n *doc.Node, env *Env) TextLook {
			l := bodyLook(env)
			l.Font = monoFont(env, 0.875)
			l.Background = env.Tint()
			l.Radius, l.PadX, l.PadY = 6, 14, 12
			l.Before, l.After = env.Font.Size*0.4, env.Font.Size
			if lang := n.Attrs().String("language"); lang != "" {
				l.Colorize = func(text string) []ui.Span { return highlight(text, lang, env) }
			}
			if env.CodeLineNumbers {
				l.Gutter = l.Font.Size * 2.4
				l.Decorate = lineNumbers
			}
			return l
		}},
		"raw": {Text: func(n *doc.Node, env *Env) TextLook {
			l := bodyLook(env)
			l.Font = monoFont(env, 0.9)
			l.Color = env.Theme.TextMuted
			return l
		}},
		"blockquote": {Container: func(n *doc.Node, env *Env) ContainerLook {
			return ContainerLook{
				Indent: env.Font.Size,
				After:  env.Font.Size * 0.6,
				// Quoted words step back: the same size in a quieter ink.
				Color: env.Theme.TextMuted,
				Paint: func(p *ui.Painter, r ui.Rect, env *Env) {
					p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 3, H: r.H}, env.Rule(), 1.5)
				},
			}
		}},
		"bullet_list":  {Container: listLook(false)},
		"ordered_list": {Container: listLook(true)},
		"list_item": {Container: func(n *doc.Node, env *Env) ContainerLook {
			// An item of text and the lists nested under it hangs together
			// as a tight list's items do.
			for i := 1; i < n.NumChildren(); i++ {
				if !n.Child(i).Type().InGroup("list") {
					return ContainerLook{}
				}
			}
			return ContainerLook{Gap: env.Font.Size * 0.3}
		}},
		"horizontal_rule": {Atom: func(n *doc.Node, env *Env) AtomLook {
			return AtomLook{
				Height: func(float32) float32 { return env.Font.Size * 1.5 },
				Paint: func(p *ui.Painter, r ui.Rect, env *Env) {
					p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H/2, W: r.W, H: 1}, env.Rule(), 0)
				},
			}
		}},
		"hard_break": {Inline: func(n *doc.Node, env *Env) InlineLook { return InlineLook{Text: "\n"} }},
		"image":      {Inline: imageView},

		"bold":   {Mark: func(m doc.Mark, env *Env, s *ui.Span) { s.Weight = 700 }},
		"italic": {Mark: func(m doc.Mark, env *Env, s *ui.Span) { s.Italic = true }},
		"strike": {Mark: func(m doc.Mark, env *Env, s *ui.Span) { s.Strikethrough = true }},
		"code": {Mark: func(m doc.Mark, env *Env, s *ui.Span) {
			s.Font = env.Mono
			s.Size = env.Font.Size * 0.875
			s.Background = env.Tint()
			s.Radius, s.PaddingX = 4, 4
		}},
		"link": {Mark: func(m doc.Mark, env *Env, s *ui.Span) {
			s.Color = env.Theme.Accent
			if IsPageTarget(m.Attrs.String("href")) {
				// A mention of another note: a chip.
				s.Background, s.Radius, s.PaddingX = env.Theme.Accent.Alpha(0.12), 4, 3
				return
			}
			s.Underline = true
		}},
		"raw_inline": {Mark: func(m doc.Mark, env *Env, s *ui.Span) {
			s.Font = env.Mono
			s.Color = env.Theme.TextMuted
		}},
	}
}

func listLook(ordered bool) func(n *doc.Node, env *Env) ContainerLook {
	return func(n *doc.Node, env *Env) ContainerLook {
		start := n.Attrs().Int("start")
		look := ContainerLook{
			Indent: env.Font.Size * 1.6,
			After:  env.Font.Size * 0.6,
			Marker: func(i int, child *doc.Node) string {
				if ordered {
					return strconv.Itoa(start+i) + "."
				}
				return Bullet
			},
			Check: func(i int, child *doc.Node) (bool, bool) {
				return child.Attrs().Bool("task"), child.Attrs().Bool("checked")
			},
		}
		if n.Attrs().Bool("tight") {
			look.Gap = env.Font.Size * 0.3
		}
		return look
	}
}

// imageView shows a picture at its size, at most maxImageWidth wide, or a
// captioned box until it loads or when it cannot.
func imageView(n *doc.Node, env *Env) InlineLook {
	size := env.Font.Size
	src := n.Attrs().String("src")
	if env.Pages != nil && IsPageTarget(src) {
		return pageCard(n, src, env)
	}
	if env.Images != nil && src != "" {
		if bmp, ready := env.Images.Image(src, env.BaseDir); ready && bmp != nil {
			iw, ih := bmp.Size()
			w, h := sized(float32(iw), float32(ih), n.Attrs().String("size"), size*maxImageWidth)
			if limit := size * maxImageWidth; w > limit {
				w, h = limit, h*limit/w
			}
			return InlineLook{Width: max(w, 1), Height: max(h, 1), Paint: func(p *ui.Painter, r ui.Rect, env *Env) {
				p.Image(bmp, r, ui.Contain)
			}}
		}
	}
	return InlineLook{Width: size * 6, Height: size * 1.4, Paint: func(p *ui.Painter, r ui.Rect, env *Env) {
		p.Fill(r, env.Tint(), 4)
		f := env.Font
		f.Size *= 0.8
		p.Clip(r, 4, func() { DrawLabelIn(p, r.X+6, r, "🖼 "+n.Attrs().String("alt"), f, env.Theme.TextMuted) })
	}}
}

// maxImageWidth caps pictures, in body font sizes (about 640 DIPs).
const maxImageWidth = 40

// sized applies a size suffix ("300x200", "300", "x200", "50%") to a
// picture of w×h; a percentage is of limit, the widest a picture shows.
func sized(w, h float32, spec string, limit float32) (float32, float32) {
	if w <= 0 || h <= 0 || spec == "" {
		return w, h
	}
	if p, ok := strings.CutSuffix(spec, "%"); ok {
		if v, err := strconv.ParseFloat(p, 32); err == nil && v > 0 {
			nw := limit * float32(v) / 100
			return nw, h * nw / w
		}
		return w, h
	}
	ws, hs, _ := strings.Cut(spec, "x")
	sw, _ := strconv.ParseFloat(ws, 32)
	sh, _ := strconv.ParseFloat(hs, 32)
	switch {
	case sw > 0 && sh > 0:
		return float32(sw), float32(sh)
	case sw > 0:
		return float32(sw), h * float32(sw) / w
	case sh > 0:
		return w * float32(sh) / h, float32(sh)
	}
	return w, h
}

// IsPageTarget reports whether a link or image points at another note
// ("other.md", "dir/note.markdown#part"): not a web page, an anchor or a
// picture.
func IsPageTarget(dest string) bool {
	if dest == "" || strings.HasPrefix(dest, "#") || strings.Contains(dest, ":") {
		return false
	}
	path, _, _ := strings.Cut(dest, "#")
	path = strings.ToLower(path)
	return strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".markdown")
}

// pageCard shows an embedded note: its title and the start of its text.
func pageCard(n *doc.Node, target string, env *Env) InlineLook {
	size := env.Font.Size
	title, excerpt, ok := env.Pages.Page(target, env.BaseDir)
	if title == "" {
		title = n.Attrs().String("alt")
	}
	return InlineLook{Width: size * maxImageWidth * 0.75, Height: size * 4, Paint: func(p *ui.Painter, r ui.Rect, env *Env) {
		p.Fill(r, env.Tint(), 8)
		p.Stroke(r, env.Theme.Border, 8, 1)
		col := env.Theme.Text
		if !ok {
			col = env.Theme.Danger
		}
		p.Clip(r, 8, func() {
			f := env.Font
			f.Weight = 600
			DrawLabel(p, r.X+14, r.Y+size*1.6, "📄 "+title, f, col)
			if excerpt != "" {
				f.Weight, f.Size = 0, size*0.8
				DrawLabel(p, r.X+14, r.Y+size*3, excerpt, f, env.Theme.TextMuted)
			}
		})
	}}
}

// lineNumbers numbers a code block's lines in its gutter: each line that
// starts after a newline, not those a long line wraps onto.
func lineNumbers(p *ui.Painter, origin ui.Point, lines []ui.TextLine, text string, env *Env) {
	f := monoFont(env, 0.75)
	units := utf16Units(text)
	number := func(n int, baseline float32) {
		l := ui.ShapeText(strconv.Itoa(n), f, 0)
		w, _ := l.Size()
		var b float32
		if ls := l.Lines(); len(ls) > 0 {
			b = ls[0].Baseline
		}
		p.TextLayout(l, origin.X-w-env.Font.Size*0.6, origin.Y+baseline-b, env.Theme.TextMuted.Alpha(0.7))
	}
	n := 0
	for _, l := range lines {
		if l.Start > 0 && (l.Start > len(units) || units[l.Start-1] != '\n') {
			continue
		}
		n++
		number(n, l.Baseline)
	}
	if len(lines) == 0 {
		number(1, lineHeight(monoFont(env, 0.875))*0.75)
	}
}

// highlight colors code in language by its syntax.
func highlight(code, language string, env *Env) []ui.Span {
	colors := CodeColors(env.Theme)
	var out []ui.Span
	for _, h := range render.Highlight(code, language) {
		out = append(out, ui.Span{Text: h.Text, Color: colors[h.Class]})
	}
	return out
}

// CodeColors maps highlight classes to colors that read on the theme.
func CodeColors(t *ui.Theme) map[string]ui.Color {
	muted := t.TextMuted
	if t.Dark {
		return map[string]ui.Color{
			"text":     t.Text,
			"keyword":  ui.Hex("#c792ea"),
			"string":   ui.Hex("#98c379"),
			"number":   ui.Hex("#d19a66"),
			"comment":  muted,
			"operator": ui.Hex("#56b6c2"),
			"type":     ui.Hex("#e5c07b"),
			"literal":  ui.Hex("#d19a66"),
			"name":     ui.Hex("#61afef"),
		}
	}
	return map[string]ui.Color{
		"text":     t.Text,
		"keyword":  ui.Hex("#a626a4"),
		"string":   ui.Hex("#50a14f"),
		"number":   ui.Hex("#b76b01"),
		"comment":  muted,
		"operator": ui.Hex("#0184bc"),
		"type":     ui.Hex("#c18401"),
		"literal":  ui.Hex("#b76b01"),
		"name":     ui.Hex("#4078f2"),
	}
}

// utf16Units returns text as UTF-16 units, to look up layout offsets.
func utf16Units(text string) []rune {
	var out []rune
	for _, r := range text {
		if r >= 0x10000 {
			out = append(out, 0xD800, 0xDC00)
		} else {
			out = append(out, r)
		}
	}
	return out
}
