package views

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/Maidang1/corra/app"
)

// View renders an app.App and routes input back to it.
type View struct {
	App    *app.App
	Images *app.ImageLoader

	// The editor column's code editor (full Markdown editing). The buffer
	// is the editing surface; syncEditor copies it back into App.Source.
	edState *editor.State
	edPath  string // note path the buffer was built for
	edBase  string // App.Source the buffer was built from
	edVer   uint64 // buffer version last synced into App.Source

	// autosaveTimer debounces writes of the current note to disk.
	autosaveTimer *time.Timer

	// Modal hosting settings bindings.
	HostOpen  bool
	HostDraft app.HostingConfig

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
	// OnNoteSaved restarts the file watcher after an autosave so the
	// change we wrote ourselves is not reported as external.
	OnNoteSaved func()
}

// Bind connects the view to the window's invalidate service.
func (v *View) Bind(svc ui.Services) { v.invalidate = svc.Invalidate }

// New creates the view.
func New(a *app.App, images *app.ImageLoader) *View {
	return &View{App: a, Images: images}
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

// Build is the MyGo view function: the unified header, the three Notes
// columns (folders, notes list, editor) and the modal dialogs.
func (v *View) Build(c *ui.Context) {
	c.SetTheme(v.theme(c))
	conflict := v.App.Conflict

	ui.Column(c).Fill().Gap(0).Children(func() {
		v.header(c)
		ui.Row(c).Fill().Gap(0).Children(func() {
			v.foldersSidebar(c)
			ui.Divider(c).FillHeight()
			v.notesList(c)
			ui.Divider(c).FillHeight()
			v.editorPane(c)
		})
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
