package std_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/doc/doctest"
	"github.com/Maidang1/corra/internal/ext/std"
)

// corpus returns the repository's Markdown files and any listed in
// CORRA_CORPUS.
func corpus(t *testing.T) []string {
	root := filepath.Join("..", "..", "..")
	files := strings.Fields(os.Getenv("CORRA_CORPUS"))
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "spec-mcp" || d.Name() == "dist") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	if len(files) == 0 {
		t.Skip("no corpus")
	}
	return files
}

// TestCorpus reads every Markdown file, and checks that writing it and
// reading it back changes nothing further and loses no text.
func TestCorpus(t *testing.T) {
	k := std.Kit()
	for _, f := range corpus(t) {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		d := k.Codec.Parse(string(data))
		md := k.Codec.Write(d)
		d2 := k.Codec.Parse(md)
		if !doc.SameContent(d, d2) {
			t.Errorf("%s: reading the written Markdown back changes the document\n%s", f, firstDiff(doctest.Dump(d), doctest.Dump(d2)))
			continue
		}
		if md2 := k.Codec.Write(d2); md2 != md {
			t.Errorf("%s: writing is not stable\n%s", f, firstDiff(md, md2))
		}
	}
}

func firstDiff(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	lo := max(0, i-120)
	return "first: …" + a[lo:min(len(a), i+120)] + "\nsecond: …" + b[lo:min(len(b), i+120)]
}
