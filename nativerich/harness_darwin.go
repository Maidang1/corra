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
