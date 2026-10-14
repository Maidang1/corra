package views

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/render"
)

// View renders an app.App and routes input back to it.
type View struct {
	App    *app.App
	Images *app.ImageLoader

	// editBufs holds per-block editable bindings for the live mode.
	editBufs []string
	// anchors remembers, per block, whether it is a heading target.
	jumpTo int // block index to scroll into view this pass, -1 none

	// Modal hosting settings bindings.
	HostOpen  bool
	HostDraft app.HostingConfig

	// pdf export busy flag
	exporting bool

	// invalidate triggers a window redraw (bound at startup).
	invalidate func()

	// Cross-package callbacks, bound from main.
	OnConflictResolve func(reload bool)
	OnSaveHosting     func(cfg app.HostingConfig)
	OnOpenRecent      func(path string)
	OnExportPDF       func()
	OnSave            func() bool
	OnOpen            func()
	OnInsertImage     func()
}

// Bind connects the view to the window's invalidate service.
func (v *View) Bind(svc ui.Services) { v.invalidate = svc.Invalidate }

// New creates the view.
func New(a *app.App, images *app.ImageLoader) *View {
	return &View{App: a, Images: images, jumpTo: -1}
}

// Serve binds window services on first build, then renders.
func (v *View) Serve(c *ui.Context) {
	if v.invalidate == nil {
		svc := c.Services()
		v.Bind(svc)
		v.App.Invalidate(svc)
	}
	v.Build(c)
}

// Build is the MyGo view function.
func (v *View) Build(c *ui.Context) {
	c.SetTheme(v.theme(c))
	v.ensureEditBufs()
	conflict := v.App.Conflict

	ui.Column(c).Fill().Gap(0).Children(func() {
		v.toolbar(c)
		v.findBar(c)
		ui.Row(c).Fill().Gap(0).Children(func() {
			if v.App.Sidebar {
				v.sidebar(c)
				ui.Divider(c).FillHeight()
			}
			if v.App.Mode == app.ModeSource {
				v.sourcePane(c)
			} else {
				v.document(c)
			}
		})
		v.statusBar(c)
	})

	if conflict {
		v.conflictDialog(c)
	}
	if v.App.EditOpen {
		v.blockEditorModal(c)
	}
	if v.HostOpen {
		v.hostingDialog(c)
	}
}

// ensureEditBufs sizes the per-block edit bindings.
func (v *View) ensureEditBufs() {
	n := len(v.App.Doc.Blocks)
	if len(v.editBufs) != n {
		bufs := make([]string, n)
		copy(bufs, v.editBufs)
		for i := range bufs {
			var b render.Block
			if i < len(v.App.Doc.Blocks) {
				b = v.App.Doc.Blocks[i]
			}
			bufs[i] = editValue(&b)
		}
		v.editBufs = bufs
	}
}

// editValue is the text an editable block shows while editing.
func editValue(b *render.Block) string {
	switch b.Kind {
	case render.KindHeading:
		return b.HeadingText()
	case render.KindList:
		// List blocks edit per-item; the whole-block binding is unused.
		return ""
	default:
		return strings.TrimRight(b.Source, "\n")
	}
}

// document renders the single-pane block editor.
func (v *View) document(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Center().Children(func() {
			ui.Column(c).MaxWidth(760).Gap(14).Padding(28, 32, 48, 32).Children(func() {
				for i := range v.App.Doc.Blocks {
					v.block(c, i)
				}
				v.footnotes(c)
				// Click below the content appends a new paragraph.
				ui.Box(c).FillWidth().Height(48).OnClick(func() {
					v.openBlockEditor(len(v.App.Doc.Blocks), "", true)
				})
			})
		})
	})
}

// sourcePane is the whole-document source mode.
func (v *View) sourcePane(c *ui.Context) {
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).FillWidth().Center().Children(func() {
			ui.Column(c).MaxWidth(900).Padding(24, 32, 48, 32).Children(func() {
				ui.TextArea(c, &v.App.Source).Font(fontMono).FontSize(13).
					LineHeight(1.5).OnChange(func() {
					v.App.MarkSourceDirty(v.App.Source)
				})
			})
		})
	})
}

func (v *View) footnotes(c *ui.Context) {
	if len(v.App.Doc.Footnotes) == 0 {
		return
	}
	ui.Column(c).FillWidth().Gap(4).PaddingY(8).Children(func() {
		ui.Divider(c).FillWidth()
		for i, fn := range v.App.Doc.Footnotes {
			idx := i
			ui.RichText(c,
				ui.Span{Text: fmt.Sprintf("[%d] ", idx+1), Weight: 600},
				ui.Span{Text: fn, Color: c.Theme().TextMuted},
			).FontSize(12)
		}
	})
}

func (v *View) statusBar(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).FillWidth().PaddingX(12).PaddingY(6).Gap(12).
		Background(t.Surface).Children(func() {
		name := v.App.Title
		if v.App.Dirty {
			name += " •"
		}
		ui.Text(c, name).FontSize(11.5).TextColor(t.TextMuted)
		ui.Text(c, modeName(v.App.Mode)).FontSize(11.5).TextColor(t.TextMuted)
		ui.Spacer(c)
		ui.Text(c, fmt.Sprintf("%d 个块", len(v.App.Doc.Blocks))).FontSize(11.5).TextColor(t.TextMuted)
	})
}

func modeName(m app.Mode) string {
	if m == app.ModeSource {
		return "源码模式"
	}
	return "单栏编辑"
}
