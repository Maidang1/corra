package app

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type destinationSpan struct {
	start, end int
	node       ast.Node
}

type destinationParser struct {
	parser.InlineParser
	spans []destinationSpan
}

func (p *destinationParser) Parse(parent ast.Node, r text.Reader, context parser.Context) ast.Node {
	line, before := r.PeekLine()
	node := p.InlineParser.Parse(parent, r, context)
	if len(line) == 0 || line[0] != ']' || node == nil {
		return node
	}
	var destination []byte
	switch n := node.(type) {
	case *ast.Link:
		if n.Reference == nil {
			destination = n.Destination
		}
	case *ast.Image:
		if n.Reference == nil {
			destination = n.Destination
		}
	}
	_, after := r.Position()
	if len(destination) > 0 && after.Start > before.Start {
		if offset := bytes.Index(r.Source()[before.Start+1:after.Start], destination); offset >= 0 {
			start := before.Start + 1 + offset
			p.spans = append(p.spans, destinationSpan{start, start + len(destination), node})
		}
	}
	return node
}

func (p *destinationParser) CloseBlock(parent ast.Node, r text.Reader, context parser.Context) {
	p.InlineParser.(parser.CloseBlocker).CloseBlock(parent, r, context)
}

// Parse real Markdown links before changing destination slices. Code blocks,
// code spans, titles and ordinary prose are never searched/replaced wholesale.
func rebaseAttachmentPaths(source, oldDir, newDir string) string {
	if filepath.Clean(oldDir) == filepath.Clean(newDir) {
		return source
	}
	data := []byte(source)
	links := &destinationParser{InlineParser: parser.NewLinkParser()}
	inlines := parser.DefaultInlineParsers()
	for i := range inlines {
		if inlines[i].Priority == 200 {
			inlines[i] = util.Prioritized(links, 200)
		}
	}
	p := parser.NewParser(parser.WithBlockParsers(parser.DefaultBlockParsers()...), parser.WithInlineParsers(inlines...), parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...))
	doc := p.Parse(text.NewReader(data))
	surviving := map[ast.Node]bool{}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		surviving[n] = true
		if ref, ok := n.(*ast.LinkReferenceDefinition); ok && len(ref.Destination) > 0 {
			start := ref.Pos()
			end := ref.Lines().At(ref.Lines().Len() - 1).Stop
			if label := bytes.Index(data[start:end], []byte("]:")); label >= 0 {
				start += label + 2
				if offset := bytes.Index(data[start:end], ref.Destination); offset >= 0 {
					start += offset
					links.spans = append(links.spans, destinationSpan{start, start + len(ref.Destination), n})
				}
			}
		}
		return ast.WalkContinue, nil
	})
	sort.Slice(links.spans, func(i, j int) bool { return links.spans[i].start > links.spans[j].start })
	for _, span := range links.spans {
		if !surviving[span.node] {
			continue
		}
		raw := source[span.start:span.end]
		decoded := html.UnescapeString(string(util.UnescapePunctuations([]byte(raw))))
		u, err := url.Parse(decoded)
		if err != nil || u.Scheme != "" || u.Host != "" || u.Path == "" || strings.HasPrefix(u.Path, "/") || filepath.IsAbs(u.Path) {
			continue
		}
		next, err := filepath.Rel(newDir, filepath.Join(oldDir, filepath.FromSlash(u.Path)))
		if err != nil {
			continue
		}
		u.Path, u.RawPath = filepath.ToSlash(next), ""
		replacement := u.String()
		if span.start > 0 && source[span.start-1] == '<' {
			replacement = strings.ReplaceAll(replacement, "%20", " ")
		}
		source = source[:span.start] + replacement + source[span.end:]
	}
	return source
}

func moveNoteWithAttachments(dir *os.Root, root, source, folder string) (string, error) {
	original, err := dir.Lstat(source)
	if err != nil {
		return "", err
	}
	data, err := dir.ReadFile(source)
	if err != nil {
		return "", err
	}
	rebased := rebaseAttachmentPaths(string(data), filepath.Join(root, filepath.Dir(source)), filepath.Join(root, folder))
	if rebased == string(data) {
		return moveLibraryNote(dir, source, folder)
	}
	for i := 1; ; i++ {
		dest := filepath.Join(folder, numberedName(filepath.Base(source), i))
		f, err := dir.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		committed := false
		defer func() {
			_ = f.Close()
			if !committed {
				_ = dir.Remove(dest)
			}
		}()
		if _, err := f.WriteString(rebased); err != nil {
			return "", err
		}
		if err := f.Chmod(original.Mode().Perm()); err != nil {
			return "", err
		}
		if err := f.Sync(); err != nil {
			return "", err
		}
		if err := f.Close(); err != nil {
			return "", err
		}
		current, err := dir.Lstat(source)
		if err != nil {
			return "", err
		}
		if !os.SameFile(original, current) || original.Size() != current.Size() || !original.ModTime().Equal(current.ModTime()) {
			return "", fmt.Errorf("note changed during move: %s", source)
		}
		if err := dir.Remove(source); err != nil {
			return "", err
		}
		committed = true
		return dest, nil
	}
}
