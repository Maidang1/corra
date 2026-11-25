//go:build darwin && !cgo

package spell

// newChecker has no spell checker without cgo: AppKit cannot be called.
func newChecker() Checker { return nil }
