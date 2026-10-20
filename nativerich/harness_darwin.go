//go:build darwin && cgo

package nativerich

/*
#include "bridge.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"runtime/cgo"
	"strings"
	"unsafe"

	"github.com/Maidang1/corra/richtext"
)

// The unexported harness operates on real NSTextView instances, without AX or
// synthetic system input. AppKit itself runs on the process's initial thread.
func nativeTestLoop() int     { return int(C.cr_test_loop()) }
func nativeTestExit(code int) { C.cr_test_exit(C.int(code)) }
func nativeTestMain(fn func()) {
	h := cgo.NewHandle(fn)
	defer h.Delete()
	C.cr_test_main(C.uintptr_t(h))
}

//export cr_test_callback
func cr_test_callback(handle C.uintptr_t)  { cgo.Handle(handle).Value().(func())() }
func nativeTestWindow() uintptr            { return uintptr(C.cr_test_window()) }
func nativeTestCloseWindow(window uintptr) { C.cr_test_close_window(C.uintptr_t(window)) }

// Call outside nativeTestMain so AppKit can finish its outer input event.
func nativeTestFinishEvent(e *Editor) bool { return C.cr_test_finish_event(e.native) != 0 }
func nativeTestSelect(e *Editor, start, length int) {
	C.cr_test_select(e.native, C.int(start), C.int(length))
}
func nativeTestSelection(e *Editor) (int, int) {
	var a, b C.int
	C.cr_test_selection(e.native, &a, &b)
	return int(a), int(b)
}
func nativeTestAction(e *Editor, action, text string) {
	a, fa := stringArg(action)
	defer fa()
	s, fs := stringArg(text)
	defer fs()
	C.cr_test_action(e.native, a, s)
}
func nativeTestHasTraits(e *Editor, traits ...string) bool {
	for _, trait := range traits {
		p, free := stringArg(trait)
		ok := C.cr_test_traits(e.native, p)
		free()
		if ok == 0 {
			return false
		}
	}
	return true
}
func nativeTestClickCheckbox(e *Editor) bool       { return C.cr_test_checkbox(e.native) != 0 }
func nativeTestSelectOpaque(e *Editor, whole bool) { C.cr_test_opaque(e.native, boolInt(whole)) }
func nativeTestScreenshot(e *Editor, path string) bool {
	p, free := stringArg(path)
	defer free()
	return C.cr_test_screenshot(e.native, p) != 0
}
func nativeTestLayout(e *Editor) bool { return C.cr_test_layout(e.native) != 0 }

func nativeTestText(e *Editor) string {
	value := C.cr_test_text(e.native)
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}

func nativeTestCodeControls(e *Editor) string {
	value := C.cr_test_code_controls(e.native)
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}

func nativeTestFootnotes(e *Editor) string {
	value := C.cr_test_footnotes(e.native)
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}

func nativeTestImages(e *Editor) string {
	value := C.cr_test_images(e.native)
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}

func nativeTestTOC(e *Editor) string {
	value := C.cr_test_toc(e.native)
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}

// nativeTestSlash reports the slash menu: open state, filter, highlighted row,
// first visible row (the palette scrolls) and entries.
func nativeTestSlash(e *Editor) (open bool, filter string, selected, offset int, items []string) {
	value := C.cr_test_slash(e.native)
	if value == nil {
		return false, "", 0, 0, nil
	}
	defer C.free(unsafe.Pointer(value))
	var state struct {
		Open     bool     `json:"open"`
		Filter   string   `json:"filter"`
		Selected int      `json:"selected"`
		Offset   int      `json:"offset"`
		Items    []string `json:"items"`
	}
	if json.Unmarshal([]byte(C.GoString(value)), &state) != nil {
		return false, "", 0, 0, nil
	}
	return state.Open, state.Filter, state.Selected, state.Offset, state.Items
}

// nativeTestFormatBar reports the floating format bar: visible state and the
// traits active on the selection.
type nativeRect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

func nativeTestFormatBar(e *Editor) (open bool, active []string, frame, window, anchor nativeRect) {
	value := C.cr_test_format_bar(e.native)
	if value == nil {
		return false, nil, nativeRect{}, nativeRect{}, nativeRect{}
	}
	defer C.free(unsafe.Pointer(value))
	var state struct {
		Open   bool       `json:"open"`
		Active []string   `json:"active"`
		Frame  nativeRect `json:"frame"`
		Window nativeRect `json:"window"`
		Anchor nativeRect `json:"anchor"`
	}
	if json.Unmarshal([]byte(C.GoString(value)), &state) != nil {
		return false, nil, nativeRect{}, nativeRect{}, nativeRect{}
	}
	return state.Open, state.Active, state.Frame, state.Window, state.Anchor
}

// nativeTestEmoji reports the emoji menu: open state, filter and names.
func nativeTestEmoji(e *Editor) (open bool, filter string, selected int, items []string) {
	value := C.cr_test_emoji(e.native)
	if value == nil {
		return false, "", 0, nil
	}
	defer C.free(unsafe.Pointer(value))
	var state struct {
		Open     bool     `json:"open"`
		Filter   string   `json:"filter"`
		Selected int      `json:"selected"`
		Items    []string `json:"items"`
	}
	if json.Unmarshal([]byte(C.GoString(value)), &state) != nil {
		return false, "", 0, nil
	}
	return state.Open, state.Filter, state.Selected, state.Items
}

func nativeTestEmojiShot(e *Editor, path string) bool {
	p, free := stringArg(path)
	defer free()
	return C.cr_test_emoji_shot(e.native, p) != 0
}

// nativeTestLink reports the link completion menu.
func nativeTestLink(e *Editor) (open bool, filter string, selected int, items []string) {
	value := C.cr_test_link(e.native)
	if value == nil {
		return false, "", 0, nil
	}
	defer C.free(unsafe.Pointer(value))
	var state struct {
		Open     bool     `json:"open"`
		Filter   string   `json:"filter"`
		Selected int      `json:"selected"`
		Items    []string `json:"items"`
	}
	if json.Unmarshal([]byte(C.GoString(value)), &state) != nil {
		return false, "", 0, nil
	}
	return state.Open, state.Filter, state.Selected, state.Items
}

func nativeTestLinkShot(e *Editor, path string) bool {
	p, free := stringArg(path)
	defer free()
	return C.cr_test_link_shot(e.native, p) != 0
}

// nativeTestHover reports the margin handle and code copy button.
func nativeTestHover(e *Editor) (copyButton, handle bool) {
	value := C.cr_test_hover(e.native)
	if value == nil {
		return false, false
	}
	defer C.free(unsafe.Pointer(value))
	var state struct {
		Copy   bool `json:"copy"`
		Handle bool `json:"handle"`
	}
	if json.Unmarshal([]byte(C.GoString(value)), &state) != nil {
		return false, false
	}
	return state.Copy, state.Handle
}

// nativeTestHoverCode hovers the first code block's copy button; with activate
// it copies the block and reports success.
func nativeTestHoverCode(e *Editor, activate bool) bool {
	return C.cr_test_hover_code(e.native, boolInt(activate)) != 0
}

// nativeTestClickCodeText clicks inside the first code block's text (not the
// copy button) and reports where the caret landed, or -1 when outside it.
func nativeTestClickCodeText(e *Editor) int {
	return int(C.cr_test_click_code_text(e.native))
}

// nativeTestDragBlock drags a paragraph's handle onto another block, dropping
// before or after it.
func nativeTestDragBlock(e *Editor, from, to int, after bool) bool {
	return C.cr_test_drag_block(e.native, C.int(from), C.int(to), boolInt(after)) != 0
}

// nativeTestHoverImage hovers the first image; with buttonIndex it applies
// that size preset (0 = original, 1..4 = 25/50/75/100%).
func nativeTestImageCopy(e *Editor) bool {
	return C.cr_test_image_copy(e.native) != 0
}

func nativeTestImageDelete(e *Editor) bool {
	return C.cr_test_image_delete(e.native) != 0
}

// nativeTestImageReplace swaps the first figure's file for a given path, the
// way the replace button does once the open panel returns.
func nativeTestImageReplace(e *Editor, path string) bool {
	p, free := stringArg(path)
	defer free()
	return C.cr_test_image_replace(e.native, p) != 0
}

// nativeTestClickImageButton sends a real mouse event to a pill button:
// 0..4 are the size presets, 5 copy, 6 delete.
func nativeTestClickImageButton(e *Editor, index int) bool {
	return C.cr_test_click_image_button(e.native, C.int(index)) != 0
}

// nativeTestClickTableButton sends a real mouse event to the hovered table's
// add-row (0) / add-column (1) button.
func nativeTestClickTableButton(e *Editor, kind int) bool {
	return C.cr_test_click_table_button(e.native, C.int(kind)) != 0
}

// nativeTestClipboardImage reports whether the pasteboard holds an image.
func nativeTestClipboardImage() bool {
	return C.cr_test_clipboard_image() != 0
}

func nativeTestHoverTable(e *Editor) bool {
	return C.cr_test_hover_table(e.native) != 0
}

// nativeTestTableButton adds a row (0) or column (1) to the hovered table.
func nativeTestTableButton(e *Editor, kind int) bool {
	return C.cr_test_table_button(e.native, C.int(kind)) != 0
}

func nativeTestHoverImage(e *Editor, buttonIndex int, activate bool) bool {
	return C.cr_test_hover_image(e.native, C.int(buttonIndex), boolInt(activate)) != 0
}

// nativeTestHoverFootnote hovers the first footnote reference.
func nativeTestHoverFootnote(e *Editor) bool {
	return C.cr_test_hover_footnote(e.native) != 0
}

// nativeTestFootnoteCard reports the hover card: open state, heading and body.
func nativeTestFootnoteCard(e *Editor) (open bool, heading, body string) {
	value := C.cr_test_footnote_card(e.native)
	if value == nil {
		return false, "", ""
	}
	defer C.free(unsafe.Pointer(value))
	var state struct {
		Open    bool   `json:"open"`
		Heading string `json:"heading"`
		Body    string `json:"body"`
	}
	if json.Unmarshal([]byte(C.GoString(value)), &state) != nil {
		return false, "", ""
	}
	return state.Open, state.Heading, state.Body
}

func nativeTestFootnoteShot(e *Editor, path string) bool {
	p, free := stringArg(path)
	defer free()
	return C.cr_test_footnote_shot(e.native, p) != 0
}

func nativeTestLanguageHover(e *Editor) bool {
	return C.cr_test_language_hover(e.native) != 0
}

func nativeTestLanguagePick(e *Editor, language string) bool {
	p, free := stringArg(language)
	defer free()
	return C.cr_test_language_pick(e.native, p) != 0
}

func nativeTestDragPreview(e *Editor, from, to int, after bool) bool {
	return C.cr_test_drag_preview(e.native, C.int(from), C.int(to), boolInt(after)) != 0
}

// nativeTestHoverBlock hovers a paragraph's margin handle; with activate it
// selects that block, the way clicking the handle does.
func nativeTestHoverBlock(e *Editor, paragraph int, activate bool) bool {
	return C.cr_test_hover_block(e.native, C.int(paragraph), boolInt(activate)) != 0
}

func nativeTestFormatBarShot(e *Editor, path string) bool {
	p, free := stringArg(path)
	defer free()
	return C.cr_test_format_bar_shot(e.native, p) != 0
}

func nativeTestSlashShot(e *Editor, path string) bool {
	p, free := stringArg(path)
	defer free()
	return C.cr_test_slash_shot(e.native, p) != 0
}

func nativeTestClipboardText() string {
	value := C.cr_test_clipboard_text()
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}

// nativeTestParagraphs lists the native document's paragraphs, one per line,
// for diagnosing a failed round trip.
func nativeTestParagraphs(e *Editor) string {
	var marked C.int
	p := C.cr_snapshot(e.native, &marked)
	if p == nil {
		return fmt.Sprintf("(no snapshot, marked=%d)\n", marked)
	}
	defer C.free(unsafe.Pointer(p))
	var doc richtext.Document
	if err := json.Unmarshal([]byte(C.GoString(p)), &doc); err != nil {
		return "(" + err.Error() + ")\n"
	}
	var out strings.Builder
	for _, q := range doc.Paragraphs {
		fmt.Fprintf(&out, "%s depth=%d quote=%d sep=%q text=%q raw=%q indent=%q\n", q.Kind, q.Depth, q.Quote, q.Separator, richtext.PlainText(q), q.Raw, q.CodeIndent)
	}
	return out.String()
}

func nativeTestMarkers(e *Editor) string {
	value := C.cr_test_markers(e.native)
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}
