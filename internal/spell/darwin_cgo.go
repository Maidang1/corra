//go:build darwin && cgo

package spell

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>
#include <stdlib.h>

typedef struct { unsigned long loc, len; } cr_range;

// cr_misspelled checks a whole paragraph at once and returns the ranges of
// its misspelled words, locations in UTF-16 units as NSString counts them,
// malloced; the caller frees. *count is 0 when nothing is misspelled.
static cr_range *cr_misspelled(const char *text, int *count) {
    @autoreleasepool {
        *count = 0;
        NSString *s = [NSString stringWithUTF8String:text];
        if (s == nil) {
            return NULL;
        }
        NSSpellChecker *checker = [NSSpellChecker sharedSpellChecker];
        cr_range *out = NULL;
        NSUInteger cap = 0, pos = 0;
        for (;;) {
            NSRange r = [checker checkSpellingOfString:s startingAt:pos
                language:nil wrap:NO inSpellDocumentWithTag:0 wordCount:NULL];
            if (r.location == NSNotFound || r.length == 0) {
                break;
            }
            if (*count >= cap) {
                cap = cap ? cap * 2 : 16;
                out = realloc(out, cap * sizeof(cr_range));
            }
            out[*count].loc = r.location;
            out[*count].len = r.length;
            (*count)++;
            pos = NSMaxRange(r);
        }
        return out;
    }
}
*/
import "C"

import (
	"unsafe"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// cocoaChecker checks text with the system's NSSpellChecker. AppKit wants
// the main thread, so checks go through mygo.RunOnMain.
type cocoaChecker struct{}

func newChecker() Checker { return &cocoaChecker{} }

// Misspelled reports a paragraph's misspelled words. NSSpellChecker splits
// the words itself and returns UTF-16 ranges, the same units the editor
// works in.
func (cocoaChecker) Misspelled(text string) []ui.TextInputRange {
	if text == "" {
		return nil
	}
	var out []ui.TextInputRange
	mygo.RunOnMain(func() {
		out = cocoaMisspelled(text)
	})
	return out
}

// cocoaMisspelled runs one check, on whatever thread calls it: on the main
// thread, as RunOnMain and the tests arrange.
func cocoaMisspelled(text string) []ui.TextInputRange {
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	var n C.int
	rs := C.cr_misspelled(cs, &n)
	if rs == nil || n == 0 {
		return nil
	}
	defer C.free(unsafe.Pointer(rs))
	cr := unsafe.Slice(rs, int(n))
	out := make([]ui.TextInputRange, 0, int(n))
	for _, r := range cr {
		out = append(out, ui.TextInputRange{Start: int(r.loc), End: int(r.loc) + int(r.len)})
	}
	return out
}
