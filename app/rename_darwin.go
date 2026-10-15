package app

import (
	"os"

	"golang.org/x/sys/unix"
)

func renameNoReplace(oldDir *os.File, old string, nextDir *os.File, next string) error {
	return unix.RenameatxNp(int(oldDir.Fd()), old, int(nextDir.Fd()), next, unix.RENAME_EXCL)
}
