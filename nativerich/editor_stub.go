//go:build !darwin || !cgo

// Package nativerich embeds the platform rich-text editor in a native window.
package nativerich

type Editor struct {
	// OnLinkCandidates mirrors the native editor's "[[" completion hook.
	OnLinkCandidates func(query string) string
}

func New(window uintptr, changed, sourceRequested func()) *Editor { return nil }
func (*Editor) Load(string)                                       {}
func (*Editor) Snapshot() (string, bool, error)                   { return "", false, nil }
func (*Editor) Format(string)                                     {}
func (*Editor) InsertMarkdown(string)                             {}
func (*Editor) SetFrame(x, y, w, h float64)                       {}
func (*Editor) SetVisible(bool)                                   {}
func (*Editor) SetStyle(zoom float64, dark, editable bool)        {}
func (*Editor) SetFont(body, mono string)                         {}
func (*Editor) SetBaseDir(string)                                 {}
func (*Editor) SetMathNumbering(string)                           {}
func (*Editor) SetCodeLineNumbers(bool)                           {}
func (*Editor) SetSpellcheck(bool)                                {}
func (*Editor) SetFocusMode(bool)                                 {}
func (*Editor) SetTypewriter(bool)                                {}
func (*Editor) ShowFindBar()                                      {}
func (*Editor) RevealHeading(int)                                 {}
func (*Editor) Print()                                            {}
func (*Editor) Focus()                                            {}
func (*Editor) Blur()                                             {}
func (*Editor) Close()                                            {}
