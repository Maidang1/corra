//go:build !darwin && !windows && !linux

package spell

// newChecker has no system spell checker to call on this platform.
func newChecker() Checker { return nil }
