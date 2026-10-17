// Command aitool is a small, AI-friendly file read/write CLI.
//
// Every command reads and writes files on disk and reports results as
// structured JSON on stdout, with a non-zero exit code and a JSON error
// object on stderr when something fails. This makes it easy for an AI
// agent to shell out to the binary and parse the outcome:
//
//	aitool read path/to/file.md --limit 50
//	aitool write path/to/file.md --content "# Hello"
//	cat big.bin | aitool write path/to/big.bin
//	aitool edit path/to/file.go --old "foo()" --new "bar()" --all
//	aitool tools        # print machine-readable tool schemas
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const version = "0.1.0"

// ---------------------------------------------------------------------------
// output helpers
// ---------------------------------------------------------------------------

func emit(v any) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fail("encode", err)
	}
	fmt.Println(string(out))
}

func fail(op string, err error) {
	if err == nil {
		err = errors.New("unknown error")
	}
	enc, _ := json.Marshal(map[string]any{
		"ok":    false,
		"op":    op,
		"error": err.Error(),
	})
	fmt.Fprintln(os.Stderr, string(enc))
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, `aitool %s — AI-friendly file read/write CLI

Usage:
  aitool <command> [flags]

Commands:
  read <path>              Read a file with line numbers (JSON)
  write <path>             Create or fully replace a file
  append <path>            Append content to a file
  edit <path>              Literal string replacement inside a file
  list <dir>               List files in a directory
  mkdir <path>             Create a directory (parents included)
  stat <path>              Show file/directory metadata
  rm <path>                Remove a file or empty directory
  cp <src> <dst>           Copy a file or directory (--recursive for trees)
  mv <src> <dst>           Move/rename a file or directory
  tools                    Print machine-readable tool descriptions
  version                  Print version

Content flags (write/append):
  --content, -c <string>   Content inline; if omitted, read stdin

Common flags:
  --read: --offset <n> (1-based line), --limit <n>, --raw (raw text)
  --edit: --old <string>, --new <string>, --all (replace every match)
  --list: --recursive, --limit <n>
  --cp:   --recursive (copy directory trees)
  --mv:   (no flags)

All commands print JSON to stdout. Errors print JSON to stderr and
exit non-zero.
`, version)
	os.Exit(2)
}

// normalizeArgs reorders interspersed flags so stdlib flag parsing works:
// all flags first, positionals last. Value flags are detected from the
// FlagSet so "--content x" keeps its value even when it follows the path.
func normalizeArgs(args []string, flg *flag.FlagSet) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if len(a) > 1 && a[0] == '-' {
			flags = append(flags, a)
			name := strings.TrimLeft(a, "-")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				name = name[:eq]
			}
			f := flg.Lookup(name)
			isBool := f != nil && fmt.Sprintf("%T", f.Value) == "*flag.boolValue"
			if !isBool && !strings.Contains(a, "=") && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			pos = append(pos, a)
		}
	}
	return append(flags, pos...)
}

// readContent returns the --content flag value, or stdin when the flag
// was not supplied.
func readContent(content *string) string {
	if *content != "" {
		return *content
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail("read-stdin", err)
	}
	return string(b)
}

// abs returns the absolute path for reporting.
func abs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}

// ---------------------------------------------------------------------------
// commands
// ---------------------------------------------------------------------------

func cmdRead(args []string) {
	fs := flag.NewFlagSet("read", flag.ExitOnError)
	offset := fs.Int("offset", 1, "1-based first line")
	limit := fs.Int("limit", 2000, "max lines")
	raw := fs.Bool("raw", false, "print raw text instead of JSON")
	_ = fs.Parse(normalizeArgs(args, fs))
	if fs.NArg() < 1 {
		fail("read", errors.New("usage: aitool read <path> [--offset n] [--limit n]"))
	}
	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		fail("read", err)
	}
	if *raw {
		fmt.Print(string(data))
		return
	}

	all := strings.Split(string(data), "\n")
	// Trim the phantom empty line produced by a trailing newline.
	if n := len(all); n > 0 && all[n-1] == "" {
		all = all[:n-1]
	}
	start := *offset
	if start < 1 {
		start = 1
	}
	end := start + *limit
	if end > len(all)+1 {
		end = len(all) + 1
	}
	type line struct {
		Number int    `json:"number"`
		Text   string `json:"text"`
	}
	lines := make([]line, 0, end-start)
	for i := start; i < end; i++ {
		lines = append(lines, line{Number: i, Text: all[i-1]})
	}
	emit(map[string]any{
		"ok":         true,
		"op":         "read",
		"path":       abs(path),
		"totalLines": len(all),
		"offset":     start,
		"lines":      lines,
	})
}

func cmdWrite(args []string, appendMode bool) {
	name := "write"
	if appendMode {
		name = "append"
	}
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	content := fs.String("content", "", "content inline (default: read stdin)")
	_ = fs.Parse(normalizeArgs(args, fs))
	if fs.NArg() < 1 {
		fail(name, fmt.Errorf("usage: aitool %s <path> [--content s]", name))
	}
	path := fs.Arg(0)
	body := readContent(content)

	_, statErr := os.Stat(path)
	op := "create"
	if statErr == nil {
		op = "update"
	} else if !os.IsNotExist(statErr) {
		fail(name, statErr)
	}
	if appendMode && statErr == nil {
		op = "append"
	}

	var err error
	if appendMode {
		var f *os.File
		f, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			_, err = f.WriteString(body)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
	} else {
		err = os.WriteFile(path, []byte(body), 0o644)
	}
	if err != nil {
		fail(name, err)
	}
	emit(map[string]any{
		"ok":        true,
		"op":        name,
		"path":      abs(path),
		"operation": op,
		"bytes":     len(body),
	})
}

func cmdEdit(args []string) {
	fs := flag.NewFlagSet("edit", flag.ExitOnError)
	old := fs.String("old", "", "literal text to replace (required)")
	newText := fs.String("new", "", "replacement text")
	all := fs.Bool("all", false, "replace every occurrence")
	_ = fs.Parse(normalizeArgs(args, fs))
	if fs.NArg() < 1 || *old == "" {
		fail("edit", errors.New("usage: aitool edit <path> --old <s> --new <s> [--all]"))
	}
	path := fs.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		fail("edit", err)
	}
	before := string(data)
	count := strings.Count(before, *old)
	if count == 0 {
		fail("edit", fmt.Errorf("old string not found in %s", path))
	}
	if !*all && count > 1 {
		fail("edit", fmt.Errorf("old string appears %d times; pass --all to replace all", count))
	}
	after := strings.ReplaceAll(before, *old, *newText)
	if err := os.WriteFile(path, []byte(after), 0o644); err != nil {
		fail("edit", err)
	}
	emit(map[string]any{
		"ok":           true,
		"op":           "edit",
		"path":         abs(path),
		"replacements": count,
		"bytesBefore":  len(before),
		"bytesAfter":   len(after),
	})
}

func cmdList(args []string) {
	flg := flag.NewFlagSet("list", flag.ExitOnError)
	recursive := flg.Bool("recursive", false, "walk subdirectories")
	limit := flg.Int("limit", 500, "max entries")
	_ = flg.Parse(normalizeArgs(args, flg))
	if flg.NArg() < 1 {
		fail("list", errors.New("usage: aitool list <dir> [--recursive] [--limit n]"))
	}
	root := flg.Arg(0)

	type entry struct {
		Path string `json:"path"`
		Size int64  `json:"size"`
		Dir  bool   `json:"dir"`
	}
	entries := []entry{}
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root {
			var size int64
			if fi, err := d.Info(); err == nil {
				size = fi.Size()
			}
			entries = append(entries, entry{
				Path: path,
				Size: size,
				Dir:  d.IsDir(),
			})
			if len(entries) >= *limit {
				return errLimit
			}
		}
		if !*recursive && path != root && d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	err := filepath.WalkDir(root, walkFn)
	if err != nil && err != errLimit {
		fail("list", err)
	}
	emit(map[string]any{
		"ok":        true,
		"op":        "list",
		"path":      abs(root),
		"entries":   entries,
		"truncated": len(entries) >= *limit,
	})
}

var errLimit = errors.New("entry limit reached")

func cmdMkdir(args []string) {
	fs := flag.NewFlagSet("mkdir", flag.ExitOnError)
	_ = fs.Parse(normalizeArgs(args, fs))
	if fs.NArg() < 1 {
		fail("mkdir", errors.New("usage: aitool mkdir <path>"))
	}
	if err := os.MkdirAll(fs.Arg(0), 0o755); err != nil {
		fail("mkdir", err)
	}
	emit(map[string]any{"ok": true, "op": "mkdir", "path": abs(fs.Arg(0))})
}

func cmdStat(args []string) {
	fs := flag.NewFlagSet("stat", flag.ExitOnError)
	_ = fs.Parse(normalizeArgs(args, fs))
	if fs.NArg() < 1 {
		fail("stat", errors.New("usage: aitool stat <path>"))
	}
	info, err := os.Stat(fs.Arg(0))
	if err != nil {
		fail("stat", err)
	}
	emit(map[string]any{
		"ok":      true,
		"op":      "stat",
		"path":    abs(fs.Arg(0)),
		"size":    info.Size(),
		"dir":     info.IsDir(),
		"mode":    info.Mode().String(),
		"modTime": info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
	})
}

func cmdRm(args []string) {
	fs := flag.NewFlagSet("rm", flag.ExitOnError)
	recursive := fs.Bool("recursive", false, "remove directories and their contents")
	_ = fs.Parse(normalizeArgs(args, fs))
	if fs.NArg() < 1 {
		fail("rm", errors.New("usage: aitool rm <path> [--recursive]"))
	}
	path := fs.Arg(0)
	var err error
	if *recursive {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}
	if err != nil {
		fail("rm", err)
	}
	emit(map[string]any{"ok": true, "op": "rm", "path": abs(path)})
}

// copyFile copies the contents and permission bits of src to dst,
// creating or truncating dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}

// resolveDst applies POSIX-like semantics: when dst is an existing
// directory, src goes inside it under its base name.
func resolveDst(src, dst string) (string, error) {
	info, err := os.Stat(dst)
	if err != nil {
		return dst, nil // does not exist: use as-is
	}
	if info.IsDir() {
		return filepath.Join(dst, filepath.Base(src)), nil
	}
	return dst, nil
}

func cmdCp(args []string) {
	flg := flag.NewFlagSet("cp", flag.ExitOnError)
	recursive := flg.Bool("recursive", false, "copy directory trees")
	_ = flg.Parse(normalizeArgs(args, flg))
	if flg.NArg() < 2 {
		fail("cp", errors.New("usage: aitool cp <src> <dst> [--recursive]"))
	}
	src, dst := flg.Arg(0), flg.Arg(1)

	srcInfo, err := os.Stat(src)
	if err != nil {
		fail("cp", err)
	}
	dst, err = resolveDst(src, dst)
	if err != nil {
		fail("cp", err)
	}

	if !srcInfo.IsDir() {
		if err := copyFile(src, dst); err != nil {
			fail("cp", err)
		}
		emit(map[string]any{
			"ok": true, "op": "cp", "src": abs(src), "dst": abs(dst), "dir": false,
		})
		return
	}

	// directory copy
	if !*recursive {
		fail("cp", fmt.Errorf("%s is a directory; pass --recursive to copy it", src))
	}
	files := 0
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			fi, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, fi.Mode().Perm())
		}
		if err := copyFile(path, target); err != nil {
			return err
		}
		files++
		return nil
	})
	if err != nil {
		fail("cp", err)
	}
	emit(map[string]any{
		"ok": true, "op": "cp", "src": abs(src), "dst": abs(dst), "dir": true, "files": files,
	})
}

func cmdMv(args []string) {
	flg := flag.NewFlagSet("mv", flag.ExitOnError)
	_ = flg.Parse(normalizeArgs(args, flg))
	if flg.NArg() < 2 {
		fail("mv", errors.New("usage: aitool mv <src> <dst>"))
	}
	src := flg.Arg(0)
	dst, err := resolveDst(src, flg.Arg(1))
	if err != nil {
		fail("mv", err)
	}
	if err := os.Rename(src, dst); err != nil {
		// cross-device or otherwise not renameable: fall back to copy+remove
		srcInfo, statErr := os.Stat(src)
		if statErr != nil {
			fail("mv", err) // report the original rename error
		}
		if srcInfo.IsDir() {
			// copy tree then remove
			var files int
			werr := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(src, path)
				if err != nil {
					return err
				}
				target := filepath.Join(dst, rel)
				if d.IsDir() {
					fi, err := d.Info()
					if err != nil {
						return err
					}
					return os.MkdirAll(target, fi.Mode().Perm())
				}
				if err := copyFile(path, target); err != nil {
					return err
				}
				files++
				return nil
			})
			if werr != nil {
				fail("mv", werr)
			}
			if err := os.RemoveAll(src); err != nil {
				fail("mv", err)
			}
			emit(map[string]any{
				"ok": true, "op": "mv", "src": abs(src), "dst": abs(dst), "fallback": "copy+remove", "files": files,
			})
			return
		}
		if err := copyFile(src, dst); err != nil {
			fail("mv", err)
		}
		if err := os.Remove(src); err != nil {
			fail("mv", err)
		}
		emit(map[string]any{
			"ok": true, "op": "mv", "src": abs(src), "dst": abs(dst), "fallback": "copy+remove",
		})
		return
	}
	emit(map[string]any{
		"ok": true, "op": "mv", "src": abs(src), "dst": abs(dst),
	})
}

// ---------------------------------------------------------------------------
// tool schemas: let an AI discover capabilities from the binary itself
// ---------------------------------------------------------------------------

func cmdTools() {
	emit(map[string]any{
		"ok":      true,
		"name":    "aitool",
		"version": version,
		"tools": []map[string]any{
			{
				"name": "read",
				"description": "Read a text file with 1-based line numbers, returned as JSON. " +
					"Use --offset/--limit to page through large files.",
				"args": map[string]any{
					"path":   "file path (required)",
					"offset": "1-based first line (default 1)",
					"limit":  "max lines (default 2000)",
					"raw":    "print raw text instead of JSON",
				},
			},
			{
				"name": "write",
				"description": "Create or fully replace a file. Content comes from --content or stdin. " +
					"Parent directories must exist.",
				"args": map[string]any{
					"path":    "file path (required)",
					"content": "file content (if omitted, stdin is used)",
				},
			},
			{
				"name":        "append",
				"description": "Append content (from --content or stdin) to a file, creating it if needed.",
				"args": map[string]any{
					"path":    "file path (required)",
					"content": "content to append (if omitted, stdin is used)",
				},
			},
			{
				"name": "edit",
				"description": "Replace a literal string inside a file. Fails if the string is missing " +
					"or matches more than once (pass --all to replace every match).",
				"args": map[string]any{
					"path": "file path (required)",
					"old":  "literal text to replace (required, must be unique unless --all)",
					"new":  "replacement text",
					"all":  "replace every occurrence",
				},
			},
			{
				"name":        "list",
				"description": "List files under a directory (optionally recursive), up to a limit.",
				"args": map[string]any{
					"path":      "directory path (required)",
					"recursive": "walk subdirectories (default false)",
					"limit":     "max entries (default 500)",
				},
			},
			{"name": "mkdir", "description": "Create a directory, parents included.", "args": map[string]any{"path": "directory path (required)"}},
			{"name": "stat", "description": "Show file/directory metadata.", "args": map[string]any{"path": "path (required)"}},
			{"name": "rm", "description": "Remove a file or empty directory (--recursive for trees).", "args": map[string]any{"path": "path (required)", "recursive": "remove directory trees"}},
			{
				"name":        "cp",
				"description": "Copy a file or directory to dst. If dst is an existing directory, the source is placed inside it. Directories require --recursive.",
				"args": map[string]any{
					"src":       "source path (required)",
					"dst":       "destination path (required)",
					"recursive": "copy directory trees (default false)",
				},
			},
			{
				"name":        "mv",
				"description": "Move/rename a file or directory to dst. If dst is an existing directory, the source is placed inside it. Falls back to copy+remove across devices.",
				"args": map[string]any{
					"src": "source path (required)",
					"dst": "destination path (required)",
				},
			},
		},
		"notes": "All commands print JSON to stdout; errors print JSON to stderr and exit 1.",
	})
}

// ---------------------------------------------------------------------------

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "read":
		cmdRead(args)
	case "write":
		cmdWrite(args, false)
	case "append":
		cmdWrite(args, true)
	case "edit":
		cmdEdit(args)
	case "list":
		cmdList(args)
	case "mkdir":
		cmdMkdir(args)
	case "stat":
		cmdStat(args)
	case "rm":
		cmdRm(args)
	case "cp":
		cmdCp(args)
	case "mv":
		cmdMv(args)
	case "tools":
		cmdTools()
	case "version":
		fmt.Println(version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		usage()
	}
}
