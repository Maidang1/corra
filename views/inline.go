package views

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/render"
)

// inlineLeaf is one flattened inline run with its complete style, produced by
// walking the Inline tree. Links become clickable text children of the
// paragraph (Typora-style) instead of static spans.
type inlineLeaf struct {
	Text string

	Bold      bool
	Italic    bool
	Strike    bool
	Code      bool
	Underline bool
	Highlight bool
	Sup       bool
	Sub       bool
	Kbd       bool
	Link      string
}

// leavesOf flattens the Inline tree into styled leaves.
func leavesOf(ins []render.Inline, st inlineLeaf, out *[]inlineLeaf) {
	for _, in := range ins {
		s := st
		switch in.Kind {
		case render.InlineBold:
			s.Bold = true
		case render.InlineItalic:
			s.Italic = true
		case render.InlineStrikethrough:
			s.Strike = true
		case render.InlineHighlight:
			s.Highlight = true
		case render.InlineUnderline:
			s.Underline = true
		case render.InlineSuperscript:
			s.Sup = true
		case render.InlineSubscript:
			s.Sub = true
		case render.InlineCode:
			s.Code = true
			s.Text = in.Text
			*out = append(*out, s)
			continue
		case render.InlineKbd:
			s.Kbd = true
			s.Text = in.Text
			*out = append(*out, s)
			continue
		case render.InlineLink:
			s.Link = in.URL
		case render.InlineImage:
			label := in.Text
			if label == "" {
				label = "图片"
			}
			s.Text = "🖼 " + label
			*out = append(*out, s)
			continue
		case render.InlineHardBreak:
			*out = append(*out, inlineLeaf{Text: "\n"})
			continue
		case render.InlineMath:
			// Preview shows the TeX source in a quiet mono style; the native
			// editor typesets inline math in place (MyGo text flow has no
			// inline image spans).
			s.Code = true
			s.Text = "$" + in.Text + "$"
			*out = append(*out, s)
			continue
		}
		if len(in.Children) > 0 {
			leavesOf(in.Children, s, out)
		} else if in.Text != "" {
			s.Text = in.Text
			*out = append(*out, s)
		}
	}
}

// highlightColor is the ==mark== background for the theme.
func highlightColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#6b5d1a66")
	}
	return ui.Hex("#fff1a8")
}

// inlineFlow renders inline markdown as one rich-text paragraph whose links
// are clickable (internal #anchors scroll, .md files open, URLs go to the
// browser). baseSize is the paragraph's font size.
func (v *View) inlineFlow(c *ui.Context, src string, t *ui.Theme, baseSize float32) ui.Element {
	var leaves []inlineLeaf
	leavesOf(render.ParseInlines(src), inlineLeaf{}, &leaves)
	return ui.RichText(c).FontSize(baseSize).LineHeight(1.4).FillWidth().Children(func() {
		for _, lf := range leaves {
			if lf.Text == "" {
				continue
			}
			e := ui.Text(c, lf.Text)
			if lf.Bold {
				e = e.FontWeight(700)
			}
			if lf.Italic {
				e = e.Italic()
			}
			if lf.Strike {
				e = e.Strikethrough()
			}
			if lf.Underline {
				e = e.Underline()
			}
			if lf.Code || lf.Kbd {
				e = e.Font(fontMono).FontSize(baseSize - 1).Background(t.SurfacePressed)
			}
			if lf.Highlight {
				e = e.Background(highlightColor(t))
			}
			if lf.Sup {
				e = e.FontSize(baseSize*0.75 - 1).FontFeatures("sups")
			}
			if lf.Sub {
				e = e.FontSize(baseSize*0.75 - 1).FontFeatures("subs")
			}
			if lf.Link != "" {
				url := lf.Link
				e = e.TextColor(t.Accent).Underline().Cursor(ui.CursorPointer).
					Tooltip(url).
					OnClick(func() { v.openLink(c, url) })
			} else if v.canEdit() {
				// Clicking plain text in the preview returns to editing,
				// Apple-Notes style; links keep their own behavior.
				e = e.OnClick(func() { v.toggleMode() })
			}
		}
	})
}

// openLink follows a markdown link: #heading anchors scroll within the
// preview, relative .md files open as notes, everything else goes to the
// system browser.
func (v *View) openLink(c *ui.Context, url string) {
	if strings.HasPrefix(url, "#") {
		anchor := strings.TrimPrefix(url, "#")
		for i := range v.App.Doc.Blocks {
			b := &v.App.Doc.Blocks[i]
			if b.Kind == render.KindHeading && anchorSlug(b.HeadingText()) == anchor {
				v.scrollToBlock = i
				v.App.InvalidateUI()
				return
			}
		}
		return
	}
	if strings.HasSuffix(strings.ToLower(url), ".md") && !strings.Contains(url, "://") {
		path := url
		if !filepath.IsAbs(path) && v.App.Path != "" {
			path = filepath.Join(dirOf(v.App.Path), filepath.FromSlash(path))
		}
		v.openNote(path)
		return
	}
	c.OpenURL(url)
}

// anchorSlug turns a heading into a GitHub-style anchor.
func anchorSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var out []rune
	for _, r := range s {
		switch {
		case r == ' ':
			out = append(out, '-')
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r > 0x7f:
			out = append(out, r)
		}
	}
	return string(out)
}

// wordCount returns Typora-style counts for the document source.
type wordCount struct {
	Words   int // latin words + CJK characters
	Chars   int // all runes, excluding whitespace
	Lines   int
	Minutes int // estimated reading time
}

// countWords counts CJK characters individually and latin runs as words, as
// Typora does for mixed documents.
func countWords(src string) wordCount {
	wc := wordCount{Lines: strings.Count(src, "\n") + 1}
	if strings.TrimSpace(src) == "" {
		wc.Lines = 0
	}
	var latin strings.Builder
	flush := func() {
		if latin.Len() > 0 {
			wc.Words++
			latin.Reset()
		}
	}
	for _, r := range src {
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		case r > 0x2e00: // CJK & full-width: each character is a word
			flush()
			wc.Words++
			wc.Chars++
		default:
			latin.WriteRune(r)
			wc.Chars++
		}
	}
	flush()
	cjk := 0
	for _, r := range src {
		if r > 0x2e00 {
			cjk++
		}
	}
	latinWords := wc.Words - cjk
	wc.Minutes = latinWords/275 + cjk/400
	if wc.Words > 0 && wc.Minutes == 0 {
		wc.Minutes = 1
	}
	return wc
}

func (wc wordCount) String() string {
	if wc.Words == 0 {
		return "0 字"
	}
	return fmt.Sprintf("%d 字 · %d 字符 · %d 行 · 约 %d 分钟", wc.Words, wc.Chars, wc.Lines, wc.Minutes)
}
