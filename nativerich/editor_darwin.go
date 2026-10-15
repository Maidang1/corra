//go:build darwin && cgo

package nativerich

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit
#include "bridge.h"
*/
import "C"

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image/color"
	"image/png"
	"runtime/cgo"
	"strings"
	"unsafe"

	"github.com/Maidang1/corra/internal/mdext"
	"github.com/Maidang1/corra/render"
	"github.com/Maidang1/corra/richtext"
)

// Editor methods and callbacks must run on the AppKit main thread.
type Editor struct {
	native                   C.uintptr_t
	handle                   cgo.Handle
	changed, sourceRequested func()
	dirty                    bool
	source                   string

	// OnSaveImage persists (or uploads) a pasted/dropped image and returns
	// the Markdown to insert; ok=false cancels the insertion.
	OnSaveImage func(name string, png []byte) (markdown string, ok bool)
	// OnImportFiles imports dropped files and returns the Markdown to
	// insert (image references, file links); ok=false cancels.
	OnImportFiles func(paths []string) (markdown string, ok bool)
}

func New(window uintptr, changed, sourceRequested func()) *Editor {
	if window == 0 {
		return nil
	}
	e := &Editor{changed: changed, sourceRequested: sourceRequested}
	e.handle = cgo.NewHandle(e)
	e.native = C.cr_new(C.uintptr_t(window), C.uintptr_t(e.handle))
	if e.native == 0 {
		e.handle.Delete()
		return nil
	}
	return e
}

//export cr_callback
func cr_callback(handle C.uintptr_t, source C.int) {
	e := cgo.Handle(handle).Value().(*Editor)
	if source != 0 {
		if e.sourceRequested != nil {
			e.sourceRequested()
		}
	} else {
		e.dirty = true
		if e.changed != nil {
			e.changed()
		}
	}
}

func stringArg(value string) (*C.char, func()) {
	p := C.CString(value)
	return p, func() { C.free(unsafe.Pointer(p)) }
}

func (e *Editor) Load(source string) {
	if e == nil || e.native == 0 {
		return
	}
	data, _ := json.Marshal(richtext.Parse(source))
	p, free := stringArg(string(data))
	defer free()
	C.cr_load(e.native, p)
	e.source, e.dirty = source, false
}

func (e *Editor) Snapshot() (string, bool, error) {
	if e == nil || e.native == 0 {
		return "", false, errors.New("native editor is closed")
	}
	var marked C.int
	p := C.cr_snapshot(e.native, &marked)
	if marked != 0 {
		return e.source, e.dirty, errors.New("请先完成输入法组词，再保存或切换模式")
	}
	if p == nil {
		return e.source, e.dirty, errors.New("native editor serialization failed")
	}
	defer C.free(unsafe.Pointer(p))
	var doc richtext.Document
	if err := json.Unmarshal([]byte(C.GoString(p)), &doc); err != nil {
		return e.source, e.dirty, err
	}
	source := richtext.Serialize(doc)
	changed := e.dirty && source != e.source
	e.source, e.dirty = source, false
	return source, changed, nil
}

func (e *Editor) Format(command string) {
	if e == nil || e.native == 0 {
		return
	}
	p, free := stringArg(command)
	defer free()
	C.cr_format(e.native, p)
}
func (e *Editor) InsertMarkdown(source string) {
	if e == nil || e.native == 0 {
		return
	}
	data, _ := json.Marshal(richtext.Parse(source))
	p, free := stringArg(string(data))
	defer free()
	C.cr_insert(e.native, p)
}
func (e *Editor) SetFrame(x, y, w, h float64) {
	if e != nil && e.native != 0 {
		C.cr_frame(e.native, C.double(x), C.double(y), C.double(w), C.double(h))
	}
}
func boolInt(value bool) C.int {
	if value {
		return 1
	}
	return 0
}
func (e *Editor) SetVisible(visible bool) {
	if e != nil && e.native != 0 {
		C.cr_visible(e.native, boolInt(visible))
	}
}
func (e *Editor) SetBaseDir(dir string) {
	if e == nil || e.native == 0 {
		return
	}
	p, free := stringArg(dir)
	defer free()
	C.cr_base_dir(e.native, p)
}
func (e *Editor) SetStyle(zoom float64, dark, editable bool) {
	if e != nil && e.native != 0 {
		C.cr_style(e.native, C.double(zoom), boolInt(dark), boolInt(editable))
	}
}
func (e *Editor) Focus() {
	if e != nil && e.native != 0 {
		C.cr_focus(e.native, 1)
	}
}
func (e *Editor) Blur() {
	if e != nil && e.native != 0 {
		C.cr_focus(e.native, 0)
	}
}
func (e *Editor) Close() {
	if e != nil && e.native != 0 {
		C.cr_close(e.native)
		e.native = 0
		e.handle.Delete()
	}
}

// SetSpellcheck toggles continuous spell checking (NSTextView).
func (e *Editor) SetSpellcheck(on bool) {
	if e != nil && e.native != 0 {
		C.cr_spellcheck(e.native, boolInt(on))
	}
}

// SetFocusMode dims everything but the caret's paragraph (Typora focus mode).
func (e *Editor) SetFocusMode(on bool) {
	if e != nil && e.native != 0 {
		C.cr_focus_mode(e.native, boolInt(on))
	}
}

// SetTypewriter keeps the caret's line vertically centered while typing.
func (e *Editor) SetTypewriter(on bool) {
	if e != nil && e.native != 0 {
		C.cr_typewriter(e.native, boolInt(on))
	}
}

// ShowFindBar opens the in-editor find bar (Cmd+F in rich mode).
func (e *Editor) ShowFindBar() {
	if e != nil && e.native != 0 {
		C.cr_find(e.native)
	}
}

// RevealHeading scrolls to the index-th heading paragraph.
func (e *Editor) RevealHeading(index int) {
	if e != nil && e.native != 0 {
		C.cr_reveal_heading(e.native, C.int(index))
	}
}

// Print opens the native print panel with the rendered document.
func (e *Editor) Print() {
	if e != nil && e.native != 0 {
		C.cr_print(e.native)
	}
}

//export cr_render_math
func cr_render_math(source *C.char, dark C.int, zoom C.double, inlineMath C.int) *C.char {
	expr := strings.TrimSpace(C.GoString(source))
	expr = strings.TrimPrefix(expr, "$$")
	expr = strings.TrimSuffix(expr, "$$")
	expr = strings.TrimSpace(expr)
	fg := color.RGBA{R: 41, G: 41, B: 41, A: 255}
	if dark != 0 {
		fg = color.RGBA{R: 231, G: 233, B: 238, A: 255}
	}
	img, err := render.RenderMath(expr, inlineMath == 0, 15*float64(zoom), fg)
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	if png.Encode(&buf, img) != nil {
		return nil
	}
	return C.CString(base64.StdEncoding.EncodeToString(buf.Bytes()))
}

//export cr_markdown_json
func cr_markdown_json(value *C.char) *C.char {
	var doc richtext.Document
	if json.Unmarshal([]byte(C.GoString(value)), &doc) != nil {
		return nil
	}
	return C.CString(richtext.Serialize(doc))
}

//export cr_parse_markdown
func cr_parse_markdown(value *C.char) *C.char {
	data, err := json.Marshal(richtext.Parse(C.GoString(value)))
	if err != nil {
		return nil
	}
	return C.CString(string(data))
}

//export cr_emoji
func cr_emoji(word *C.char) *C.char {
	if emoji, ok := mdext.EmojiFor(C.GoString(word)); ok {
		return C.CString(emoji)
	}
	return nil
}

//export cr_paste_image
func cr_paste_image(handle C.uintptr_t, pngBase64, name *C.char) *C.char {
	e := cgo.Handle(handle).Value().(*Editor)
	if e.OnSaveImage == nil {
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(C.GoString(pngBase64))
	if err != nil {
		return nil
	}
	md, ok := e.OnSaveImage(C.GoString(name), data)
	if !ok || md == "" {
		return nil
	}
	return C.CString(md)
}

//export cr_drop_files
func cr_drop_files(handle C.uintptr_t, pathsJSON *C.char) *C.char {
	e := cgo.Handle(handle).Value().(*Editor)
	if e.OnImportFiles == nil {
		return nil
	}
	var paths []string
	if json.Unmarshal([]byte(C.GoString(pathsJSON)), &paths) != nil || len(paths) == 0 {
		return nil
	}
	md, ok := e.OnImportFiles(paths)
	if !ok || md == "" {
		return nil
	}
	return C.CString(md)
}

//export cr_html_to_markdown
func cr_html_to_markdown(html *C.char) *C.char {
	md, err := HTMLToMarkdown(C.GoString(html))
	if err != nil || md == "" {
		return nil
	}
	return C.CString(md)
}
