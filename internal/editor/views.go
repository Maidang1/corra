package editor

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/internal/doc"
)

// headingScale is a heading's size relative to the body, by level.
var headingScale = [7]float32{1, 1.75, 1.45, 1.25, 1.1, 1, 0.95}

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
			l.Font.Weight = 650
			l.Font.LineHeight = 1.3
			l.Before = l.Font.Size * 0.5
			l.After = l.Font.Size * 0.35
			return l
		}},
		"code_block": {Text: func(n *doc.Node, env *Env) TextLook {
			l := bodyLook(env)
			l.Font = monoFont(env, 0.9)
			l.Background = env.Theme.Surface
			l.Radius, l.PadX, l.PadY = 6, 12, 10
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
				Paint: func(p *ui.Painter, r ui.Rect, env *Env) {
					p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 3, H: r.H}, env.Theme.Border, 1.5)
				},
			}
		}},
		"bullet_list":  {Container: listLook(false)},
		"ordered_list": {Container: listLook(true)},
		"list_item": {Container: func(n *doc.Node, env *Env) ContainerLook {
			return ContainerLook{}
		}},
		"horizontal_rule": {Atom: func(n *doc.Node, env *Env) AtomLook {
			return AtomLook{
				Height: func(float32) float32 { return env.Font.Size * 1.5 },
				Paint: func(p *ui.Painter, r ui.Rect, env *Env) {
					p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H/2, W: r.W, H: 1}, env.Theme.Border, 0)
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
			s.Size = env.Font.Size * 0.9
			s.Background = env.Theme.Surface
			s.Radius = 3
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
		return ContainerLook{
			Indent: env.Font.Size * 1.6,
			After:  env.Font.Size * 0.6,
			Marker: func(i int, child *doc.Node) string {
				if ordered {
					return strconv.Itoa(start+i) + "."
				}
				return "•"
			},
			Check: func(i int, child *doc.Node) (bool, bool) {
				return child.Attrs().Bool("task"), child.Attrs().Bool("checked")
			},
		}
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
		p.Fill(r, env.Theme.Surface, 4)
		p.Text(r.X+4, r.Y+r.H*0.7, "🖼 "+n.Attrs().String("alt"), size*0.8, env.Theme.TextMuted)
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
		p.Fill(r, env.Theme.Surface, 8)
		p.Stroke(r, env.Theme.Border, 1, 8)
		col := env.Theme.Text
		if !ok {
			col = env.Theme.Danger
		}
		p.Text(r.X+14, r.Y+size*1.5, "📄 "+title, size, col)
		if excerpt != "" {
			p.Text(r.X+14, r.Y+size*2.9, excerpt, size*0.8, env.Theme.TextMuted)
		}
	}}
}

// lineNumbers numbers a code block's lines in its gutter: each line that
// starts after a newline, not those a long line wraps onto.
func lineNumbers(p *ui.Painter, origin ui.Point, lines []ui.TextLine, text string, env *Env) {
	size := env.Font.Size * 0.75
	units := utf16Units(text)
	n := 0
	for _, l := range lines {
		if l.Start > 0 && (l.Start > len(units) || units[l.Start-1] != '\n') {
			continue
		}
		n++
		label := strconv.Itoa(n)
		p.Text(origin.X-size*0.62*float32(len(label))-env.Font.Size*0.6, origin.Y+l.Baseline, label, size, env.Theme.TextMuted)
	}
	if len(lines) == 0 {
		p.Text(origin.X-env.Font.Size*1.1, origin.Y+env.Font.Size, "1", size, env.Theme.TextMuted)
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
