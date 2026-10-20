package views

import (
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/Maidang1/corra/app"
)

// panelTransition is how the floating panels — the folders sidebar and the
// outline — arrive and leave: they collapse along the window row's width,
// sliding away like a macOS sidebar rather than snapping in and out.
var panelTransition = ui.ElementTransition{
	Duration: 250 * time.Millisecond,
	Ease:     ui.EaseInOut,
	Enter:    &ui.Motion{Collapse: true},
	Exit:     &ui.Motion{Collapse: true},
}

// workspaceTransition keeps the writing area moving in step with the
// panels: no Enter or Exit, just its position and size animating as a
// panel's collapse makes room for it (or takes it away).
var workspaceTransition = ui.ElementTransition{
	Duration: 250 * time.Millisecond,
	Ease:     ui.EaseInOut,
}

// View renders an app.App and routes input back to it.
type View struct {
	App    *app.App
	Images *app.ImageLoader

	// The editor column's code editor (full Markdown editing). The buffer
	// is the editing surface; syncEditor copies it back into App.Source.
	edState           *editor.State
	edPath            string // note path the buffer was built for
	edBase            string // App.Source the buffer was built from
	edVer             uint64 // buffer version last synced into App.Source
	rich              RichEditor
	richPath          string
	richBase          string
	richLoaded        bool
	richSnapshotError bool

	// autosaveTimer debounces writes of the current note to disk.
	autosaveTimer  *time.Timer
	autosaveDue    time.Time
	autosaveSource string
	SaveError      string
	focusEditor    bool
	focusSearch    bool
	FolderDialog   string
	FolderDraft    string
	FolderOriginal string
	FolderError    string
	DeleteOpen     bool

	// Insert-link modal (Cmd+K).
	LinkOpen bool
	LinkText string
	LinkURL  string

	// Modal hosting settings bindings.
	HostOpen  bool
	HostDraft app.HostingConfig

	// Settings modal (preferences: appearance, fonts, writing aids).
	SettingsOpen bool

	// Export panel: one card per output format (PDF, HTML, pandoc).
	ExportOpen bool

	// Quick switcher (Cmd+Shift+P): filter the library by title or path.
	QuickOpen  bool
	QuickQuery string
	QuickIndex int

	// StatsOpen expands the status bar's document statistics strip.
	StatsOpen bool

	// windowW is the window's width as of the last frame, for the
	// toolbar's compact layout.
	windowW float32

	// invalidate triggers a window redraw (bound at startup).
	invalidate func()

	// Cross-package callbacks, bound from main.
	OnConflictResolve func(reload bool)
	OnSaveHosting     func(cfg app.HostingConfig)
	OnSaveSettings    func()
	OnOpenRecent      func(path string)
	OnExportPDF       func()
	OnExportHTML      func()
	OnExportPandoc    func(label, ext, pandocArg string)
	// PandocPath resolves the pandoc binary for the export panel; nil falls
	// back to app.PandocPath. Tests stub it for a deterministic panel.
	PandocPath func() string
	OnCopyHTML        func() ([]byte, error)
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

func (v *View) Close() {
	if v.autosaveTimer != nil {
		v.autosaveTimer.Stop()
	}
	if v.rich != nil {
		v.rich.Close()
		v.rich = nil
	}
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

// Build is the MyGo view function, laid out the way macOS 26 apps are: the
// navigation sidebar floats as an inset panel and the writing area runs
// full-bleed to the window edges under a borderless toolbar, plus the modal
// dialogs.
func (v *View) Build(c *ui.Context) {
	c.SetTheme(v.theme(c))
	if v.rich != nil && v.richPath != v.App.Path {
		v.App.Mode = app.ModeRich
	}
	if v.syncEditor() {
		v.processAutosave()
	}
	if v.rich != nil {
		v.rich.SetVisible(v.richVisible())
	}
	conflict := v.App.Conflict

	t := c.Theme()

	root := ui.Row(c).Key("window").Fill().Gap(0).Background(t.Surface)
	v.windowW = root.Bounds().W // last frame's width; toggles don't change it
	root.Children(func() {
		if v.App.Sidebar {
			v.foldersSidebar(c)
		}
		// The writing area is the window itself: no card, rim or shadow,
		// and no rule under the toolbar — the page simply starts below it.
		ui.Column(c).Key("workspace").Fill().Gap(0).Background(t.Surface).
			Transition(workspaceTransition).Children(func() {
			v.header(c)
			ui.Row(c).Fill().Gap(0).Children(func() {
				v.editorPane(c)
				if v.App.Outline {
					v.outlinePanel(c)
				}
			})
		})
	})

	if conflict {
		v.conflictDialog(c)
	}
	if v.HostOpen {
		v.hostingDialog(c)
	}
	if v.SettingsOpen {
		v.settingsDialog(c)
	}
	if v.ExportOpen {
		v.exportDialog(c)
	}
	if v.FolderDialog != "" {
		v.folderDialog(c)
	}
	if v.DeleteOpen {
		v.deleteDialog(c)
	}
	if v.LinkOpen {
		v.linkDialog(c)
	}
	if v.QuickOpen {
		v.quickOpenDialog(c)
	}
}
