package app

import (
	"os"
	"strings"

	"golang.org/x/sys/windows"
)

// renameNoReplace moves old to next without replacing an existing target.
// MoveFileEx refuses to overwrite unless MOVEFILE_REPLACE_EXISTING is set.
// It takes paths, so each one is resolved from the already-open directory
// handle: a junction swapped in after the library check cannot redirect it.
func renameNoReplace(oldDir *os.File, old string, nextDir *os.File, next string) error {
	from, err := handlePath(oldDir)
	if err != nil {
		return err
	}
	to, err := handlePath(nextDir)
	if err != nil {
		return err
	}
	// Plain concatenation: Clean must not touch the \\?\ prefix, and both
	// names are single path elements.
	from, to = from+`\`+old, to+`\`+next
	fromPtr, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	toPtr, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	if err := windows.MoveFileEx(fromPtr, toPtr, 0); err != nil {
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: err}
	}
	return nil
}

// handlePath returns the final path of an open directory.
func handlePath(dir *os.File) (string, error) {
	buf := make([]uint16, windows.MAX_LONG_PATH)
	// Flags 0: FILE_NAME_NORMALIZED | VOLUME_NAME_DOS, a \\?\C:\... path.
	n, err := windows.GetFinalPathNameByHandle(windows.Handle(dir.Fd()), &buf[0], uint32(len(buf)), 0)
	if err != nil {
		return "", &os.PathError{Op: "GetFinalPathNameByHandle", Path: dir.Name(), Err: err}
	}
	if int(n) > len(buf) {
		return "", &os.PathError{Op: "GetFinalPathNameByHandle", Path: dir.Name(), Err: windows.ERROR_FILENAME_EXCED_RANGE}
	}
	// A drive root ends in a separator; \\?\ paths are not normalized.
	return strings.TrimSuffix(windows.UTF16ToString(buf[:n]), `\`), nil
}
