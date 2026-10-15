//go:build !darwin && !linux

package app

import (
	"fmt"
	"os"
)

func renameNoReplace(_ *os.File, _ string, _ *os.File, _ string) error {
	return fmt.Errorf("safe folder renaming is not supported on this platform")
}
