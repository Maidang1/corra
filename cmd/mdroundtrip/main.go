// Command mdroundtrip reads a Markdown file with the new document model and
// prints how writing it back changes it, as a unified diff, for checking
// that the changes are the accepted normalizations.
//
//	go run ./cmd/mdroundtrip file.md
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Maidang1/corra/internal/ext/std"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: mdroundtrip file.md")
		os.Exit(2)
	}
	src, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	k := std.Kit()
	out := k.Codec.Write(k.Codec.Parse(string(src)))
	dir, err := os.MkdirTemp("", "mdroundtrip")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	written := filepath.Join(dir, "written.md")
	if err := os.WriteFile(written, []byte(out), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cmd := exec.Command("diff", "-u", "--label", "original", "--label", "written", os.Args[1], written)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
