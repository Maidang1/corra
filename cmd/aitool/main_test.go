package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// build once for all sub-tests
var bin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "aitool-test")
	if err != nil {
		panic(err)
	}
	bin = filepath.Join(tmp, "aitool")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		panic(string(out))
	}
	os.Exit(m.Run())
}

func run(t *testing.T, stdin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var so, se strings.Builder
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		return so.String(), se.String(), ee.ExitCode()
	}
	if err != nil {
		t.Fatalf("run %v: %v", args, err)
	}
	return so.String(), se.String(), 0
}

func mustJSON(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, s)
	}
	return m
}

func TestWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.md")

	stdout, _, code := run(t, "# Hello\nworld\n", "write", path)
	if code != 0 {
		t.Fatalf("write exit %d", code)
	}
	res := mustJSON(t, stdout)
	if res["operation"] != "create" || res["bytes"].(float64) != 14 {
		t.Fatalf("unexpected write result: %v", res)
	}

	stdout, _, code = run(t, "", "read", path)
	if code != 0 {
		t.Fatalf("read exit %d", code)
	}
	res = mustJSON(t, stdout)
	if res["totalLines"].(float64) != 2 {
		t.Fatalf("expected 2 lines, got %v", res["totalLines"])
	}
	lines := res["lines"].([]any)
	first := lines[0].(map[string]any)
	if first["number"].(float64) != 1 || first["text"] != "# Hello" {
		t.Fatalf("bad line object: %v", first)
	}

	// overwrite => operation "update"
	stdout, _, code = run(t, "", "write", path, "--content", "x")
	if code != 0 {
		t.Fatalf("rewrite exit %d", code)
	}
	if mustJSON(t, stdout)["operation"] != "update" {
		t.Fatalf("expected update op: %s", stdout)
	}
}

func TestEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.go")
	if _, _, code := run(t, "foo()\n", "write", path); code != 0 {
		t.Fatal("setup write failed")
	}

	stdout, _, code := run(t, "", "edit", path, "--old", "foo()", "--new", "bar()")
	if code != 0 {
		t.Fatalf("edit exit %d", code)
	}
	if mustJSON(t, stdout)["replacements"].(float64) != 1 {
		t.Fatalf("expected 1 replacement: %s", stdout)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "bar()\n" {
		t.Fatalf("file content: %q", data)
	}

	// missing old string => exit 1, JSON error on stderr
	_, stderr, code := run(t, "", "edit", path, "--old", "nope", "--new", "x")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errObj := mustJSON(t, stderr)
	if errObj["ok"] != false {
		t.Fatalf("expected ok=false error: %s", stderr)
	}
}

func TestEditMultiMatchRequiresAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	run(t, "aa aa", "write", path)

	_, _, code := run(t, "", "edit", path, "--old", "aa", "--new", "b")
	if code != 1 {
		t.Fatalf("ambiguous edit should fail, got %d", code)
	}
	_, _, code = run(t, "", "edit", path, "--old", "aa", "--new", "b", "--all")
	if code != 0 {
		t.Fatalf("--all edit should succeed, got %d", code)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "b b" {
		t.Fatalf("content: %q", data)
	}
}

func TestAppendMkdirStatRm(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b")
	if _, _, code := run(t, "", "mkdir", sub); code != 0 {
		t.Fatal("mkdir failed")
	}
	path := filepath.Join(sub, "log.txt")
	run(t, "line1\n", "append", path)
	stdout, _, code := run(t, "line2\n", "append", path)
	if code != 0 {
		t.Fatal("append failed")
	}
	if mustJSON(t, stdout)["operation"] != "append" {
		t.Fatalf("expected append op: %s", stdout)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "line1\nline2\n" {
		t.Fatalf("content: %q", data)
	}

	stdout, _, code = run(t, "", "stat", path)
	if code != 0 || mustJSON(t, stdout)["size"].(float64) != 12 {
		t.Fatalf("stat: %s", stdout)
	}

	if _, _, code = run(t, "", "rm", path); code != 0 {
		t.Fatal("rm failed")
	}
	if _, _, code = run(t, "", "stat", path); code != 1 {
		t.Fatal("stat after rm should fail")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	run(t, "", "mkdir", filepath.Join(dir, "sub"))
	run(t, "x", "write", filepath.Join(dir, "one.txt"))
	run(t, "x", "write", filepath.Join(dir, "sub", "two.txt"))

	stdout, _, code := run(t, "", "list", dir)
	if code != 0 {
		t.Fatal("list failed")
	}
	res := mustJSON(t, stdout)
	entries := res["entries"].([]any)
	if len(entries) != 2 { // one.txt + sub (not recursive)
		t.Fatalf("expected 2 entries: %s", stdout)
	}

	stdout, _, code = run(t, "", "list", dir, "--recursive")
	if code != 0 {
		t.Fatal("recursive list failed")
	}
	entries = mustJSON(t, stdout)["entries"].([]any)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries recursive: %d", len(entries))
	}
}

func TestReadMissingFile(t *testing.T) {
	_, stderr, code := run(t, "", "read", filepath.Join(t.TempDir(), "nope.md"))
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "\"ok\":false") {
		t.Fatalf("expected JSON error, got: %s", stderr)
	}
}

func TestToolsDiscovery(t *testing.T) {
	stdout, _, code := run(t, "", "tools")
	if code != 0 {
		t.Fatal("tools failed")
	}
	res := mustJSON(t, stdout)
	tools, ok := res["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("no tools listed: %s", stdout)
	}
}
func TestCpFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	run(t, "hello cp\n", "write", src)

	dst := filepath.Join(dir, "dst.txt")
	stdout, _, code := run(t, "", "cp", src, dst)
	if code != 0 {
		t.Fatalf("cp exit %d", code)
	}
	res := mustJSON(t, stdout)
	if res["dir"] != false {
		t.Fatalf("expected file copy: %s", stdout)
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "hello cp\n" {
		t.Fatalf("copied content: %q", data)
	}
	// src still exists
	if _, err := os.Stat(src); err != nil {
		t.Fatal("src should remain after cp")
	}

	// cp into an existing directory places the file inside under its base name
	inner := filepath.Join(dir, "inner")
	run(t, "", "mkdir", inner)
	if _, _, code := run(t, "", "cp", src, inner); code != 0 {
		t.Fatal("cp into dir failed")
	}
	data, err := os.ReadFile(filepath.Join(inner, "src.txt"))
	if err != nil || string(data) != "hello cp\n" {
		t.Fatalf("content inside dir: %v %q", err, data)
	}
}

func TestCpDirRequiresRecursive(t *testing.T) {
	dir := t.TempDir()
	tree := filepath.Join(dir, "tree")
	run(t, "", "mkdir", filepath.Join(tree, "sub"))
	run(t, "x", "write", filepath.Join(tree, "a.txt"))
	run(t, "y", "write", filepath.Join(tree, "sub", "b.txt"))

	// without --recursive => exit 1
	_, _, code := run(t, "", "cp", tree, filepath.Join(dir, "copy"))
	if code != 1 {
		t.Fatalf("dir cp without --recursive should fail, got %d", code)
	}

	// with --recursive => whole tree copied
	stdout, _, code := run(t, "", "cp", tree, filepath.Join(dir, "copy"), "--recursive")
	if code != 0 {
		t.Fatalf("recursive cp exit %d", code)
	}
	res := mustJSON(t, stdout)
	if res["files"].(float64) != 2 {
		t.Fatalf("expected 2 files copied: %s", stdout)
	}
	for _, p := range []string{"copy/a.txt", "copy/sub/b.txt"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Fatalf("missing %s", p)
		}
	}
}

func TestMv(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	run(t, "move me\n", "write", src)

	dst := filepath.Join(dir, "b.txt")
	stdout, _, code := run(t, "", "mv", src, dst)
	if code != 0 {
		t.Fatalf("mv exit %d", code)
	}
	if mustJSON(t, stdout)["op"] != "mv" {
		t.Fatalf("bad result: %s", stdout)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatal("src should be gone after mv")
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "move me\n" {
		t.Fatalf("content: %q", data)
	}

	// mv into an existing directory
	inner := filepath.Join(dir, "inner")
	run(t, "", "mkdir", inner)
	if _, _, code := run(t, "", "mv", dst, inner); code != 0 {
		t.Fatal("mv into dir failed")
	}
	if _, err := os.Stat(filepath.Join(inner, "b.txt")); err != nil {
		t.Fatal("b.txt should be inside inner/")
	}

	// mv a directory tree
	tree := filepath.Join(dir, "tree")
	run(t, "", "mkdir", tree)
	run(t, "z", "write", filepath.Join(tree, "f.txt"))
	if _, _, code := run(t, "", "mv", tree, filepath.Join(dir, "moved")); code != 0 {
		t.Fatal("mv dir failed")
	}
	if _, err := os.Stat(filepath.Join(dir, "moved", "f.txt")); err != nil {
		t.Fatal("moved/f.txt missing")
	}
}
