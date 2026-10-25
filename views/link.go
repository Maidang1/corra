package views

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
)

// LinkCandidates returns note suggestions for the rich editor's "[[" link
// completion as JSON [{"title","path"}], with paths relative to the open note.
func (v *View) LinkCandidates(query string) string {
	notes := v.App.VisibleNotes()
	if len(notes) == 0 {
		return "[]"
	}
	q := strings.ToLower(strings.TrimSpace(query))
	base := filepath.Dir(v.App.Path)
	type candidate struct {
		Title string `json:"title"`
		Path  string `json:"path"`
	}
	out := make([]candidate, 0, 24)
	for _, note := range notes {
		if note.Path == v.App.Path {
			continue
		}
		if q != "" {
			title := strings.ToLower(note.Title)
			name := strings.ToLower(filepath.Base(note.Path))
			if !strings.Contains(title, q) && !strings.Contains(name, q) {
				continue
			}
		}
		rel := note.Path
		if v.App.Path != "" {
			if value, err := filepath.Rel(base, note.Path); err == nil {
				rel = value
			}
		}
		out = append(out, candidate{Title: note.Title, Path: filepath.ToSlash(rel)})
		if len(out) >= 24 {
			break
		}
	}
	data, err := json.Marshal(out)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// LinkDialog state for 插入链接 (Cmd+K).

// openLinkDialog opens the insert-link modal, seeding the selected text.
func (v *View) openLinkDialog() {
	if !v.canEdit() || !v.syncEditor() {
		return
	}
	v.LinkText = ""
	if v.App.Mode == app.ModeEdit && v.edState != nil {
		sel := v.edState.Selection()
		v.LinkText = v.edState.Buffer.Slice(sel.Start(), sel.End())
	}
	v.LinkURL = ""
	v.LinkOpen = true
	v.App.InvalidateUI()
}

// linkDialog renders the insert-link modal.
func (v *View) linkDialog(c *ui.Context) {
	open := true
	ui.Modal(c, &open, func() {
		ui.Column(c).Width(400).Padding(22).Gap(14).Radius(16).Background(c.Theme().Surface).Children(func() {
			ui.Text(c, "插入链接").FontSize(16).Bold()
			ui.TextInput(c, &v.LinkText).Label("链接文本").Placeholder("显示文本（留空则用网址）")
			ui.TextInput(c, &v.LinkURL).Label("链接地址").Placeholder("https://、#标题 或 文件.md").AutoFocus()
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Spacer(c)
				ui.Button(c, "取消").OnClick(func() { v.LinkOpen = false })
				ui.PrimaryButton(c, "确定").Disabled(strings.TrimSpace(v.LinkURL) == "").OnClick(v.applyLink)
			})
		})
	})
	if !open {
		v.LinkOpen = false
	}
}

// applyLink inserts the link into the active editor surface.
func (v *View) applyLink() {
	url := strings.TrimSpace(v.LinkURL)
	text := strings.TrimSpace(v.LinkText)
	if text == "" {
		text = url
	}
	v.LinkOpen = false
	if v.rich != nil && v.App.Mode != app.ModeEdit {
		v.App.Mode = app.ModeRich
		v.ensureRichEditor()
		// With no selection the native editor inserts the text as a link.
		v.rich.Format("link:" + url + "\n" + strings.TrimSpace(v.LinkText))
		v.rich.Focus()
		v.NativeChanged()
		return
	}
	v.App.Mode = app.ModeEdit
	v.ensureEditor()
	sel := v.edState.Selection()
	selected := v.edState.Buffer.Slice(sel.Start(), sel.End())
	if selected != "" {
		text = selected
	}
	v.edState.Insert("[" + text + "](" + url + ")")
	v.syncEditor()
	v.focusEditor = true
	v.App.InvalidateUI()
}
