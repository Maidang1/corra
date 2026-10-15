//go:build darwin && cgo

package render

/*
#cgo LDFLAGS: -framework Foundation -framework CoreText -framework CoreGraphics
#include <stdlib.h>
#include "mathtext_darwin.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func nativeMathTextPath(text string, size float64, bold, italic bool) (string, error) {
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	traits := 0
	if bold {
		traits |= 1
	}
	if italic {
		traits |= 2
	}
	path := C.corraMathTextPath(value, C.double(size), C.int(traits))
	if path == nil {
		return "", fmt.Errorf("math: native font has no outline for %q", text)
	}
	defer C.free(unsafe.Pointer(path))
	return C.GoString(path), nil
}
