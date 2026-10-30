package views

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"github.com/Maidang1/corra/internal/mgdoc"
)

// inlineRich is implemented by editors drawn inside the MyGo view tree
// instead of a native overlay; it is what Windows and Linux use.
type inlineRich interface {
	Render(c *ui.Context)
}

const richFontSize = 16

// MygoRich is a RichEditor built on MyGo's own rich-text component, so the
// 文稿 mode works on every platform. Markdown is the interchange format; what
// the component cannot show stays a protected block (see mgdoc).
type MygoRich struct {
	doc      *richtext.Document
	codec    *mgdoc.Codec
	onChange func()
	editable bool
	zoom     float64
	body     string
	mono     string
	snapVer  uint64
	focus    bool

	// OnOpenDocument opens a link to another note; it reports success.
	OnOpenDocument func(target string) bool
	// OnLinkRequest asks the app for its insert-link dialog (⌘K).
	OnLinkRequest func()

	find findState
}

type findState struct {
	open, replace bool
	query, repl   string
	lastQuery     string
	lastVersion   uint64
	matches       [][2]int
	cur           int
	focusQuery    bool
}

// NewMygoRich returns an editor that calls changed after each edit.
// loadImage reads local pictures; invalidate repaints after background
// rendering finishes.
func NewMygoRich(changed func(), loadImage func(url, baseDir string) ([]byte, error), invalidate func()) *MygoRich {
	m := &MygoRich{
		doc:      richtext.NewDocument(richtext.Text{}),
		onChange: changed,
		editable: true,
		zoom:     1,
		codec: &mgdoc.Codec{
			FontSize:   richFontSize,
			LoadImage:  loadImage,
			Invalidate: invalidate,
		},
	}
	m.snapVer = m.doc.Version()
	return m
}

func (m *MygoRich) touched() {
	if m.onChange != nil {
		m.onChange()
	}
}

func (m *MygoRich) Load(source string) {
	m.doc.SetText(m.codec.Load(source))
	m.snapVer = m.doc.Version()
	m.find.lastVersion = 0
}

func (m *MygoRich) Snapshot() (string, bool, error) {
	ver := m.doc.Version()
	if ver == m.snapVer {
		return "", false, nil
	}
	m.snapVer = ver
	return m.codec.Save(m.doc.Text()), true, nil
}

func (m *MygoRich) tagStyle(tag string, s *ui.Span) {
	base := float32(richFontSize * m.zoom)
	for _, t := range strings.Split(tag, "+") {
		switch t {
		case mgdoc.TagMark:
			s.Background, s.Color, s.Radius = ui.Hex("#fde68a"), ui.Hex("#1f2937"), 3
		case mgdoc.TagSup:
			s.Size, s.BaselineShift = base*0.72, base*0.36
		case mgdoc.TagSub:
			s.Size, s.BaselineShift = base*0.72, -base*0.14
		}
	}
}

func (m *MygoRich) Render(c *ui.Context) {
	m.codec.Retint(c.Theme().Dark)
	ui.Column(c).Fill().Gap(0).Children(func() {
		if m.editable {
			m.toolbar(c)
		}
		if m.find.open {
			m.findBar(c)
		}
		ui.Box(c).Fill().Children(func() { m.editor(c) })
	})
}

func (m *MygoRich) editor(c *ui.Context) {
	body := fontUI
	if m.body != "" {
		body = m.body + ", " + fontUI
	}
	mono := fontMono
	if m.mono != "" {
		mono = m.mono + ", " + fontMono
	}
	opts := richtext.EditorOptions{
		Label:         "文稿",
		ReadOnly:      !m.editable,
		Font:          ui.Font{Family: body, Size: float32(richFontSize * m.zoom)},
		TagStyle:      m.tagStyle,
		OnLinkCommand: m.OnLinkRequest,
		OnLink: func(url string) {
			if m.OnOpenDocument != nil && m.OnOpenDocument(url) {
				return
			}
			if ui.SafeToOpen(url) {
				c.OpenURL(url)
			}
		},
		Theme: &richtext.Theme{CodeFont: mono, ParagraphSpacing: 0.55},
	}
	ui.Column(c).Fill().AlignItems(ui.Center).Children(func() {
		el := richtext.EditorBase(c, m.doc, opts).Fill().MaxWidth(float32(820*m.zoom)).Padding(12, 28)
		if m.focus {
			el.Focus()
		}
		if el.Changed() {
			m.touched()
		}
	})
	m.focus = false
}

type toolButton struct {
	label, tip string
	on         bool
	run        func()
}

func (m *MygoRich) toolbar(c *ui.Context) {
	f := m.doc.Formatting()
	tagOn := func(tag string) bool { return hasTag(f.Attributes.Tag, tag) }
	rows := [][]toolButton{{
		{"B", "粗体", f.Bold == richtext.FormatOn, func() { m.Format("bold") }},
		{"I", "斜体", f.Italic == richtext.FormatOn, func() { m.Format("italic") }},
		{"U", "下划线", f.Underline == richtext.FormatOn, func() { m.Format("underline") }},
		{"S", "删除线", f.Strikethrough == richtext.FormatOn, func() { m.Format("strike") }},
		{"</>", "行内代码", f.Code == richtext.FormatOn, func() { m.Format("code") }},
		{"高亮", "高亮", tagOn(mgdoc.TagMark), func() { m.Format("highlight") }},
		{"x²", "上标", tagOn(mgdoc.TagSup), func() { m.Format("sup") }},
		{"x₂", "下标", tagOn(mgdoc.TagSub), func() { m.Format("sub") }},
		{"链接", "插入链接", f.Link != "", func() {
			if m.OnLinkRequest != nil {
				m.OnLinkRequest()
			}
		}},
		{"H1", "一级标题", f.Paragraph.Heading == 1, func() { m.Format("heading1") }},
		{"H2", "二级标题", f.Paragraph.Heading == 2, func() { m.Format("heading2") }},
		{"H3", "三级标题", f.Paragraph.Heading == 3, func() { m.Format("heading3") }},
		{"•", "无序列表", f.Paragraph.List == richtext.Bullets, func() { m.Format("bullet") }},
		{"1.", "有序列表", f.Paragraph.List == richtext.Numbers, func() { m.Format("ordered") }},
		{"☐", "任务列表", f.Paragraph.List == richtext.Tasks, func() { m.Format("checklist") }},
		{"❝", "引用", f.Paragraph.Quote, func() { m.Format("quote") }},
		{"{ }", "代码块", f.Paragraph.CodeBlock, func() { m.Format("insert:```") }},
	}, {
		{"表格", "插入表格", false, func() { m.InsertMarkdown(tableTemplate) }},
		{"公式", "插入公式块", false, func() { m.Format("insert:$$") }},
		{"提示", "插入提示块", false, func() { m.Format("quote-alert:NOTE") }},
		{"脚注", "插入脚注", false, func() { m.Format("footnote:insert") }},
		{"图表", "插入 Mermaid 图表", false, func() { m.InsertMarkdown(diagramTemplate) }},
		{"分隔线", "插入分隔线", false, func() { m.Format("insert:---") }},
		{"查找", "查找与替换", m.find.open, func() { m.ShowFindReplaceBar() }},
	}}
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(2).Padding(4, 10).Children(func() {
		for ri, row := range rows {
			ui.Row(c).Gap(2).Children(func() {
				for _, b := range row {
					btn := ui.Button(c, b.label).Key(strconv.Itoa(ri)+b.label).KeepFocus().Tooltip(b.tip).Padding(2, 8).FontSize(12)
					if b.on {
						btn.Background(t.Accent.Alpha(0.18))
					}
					if btn.Clicked() {
						b.run()
					}
				}
			})
		}
	})
}

const (
	tableTemplate   = "| 列 1 | 列 2 |\n| --- | --- |\n| 内容 | 内容 |"
	diagramTemplate = "```mermaid\ngraph TD\n  A[开始] --> B[结束]\n```"
)

func hasTag(tag, want string) bool {
	for _, t := range strings.Split(tag, "+") {
		if t == want {
			return true
		}
	}
	return false
}

func toggleTag(tag, which string, on bool) string {
	var out []string
	for _, t := range []string{mgdoc.TagMark, mgdoc.TagSup, mgdoc.TagSub} {
		has := hasTag(tag, t)
		if t == which {
			has = on
		}
		// A run is either raised or lowered.
		if on && ((which == mgdoc.TagSup && t == mgdoc.TagSub) || (which == mgdoc.TagSub && t == mgdoc.TagSup)) {
			has = false
		}
		if has {
			out = append(out, t)
		}
	}
	return strings.Join(out, "+")
}

func (m *MygoRich) toggleTag(which string) {
	on := !hasTag(m.doc.Formatting().Attributes.Tag, which)
	m.doc.Format(func(a *richtext.Attributes) { a.Tag = toggleTag(a.Tag, which, on) })
}

func (m *MygoRich) Format(command string) {
	d := m.doc
	switch {
	case command == "bold":
		d.ToggleBold()
	case command == "italic":
		d.ToggleItalic()
	case command == "underline":
		d.ToggleUnderline()
	case command == "strike":
		d.ToggleStrikethrough()
	case command == "code":
		d.ToggleCode()
	case command == "highlight":
		m.toggleTag(mgdoc.TagMark)
	case command == "sup":
		m.toggleTag(mgdoc.TagSup)
	case command == "sub":
		m.toggleTag(mgdoc.TagSub)
	case command == "clear":
		d.Format(func(a *richtext.Attributes) {
			*a = richtext.Attributes{Attachment: a.Attachment}
		})
	case command == "body":
		d.FormatParagraphs(func(p *richtext.ParagraphStyle) { *p = richtext.ParagraphStyle{} })
	case command == "bullet":
		d.ToggleList(richtext.Bullets)
	case command == "ordered":
		d.ToggleList(richtext.Numbers)
	case command == "checklist":
		d.ToggleList(richtext.Tasks)
	case command == "quote":
		d.ToggleQuote()
	case command == "insert:```":
		d.ToggleCodeBlock()
	case command == "insert:$$":
		m.InsertMarkdown("$$\nE = mc^2\n$$")
	case command == "insert:---":
		m.InsertMarkdown("---")
	case command == "math-refresh":
		m.Load(m.codec.Save(d.Text()))
	case command == "footnote:insert":
		m.insertFootnote()
	case strings.HasPrefix(command, "quote-alert:"):
		label := strings.TrimPrefix(command, "quote-alert:")
		m.InsertMarkdown("> [!" + strings.ToUpper(label) + "]\n> 内容")
	case strings.HasPrefix(command, "link:"):
		m.insertLink(strings.TrimPrefix(command, "link:"))
	case strings.HasPrefix(command, "heading"):
		if n := strings.TrimPrefix(command, "heading"); len(n) == 1 && n[0] >= '1' && n[0] <= '6' {
			level := int(n[0] - '0')
			if d.Formatting().Paragraph.Heading == level {
				level = 0
			}
			d.SetHeading(level)
		}
	default:
		return
	}
	m.touched()
}

func (m *MygoRich) insertLink(spec string) {
	url, text, _ := strings.Cut(spec, "\n")
	if url == "" {
		return
	}
	if a, z := m.doc.Selection(); a != z && text == "" {
		m.doc.SetLink(url)
		return
	}
	if text == "" {
		text = url
	}
	m.doc.Insert(richtext.TextOf(richtext.Paragraph{Runs: []richtext.Run{{
		Text: text, Attributes: richtext.Attributes{Link: url},
	}}}))
}

// insertFootnote drops a numbered mark at the caret and appends its
// definition, a protected block edited in source mode.
func (m *MygoRich) insertFootnote() {
	n := strconv.Itoa(mgdoc.Footnotes(m.doc.Text()) + 1)
	mark := m.codec.Load("[^" + n + "]")
	m.doc.Insert(mark)
	def := m.codec.Load("[^" + n + "]: 脚注内容")
	m.doc.Edit(func(e *richtext.Edit) {
		end := e.Text().Len()
		a, z := e.Selection()
		e.Replace(end, end, richtext.TextOf(richtext.Paragraph{}, firstParagraph(def)))
		e.Select(a, z)
	})
}

func firstParagraph(t richtext.Text) richtext.Paragraph {
	for _, p := range t.Paragraphs() {
		return p
	}
	return richtext.Paragraph{}
}

// InsertMarkdown inserts source at the caret; blocks get their own
// paragraphs.
func (m *MygoRich) InsertMarkdown(source string) {
	t := m.codec.Load(source)
	if t.ParagraphCount() > 1 || mgdoc.HasBlock(t) {
		var paras []richtext.Paragraph
		paras = append(paras, richtext.Paragraph{})
		for _, p := range t.Paragraphs() {
			paras = append(paras, p)
		}
		paras = append(paras, richtext.Paragraph{})
		t = richtext.TextOf(paras...)
	}
	m.doc.Insert(t)
	m.touched()
}

func (m *MygoRich) SetFrame(x, y, width, height float64) {}
func (m *MygoRich) SetVisible(bool)                      {}

func (m *MygoRich) SetStyle(zoom float64, dark, editable bool) {
	if zoom > 0 {
		m.zoom = zoom
	}
	m.editable = editable
}

func (m *MygoRich) SetFont(body, mono string) { m.body, m.mono = body, mono }
func (m *MygoRich) SetBaseDir(dir string)     { m.codec.BaseDir = dir }

func (m *MygoRich) SetMathNumbering(string) {}
func (m *MygoRich) SetCodeLineNumbers(bool) {}
func (m *MygoRich) SetSpellcheck(bool)      {}
func (m *MygoRich) SetFocusMode(bool)       {}
func (m *MygoRich) SetTypewriter(bool)      {}
func (m *MygoRich) Print()                  {}
func (m *MygoRich) Focus()                  { m.focus = true }
func (m *MygoRich) Blur()                   {}
func (m *MygoRich) Close()                  {}
func (m *MygoRich) ShowFindBar()            { m.openFind(false) }
func (m *MygoRich) ShowFindReplaceBar()     { m.openFind(true) }

// RevealHeading moves the caret to the index-th heading, which the editor
// scrolls into view.
func (m *MygoRich) RevealHeading(index int) {
	off, seen := 0, 0
	for _, p := range m.doc.Text().Paragraphs() {
		if p.Style.Heading > 0 {
			if seen == index {
				m.doc.SetSelection(off, off)
				m.focus = true
				return
			}
			seen++
		}
		for _, r := range p.Runs {
			off += len([]rune(r.Text))
		}
		off++
	}
}

func (m *MygoRich) openFind(replace bool) {
	m.find.open = true
	m.find.replace = m.find.replace || replace
	m.find.focusQuery = true
	if a, z := m.doc.Selection(); a != z {
		if s := []rune(m.doc.Text().String()); max(a, z) <= len(s) && max(a, z)-min(a, z) < 80 {
			if sel := string(s[min(a, z):max(a, z)]); !strings.Contains(sel, "\n") {
				m.find.query = sel
			}
		}
	}
	m.find.lastVersion = 0
}

func foldRunes(s string) []rune {
	r := []rune(s)
	for i, c := range r {
		r[i] = unicode.ToLower(c)
	}
	return r
}

func (m *MygoRich) refreshMatches() {
	f := &m.find
	if f.query == f.lastQuery && f.lastVersion == m.doc.Version() {
		return
	}
	f.lastQuery, f.lastVersion = f.query, m.doc.Version()
	f.matches = f.matches[:0]
	if f.query == "" {
		f.cur = 0
		return
	}
	hay, needle := foldRunes(m.doc.Text().String()), foldRunes(f.query)
	for i := 0; i+len(needle) <= len(hay); i++ {
		if string(hay[i:i+len(needle)]) == string(needle) {
			f.matches = append(f.matches, [2]int{i, i + len(needle)})
			i += len(needle) - 1
		}
	}
	if f.cur >= len(f.matches) {
		f.cur = 0
	}
}

func (m *MygoRich) step(delta int) {
	f := &m.find
	m.refreshMatches()
	if len(f.matches) == 0 {
		return
	}
	f.cur = (f.cur + delta + len(f.matches)) % len(f.matches)
	mt := f.matches[f.cur]
	m.doc.SetSelection(mt[0], mt[1])
}

func (m *MygoRich) replaceCurrent() {
	f := &m.find
	m.refreshMatches()
	if len(f.matches) == 0 || !m.editable {
		return
	}
	mt := f.matches[f.cur]
	m.doc.Edit(func(e *richtext.Edit) { e.ReplaceString(mt[0], mt[1], f.repl, richtext.Attributes{}) })
	m.touched()
	m.refreshMatches()
}

func (m *MygoRich) replaceAll() {
	f := &m.find
	m.refreshMatches()
	if len(f.matches) == 0 || !m.editable {
		return
	}
	matches := append([][2]int(nil), f.matches...)
	m.doc.Edit(func(e *richtext.Edit) {
		for i := len(matches) - 1; i >= 0; i-- {
			e.ReplaceString(matches[i][0], matches[i][1], f.repl, richtext.Attributes{})
		}
	})
	m.touched()
	m.refreshMatches()
}

func (m *MygoRich) findBar(c *ui.Context) {
	f := &m.find
	t := c.Theme()
	m.refreshMatches()
	ui.Column(c).FillWidth().Gap(4).Padding(6, 12).Background(t.Surface).Children(func() {
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
			in := ui.TextInput(c, &f.query).Label("查找").Placeholder("查找").Key("find-query")
			if f.focusQuery {
				in.Focus()
				f.focusQuery = false
			}
			if in.Submitted() {
				m.step(1)
			}
			count := "无结果"
			if len(f.matches) > 0 {
				count = strconv.Itoa(f.cur+1) + " / " + strconv.Itoa(len(f.matches))
			}
			if f.query == "" {
				count = ""
			}
			ui.Text(c, count).FontSize(12).TextColor(t.TextMuted)
			if ui.Button(c, "↑").Key("find-prev").KeepFocus().Padding(2, 8).Clicked() {
				m.step(-1)
			}
			if ui.Button(c, "↓").Key("find-next").KeepFocus().Padding(2, 8).Clicked() {
				m.step(1)
			}
			if m.editable && ui.Button(c, "替换").Key("find-toggle").KeepFocus().Padding(2, 8).Clicked() {
				f.replace = !f.replace
			}
			if ui.Button(c, "✕").Key("find-close").KeepFocus().Padding(2, 8).Clicked() {
				f.open = false
				m.focus = true
			}
		})
		if f.replace && m.editable {
			ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
				in := ui.TextInput(c, &f.repl).Label("替换为").Placeholder("替换为").Key("find-repl")
				if in.Submitted() {
					m.replaceCurrent()
				}
				if ui.Button(c, "替换").Key("repl-one").KeepFocus().Padding(2, 8).Clicked() {
					m.replaceCurrent()
				}
				if ui.Button(c, "全部替换").Key("repl-all").KeepFocus().Padding(2, 8).Clicked() {
					m.replaceAll()
				}
			})
		}
	})
}
