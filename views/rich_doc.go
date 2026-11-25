package views

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/ext/std"
	"github.com/Maidang1/corra/nativerich"
)

// DocRich is the new document editor (preview): a doc tree edited by
// internal/editor, drawn by Corra on every platform. Markdown is read
// once on load and written back only after an edit.
type DocRich struct {
	images        *docImages
	lineNumbers   bool
	mathNumbering string
	baseDir       string
	kit           *ext.Kit
	ed            *editor.Editor
	onChange      func()
	dirty         bool
	editable      bool
	zoom          float64
	body          string
	mono          string

	// OnOpenDocument opens a link to another note; it reports success.
	OnOpenDocument func(target string) bool
	// OnLinkRequest asks the app for its insert-link dialog.
	OnLinkRequest func()
	// Attach stores an attachment beside the note (or on the image host)
	// and returns its Markdown, as App.SaveAttachment.
	Attach func(name string, data []byte) (string, error)
	// OnLinkCandidates lists notes matching a query as JSON
	// [{"title","path"}], for "[[" completion.
	OnLinkCandidates func(query string) string

	openURL func(string)

	find struct {
		open, replace bool
		query, repl   string
		count         int
	}
}

// NewDocRich returns the new editor; changed runs after each edit.
// images loads pictures; invalidate asks for a frame from any goroutine.
func NewDocRich(changed func(), images *app.ImageLoader, invalidate func()) *DocRich {
	k := std.Kit()
	d := &DocRich{kit: k, onChange: changed, editable: true, zoom: 1}
	if images != nil {
		d.images = &docImages{loader: images, invalidate: invalidate}
	}
	cfg := k.EditorConfig()
	cfg.Clipboard = &systemClipboard{}
	cfg.HTMLToMarkdown = nativerich.HTMLToMarkdown
	cfg.OnLink = func() {
		if d.OnLinkRequest != nil {
			d.OnLinkRequest()
		}
	}
	cfg.OnOpenLink = func(href string) bool {
		if editor.IsPageTarget(href) {
			return d.OnOpenDocument != nil && d.OnOpenDocument(href)
		}
		if d.openURL != nil && ui.SafeToOpen(href) {
			d.openURL(href)
			return true
		}
		return false
	}
	cfg.Completions = append(cfg.Completions, d.noteLinks())
	cfg.OnPasteImage = func(png []byte) (string, error) {
		if d.Attach == nil {
			return "", errors.New("no attachments")
		}
		return d.Attach("pasted.png", png)
	}
	cfg.OnDropFiles = func(paths []string) string {
		if d.Attach == nil {
			return ""
		}
		var parts []string
		for _, p := range paths {
			data, err := os.ReadFile(filepath.Clean(p))
			if err != nil {
				continue
			}
			if md, err := d.Attach(filepath.Base(p), data); err == nil {
				parts = append(parts, md)
			}
		}
		return strings.Join(parts, "\n\n")
	}
	cfg.OnChange = func(doc.Commit) {
		d.dirty = true
		if d.onChange != nil {
			d.onChange()
		}
	}
	d.ed = editor.New(cfg, k.Codec.Parse(""))
	return d
}

func (d *DocRich) Load(source string) {
	d.ed.SetDoc(d.kit.Codec.Parse(source))
	d.dirty = false
}

// Reload reads source again for the same document, as after editing it
// in source mode: one undoable change that keeps unchanged blocks.
func (d *DocRich) Reload(source string) {
	d.ed.Reload(d.kit.Codec.Parse(source))
	d.dirty = false
}

// Snapshot returns the Markdown after an edit; an untouched document is
// never rewritten.
func (d *DocRich) Snapshot() (string, bool, error) {
	if !d.dirty {
		return "", false, nil
	}
	d.dirty = false
	return d.kit.Codec.Write(d.ed.Doc()), true, nil
}

// formatCommands maps the app's format commands to the editor's.
var formatCommands = map[string]string{
	"bold": "bold", "italic": "italic", "strike": "strikethrough", "code": "code",
	"body": "paragraph", "bullet": "bulletList", "ordered": "orderedList", "checklist": "taskList",
	"quote": "blockquote", "insert:```": "codeBlock", "insert:---": "horizontalRule",
	"heading1": "heading1", "heading2": "heading2", "heading3": "heading3",
	"heading4": "heading4", "heading5": "heading5", "heading6": "heading6",
	"undo": "undo", "redo": "redo",
}

func (d *DocRich) Format(command string) {
	if name, ok := formatCommands[command]; ok {
		d.ed.Command(name)
		return
	}
	switch {
	case strings.HasPrefix(command, "link:"):
		url, text, _ := strings.Cut(strings.TrimPrefix(command, "link:"), "\n")
		if url == "" {
			return
		}
		if sel, ok := d.ed.State().TextSel(); ok && !sel.Empty() && text == "" {
			d.ed.Run(doc.AddMark(doc.NewMark(d.kit.Registry.Mark("link"), doc.A{"href": url})), "format")
			return
		}
		if text == "" {
			text = url
		}
		d.InsertMarkdown("[" + text + "](" + url + ")")
	case command == "insert:$$":
		d.InsertMarkdown("$$\nE = mc^2\n$$")
	case strings.HasPrefix(command, "quote-alert:"):
		d.InsertMarkdown("> [!" + strings.ToUpper(strings.TrimPrefix(command, "quote-alert:")) + "]\n> 内容")
	}
}

// InsertMarkdown inserts source at the caret, read as Markdown.
func (d *DocRich) InsertMarkdown(source string) {
	d.ed.PasteClip(editor.Clip{Markdown: source})
}

func (d *DocRich) SetFrame(x, y, width, height float64) {}
func (d *DocRich) SetVisible(bool)                      {}

func (d *DocRich) SetStyle(zoom float64, dark, editable bool) {
	if zoom > 0 {
		d.zoom = zoom
	}
	d.editable = editable
}

func (d *DocRich) SetFont(body, mono string)    { d.body, d.mono = body, mono }
func (d *DocRich) SetBaseDir(dir string)        { d.baseDir = dir }
func (d *DocRich) SetMathNumbering(mode string) { d.mathNumbering = mode }
func (d *DocRich) SetCodeLineNumbers(on bool)   { d.lineNumbers = on }
func (d *DocRich) SetSpellcheck(on bool)        { d.ed.SetSpellcheck(on) }
func (d *DocRich) SetFocusMode(on bool)         { d.ed.SetFocusMode(on) }
func (d *DocRich) SetTypewriter(on bool)        { d.ed.SetTypewriter(on) }
func (d *DocRich) Print()                       {}
func (d *DocRich) Focus()                       { d.ed.RequestFocus() }
func (d *DocRich) Blur()                        {}
func (d *DocRich) Close()                       {}
func (d *DocRich) ShowFindBar()                 { d.find.open, d.find.replace = true, false }
func (d *DocRich) ShowFindReplaceBar()          { d.find.open, d.find.replace = true, true }

// RevealHeading puts the caret at the start of the index-th heading.
func (d *DocRich) RevealHeading(index int) {
	seen := 0
	var target *doc.Node
	d.ed.Doc().Walk(func(n *doc.Node) bool {
		if target == nil && n.Type().Name() == "heading" {
			if seen == index {
				target = n
			}
			seen++
		}
		return target == nil
	})
	if target != nil {
		d.ed.SetSelection(doc.Caret(doc.Point{Leaf: target.ID()}))
		d.ed.RequestFocus()
	}
}

// Render draws the toolbar, the find bar and the editor.
func (d *DocRich) Render(c *ui.Context) {
	body := fontUI
	if d.body != "" {
		body = d.body + ", " + fontUI
	}
	mono := fontMono
	if d.mono != "" {
		mono = d.mono + ", " + fontMono
	}
	env := editor.Env{
		Theme:   c.Theme(),
		Font:    ui.Font{Family: body, Size: float32(richFontSize * d.zoom), LineHeight: 1.6},
		Mono:    mono,
		Dark:    c.Theme().Dark,
		BaseDir: d.baseDir,

		CodeLineNumbers: d.lineNumbers,
		MathNumbering:   d.mathNumbering,
	}
	d.openURL = c.OpenURL
	env.Pages = docPages{}
	if d.images != nil {
		env.Images = d.images
		if d.images.loaded.Swap(false) {
			d.ed.Relayout()
		}
	}
	ui.Column(c).Fill().Gap(0).Children(func() {
		if d.editable {
			d.toolbar(c)
		}
		if d.find.open {
			d.findBar(c)
		}
		ui.Column(c).Fill().AlignItems(ui.Center).Children(func() {
			ui.Box(c).Fill().MaxWidth(float32(860 * d.zoom)).Children(func() { d.ed.View(c, env) })
		})
	})
}

func (d *DocRich) toolbar(c *ui.Context) {
	buttons := []struct{ label, tip, cmd string }{
		{"B", "粗体", "bold"}, {"I", "斜体", "italic"}, {"S", "删除线", "strikethrough"}, {"</>", "行内代码", "code"},
		{"H1", "一级标题", "heading1"}, {"H2", "二级标题", "heading2"}, {"H3", "三级标题", "heading3"},
		{"•", "无序列表", "bulletList"}, {"1.", "有序列表", "orderedList"}, {"☐", "任务列表", "taskList"},
		{"❝", "引用", "blockquote"}, {"{ }", "代码块", "codeBlock"}, {"—", "分隔线", "horizontalRule"},
	}
	ui.Row(c).FillWidth().Gap(2).Padding(4, 10).Children(func() {
		for _, b := range buttons {
			btn := ui.Button(c, b.label).Key(b.cmd).KeepFocus().Tooltip(b.tip).Padding(2, 8).FontSize(12)
			if !d.ed.Can(b.cmd) {
				btn.Disabled(true)
			}
			if btn.Clicked() {
				d.ed.Command(b.cmd)
			}
		}
		if ui.Button(c, "链接").Key("link").KeepFocus().Tooltip("插入链接").Padding(2, 8).FontSize(12).Clicked() && d.OnLinkRequest != nil {
			d.OnLinkRequest()
		}
		if ui.Button(c, "查找").Key("find").KeepFocus().Tooltip("查找与替换").Padding(2, 8).FontSize(12).Clicked() {
			d.ShowFindReplaceBar()
		}
	})
}

func (d *DocRich) findBar(c *ui.Context) {
	ui.Row(c).FillWidth().Gap(6).Padding(4, 10).AlignItems(ui.Center).Children(func() {
		q := ui.TextInput(c, &d.find.query).Placeholder("查找").Width(200)
		if q.Changed() {
			d.find.count = d.ed.Find(d.find.query)
		}
		if q.Submitted() {
			d.ed.FindNext(false)
		}
		ui.Text(c, matchLabel(d.find.query, d.find.count)).FontSize(12).TextColor(c.Theme().TextMuted)
		if ui.Button(c, "↑").Key("prev").KeepFocus().Clicked() {
			d.ed.FindNext(true)
		}
		if ui.Button(c, "↓").Key("next").KeepFocus().Clicked() {
			d.ed.FindNext(false)
		}
		if d.find.replace {
			ui.TextInput(c, &d.find.repl).Placeholder("替换为").Width(160)
			if ui.Button(c, "替换").Key("replace").KeepFocus().Clicked() {
				d.ed.Replace(d.find.repl)
				d.find.count = len(d.ed.Matches())
			}
			if ui.Button(c, "全部替换").Key("replaceAll").KeepFocus().Clicked() {
				d.ed.ReplaceAll(d.find.repl)
				d.find.count = d.ed.Find(d.find.query)
			}
		}
		ui.Spacer(c)
		if ui.Button(c, "完成").Key("closeFind").Clicked() {
			d.find.open = false
			d.ed.Find("")
			d.ed.RequestFocus()
		}
	})
}

func matchLabel(query string, n int) string {
	if query == "" {
		return ""
	}
	if n == 0 {
		return "无结果"
	}
	return itoa(n) + " 处"
}

// systemClipboard writes plain text, Markdown and HTML to the system
// clipboard, and keeps the editor's own fragment in memory: pasting what
// it copied, unchanged since, gives the fragment back with no loss.
type systemClipboard struct {
	last  editor.Clip
	plain string
}

func (s *systemClipboard) Write(c editor.Clip) {
	s.last, s.plain = c, c.Plain
	items := []transfer.Representation{transfer.Bytes(transfer.Text, []byte(c.Plain))}
	if c.Markdown != "" {
		items = append(items, transfer.Bytes(transfer.Markdown, []byte(c.Markdown)))
	}
	_ = mygo.Clipboard.Write(transfer.New(transfer.NewItem(items...)))
}

func (s *systemClipboard) Read() editor.Clip {
	data, err := mygo.Clipboard.Read(transfer.Markdown, transfer.HTML, transfer.Text, transfer.PNG)
	if err != nil {
		return editor.Clip{}
	}
	var c editor.Clip
	if b, err := data.Read(transfer.Text); err == nil {
		c.Plain = string(b)
	}
	if c.Plain != "" && c.Plain == s.plain {
		return s.last
	}
	if b, err := data.Read(transfer.Markdown); err == nil {
		c.Markdown = string(b)
	}
	if b, err := data.Read(transfer.HTML); err == nil {
		c.HTML = string(b)
	}
	if b, err := data.Read(transfer.PNG); err == nil {
		c.Image = b
	}
	return c
}

// docImages loads the editor's pictures with the app's loader; a picture
// that finishes loading lays the document out again on the next frame.
type docImages struct {
	loader     *app.ImageLoader
	invalidate func()
	loaded     atomic.Bool
}

func (i *docImages) Image(src, baseDir string) (*ui.Bitmap, bool) {
	e := i.loader.Get(src, baseDir, func() {
		i.loaded.Store(true)
		if i.invalidate != nil {
			i.invalidate()
		}
	})
	return e.Result()
}

// docPages reads notes for page cards.
type docPages struct{}

func (docPages) Page(target, baseDir string) (string, string, bool) {
	path, _, _ := strings.Cut(target, "#")
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, filepath.FromSlash(path))
	}
	card := nativerich.PageInfo(path)
	return card.Title, card.Excerpt, !card.Missing
}

// noteLinks is "[[" completion: notes matching what is typed, inserted as
// links.
func (d *DocRich) noteLinks() editor.Completion {
	link := d.kit.Registry.Mark("link")
	type candidate struct {
		Title string `json:"title"`
		Path  string `json:"path"`
	}
	return editor.Completion{
		Trigger:     "[[",
		AnyPosition: true,
		Items: func(q string) []editor.MenuItem {
			if d.OnLinkCandidates == nil {
				return nil
			}
			var cs []candidate
			if json.Unmarshal([]byte(d.OnLinkCandidates(q)), &cs) != nil {
				return nil
			}
			var out []editor.MenuItem
			for _, c := range cs {
				out = append(out, editor.MenuItem{Title: c.Title, Hint: filepath.Base(c.Path), Value: c})
			}
			return out
		},
		Apply: func(it editor.MenuItem) doc.Command {
			c := it.Value.(candidate)
			title := c.Title
			if title == "" {
				title = strings.TrimSuffix(filepath.Base(c.Path), filepath.Ext(c.Path))
			}
			return func(s doc.State, tx *doc.Tx) bool {
				sel, ok := s.TextSel()
				if !ok || link == nil {
					return false
				}
				in := doc.Text(title, doc.NewMark(link, doc.A{"href": c.Path}))
				tx.ReplaceText(sel.Head.Leaf, sel.Head.Offset, sel.Head.Offset, in)
				tx.SetSelection(doc.Caret(doc.Point{Leaf: sel.Head.Leaf, Offset: sel.Head.Offset + in.Len()}))
				// What is typed next is not part of the link.
				tx.SetStoredMarks(nil)
				return true
			}
		},
	}
}
