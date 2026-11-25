//go:build windows

package spell

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/egoist/mygo/ui"
	"golang.org/x/sys/windows"
)

// The Windows Spell Checking API, spellcheck.h. Vtable indexes count the
// IUnknown methods QueryInterface, AddRef and Release as 0, 1 and 2.
var (
	clsidSpellCheckerFactory = windows.GUID{Data1: 0x7ab36653, Data2: 0x1796, Data3: 0x484b,
		Data4: [8]byte{0xbd, 0xfa, 0xe7, 0x4f, 0x1d, 0xb7, 0xc1, 0xdc}}
	iidSpellCheckerFactory = windows.GUID{Data1: 0x8e018a9d, Data2: 0x2415, Data3: 0x4677,
		Data4: [8]byte{0xbf, 0x08, 0x79, 0x4e, 0xa6, 0x1f, 0x94, 0xbb}}
	iidSpellChecker = windows.GUID{Data1: 0xb6fd0b71, Data2: 0xe2bc, Data3: 0x4653,
		Data4: [8]byte{0x8d, 0x05, 0xf1, 0x97, 0xe4, 0x12, 0x77, 0x0b}}

	ole32           = windows.NewLazySystemDLL("ole32.dll")
	procCoCreate    = ole32.NewProc("CoCreateInstance")
	procCoInit      = ole32.NewProc("CoInitializeEx")
	kernel32        = windows.NewLazySystemDLL("kernel32.dll")
	procDefaultName = kernel32.NewProc("GetUserDefaultLocaleName")
)

// COM servers want a home thread, so the checker runs on one of its own:
// requests are served by a goroutine locked to a single-threaded
// apartment, in document order.
type comChecker struct {
	once   sync.Once
	reqs   chan comRequest
	usable bool
}

type comRequest struct {
	text string
	out  chan []ui.TextInputRange
}

func newChecker() Checker { return &comChecker{} }

// Misspelled reports a paragraph's misspelled words. ISpellChecker splits
// the words itself and returns UTF-16 ranges, the units the editor works
// in.
func (c *comChecker) Misspelled(text string) []ui.TextInputRange {
	c.once.Do(c.start)
	if !c.usable {
		return nil
	}
	req := comRequest{text: text, out: make(chan []ui.TextInputRange, 1)}
	c.reqs <- req
	return <-req.out
}

// start brings the worker up and reports whether it found a checker.
func (c *comChecker) start() {
	c.reqs = make(chan comRequest)
	usable := make(chan bool, 1)
	go c.serve(usable)
	c.usable = <-usable
}

func (c *comChecker) serve(usable chan bool) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	checker := comSetup()
	usable <- checker != nil
	if checker == nil {
		return
	}
	defer comRelease(checker)
	for req := range c.reqs {
		req.out <- comCheck(checker, req.text)
	}
}

// comSetup initializes COM on this thread and creates a spell checker for
// the user's language, or English. It returns nil when none could be made.
func comSetup() unsafe.Pointer {
	if hr, _, _ := procCoInit.Call(0, 2 /* COINIT_APARTMENTTHREADED */); hr != 0 && hr != 1 /* S_FALSE */ {
		return nil
	}
	var factory unsafe.Pointer
	hr, _, _ := procCoCreate.Call(
		uintptr(unsafe.Pointer(&clsidSpellCheckerFactory)),
		0, 1, /* CLSCTX_INPROC_SERVER */
		uintptr(unsafe.Pointer(&iidSpellCheckerFactory)),
		uintptr(unsafe.Pointer(&factory)))
	if hr != 0 || factory == nil {
		return nil
	}
	defer comRelease(factory)
	for _, lang := range []string{localeName(), "en-US"} {
		if !comIsSupported(factory, lang) {
			continue
		}
		ws, _ := syscall.UTF16FromString(lang)
		var checker unsafe.Pointer
		hr := comCall(factory, 5, /* CreateSpellChecker */
			uintptr(unsafe.Pointer(&ws[0])), uintptr(unsafe.Pointer(&checker)))
		runtime.KeepAlive(ws)
		if hr == 0 && checker != nil {
			return checker
		}
	}
	return nil
}

// localeName reports the user's locale as a BCP 47 tag, as "en-US".
func localeName() string {
	buf := make([]uint16, 85)
	n, _, _ := procDefaultName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return "en-US"
	}
	return syscall.UTF16ToString(buf)
}

// comMethod returns obj's method pointer at vtable index i. Object
// pointers stay unsafe.Pointer until a call turns them into arguments.
func comMethod(obj unsafe.Pointer, i int) unsafe.Pointer {
	vtbl := *(*unsafe.Pointer)(obj)
	return *(*unsafe.Pointer)(unsafe.Add(vtbl, i*int(unsafe.Sizeof(vtbl))))
}

// comCall runs obj's method index i and returns its HRESULT.
func comCall(obj unsafe.Pointer, i int, args ...uintptr) uintptr {
	hr, _, _ := syscall.SyscallN(uintptr(comMethod(obj, i)), append([]uintptr{uintptr(obj)}, args...)...)
	return hr
}

// comRelease drops a reference to a COM object.
func comRelease(obj unsafe.Pointer) {
	if obj != nil {
		syscall.SyscallN(uintptr(comMethod(obj, 2)), uintptr(obj))
	}
}

// comIsSupported asks the factory whether it has a checker for a language.
func comIsSupported(factory unsafe.Pointer, lang string) bool {
	ws, _ := syscall.UTF16FromString(lang)
	defer runtime.KeepAlive(ws)
	var ok uintptr
	return comCall(factory, 4 /* IsSupported */, uintptr(unsafe.Pointer(&ws[0])), uintptr(unsafe.Pointer(&ok))) == 0 && ok != 0
}

// comCheck returns a paragraph's misspelled ranges.
func comCheck(checker unsafe.Pointer, text string) []ui.TextInputRange {
	ws, _ := syscall.UTF16FromString(text)
	defer runtime.KeepAlive(ws)
	var enum unsafe.Pointer
	if hr := comCall(checker, 4 /* Check */, uintptr(unsafe.Pointer(&ws[0])), uintptr(unsafe.Pointer(&enum))); hr != 0 || enum == nil {
		return nil
	}
	defer comRelease(enum)
	var out []ui.TextInputRange
	for {
		var err unsafe.Pointer
		if hr := comCall(enum, 3 /* Next */, uintptr(unsafe.Pointer(&err))); hr != 0 || err == nil {
			return out
		}
		var start, length uint32
		hrStart := comCall(err, 3 /* get_StartIndex */, uintptr(unsafe.Pointer(&start)))
		hrLen := comCall(err, 4 /* get_Length */, uintptr(unsafe.Pointer(&length)))
		comRelease(err)
		if hrStart == 0 && hrLen == 0 && length > 0 {
			out = append(out, ui.TextInputRange{Start: int(start), End: int(start) + int(length)})
		}
	}
}
