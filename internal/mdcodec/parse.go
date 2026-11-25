package mdcodec

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/Maidang1/corra/internal/doc"
)

// Parse reads Markdown into a document. Every node gets a new ID. Syntax
// no rule handles stays as raw source, so nothing is lost; the result is
// normalized to the schema.
func (c *Codec) Parse(src string) *doc.Node { return c.parse(src, false) }

// parse reads src; a fragment, not at a document's start, skips the
// preprocessing of what only a document may start with.
func (c *Codec) parse(src string, fragment bool) *doc.Node {
	p := &Parser{c: c}
	p.Open(c.reg.Root(), nil)
	for _, pre := range c.rules.Preprocess {
		if !fragment {
			src = pre(p, src)
		}
	}
	p.src = []byte(src)
	root := c.md.Parser().Parse(text.NewReader(p.src))
	for _, t := range c.rules.Transforms {
		t(root, p.src)
	}
	p.Blocks(root)
	for len(p.stack) > 1 {
		p.Close()
	}
	f := p.stack[0]
	p.closeText()
	d := doc.NewBlock(f.typ, f.attrs, f.kids...)
	return doc.Normalized(d)
}

// Parser builds a document from goldmark nodes. Rules open and close
// blocks and add text; text added to a container opens its default text
// block.
type Parser struct {
	c     *Codec
	src   []byte
	stack []*frame
	marks doc.MarkSet
	// TrimNextSpace drops leading spaces from the next text added, as
	// after a task list's checkbox.
	TrimNextSpace bool
}

type frame struct {
	typ      *doc.NodeType
	attrs    doc.A
	kids     []*doc.Node
	segs     []doc.Segment
	implicit bool
}

func (p *Parser) Source() []byte                 { return p.src }
func (p *Parser) Registry() *doc.Registry        { return p.c.reg }
func (p *Parser) Node(name string) *doc.NodeType { return p.c.reg.Node(name) }
func (p *Parser) Mark(name string) *doc.MarkType { return p.c.reg.Mark(name) }

func (p *Parser) top() *frame { return p.stack[len(p.stack)-1] }

// Open starts a container or text block of type t.
func (p *Parser) Open(t *doc.NodeType, attrs doc.A) {
	if len(p.stack) > 0 {
		p.closeText()
		if p.top().typ.IsTextBlock() {
			// A block inside text, as from an extension: end the text.
			p.Close()
		}
	}
	p.stack = append(p.stack, &frame{typ: t, attrs: attrs})
}

// Close ends the innermost open block and adds it to its parent.
func (p *Parser) Close() {
	p.closeText()
	f := p.top()
	p.stack = p.stack[:len(p.stack)-1]
	var n *doc.Node
	if f.typ.IsTextBlock() {
		n = doc.NewTextBlock(f.typ, f.attrs, doc.NewInline(f.segs...))
	} else {
		n = doc.NewBlock(f.typ, f.attrs, f.kids...)
	}
	p.top().kids = append(p.top().kids, n)
}

// closeText closes a text block opened implicitly by added text.
func (p *Parser) closeText() {
	if len(p.stack) > 1 && p.top().implicit {
		p.Close()
	}
}

// AddNode adds a finished block, as an atom.
func (p *Parser) AddNode(n *doc.Node) {
	p.closeText()
	if p.top().typ.IsTextBlock() {
		p.Close()
	}
	p.top().kids = append(p.top().kids, n)
}

func (p *Parser) textFrame() *frame {
	if !p.top().typ.IsTextBlock() {
		t := p.c.reg.DefaultTextBlock(p.top().typ)
		if t == nil {
			t = p.c.reg.DefaultTextBlock(p.c.reg.Root())
		}
		p.stack = append(p.stack, &frame{typ: t, implicit: true})
	}
	return p.top()
}

// AddText adds text with the marks currently applied.
func (p *Parser) AddText(s string) {
	if p.TrimNextSpace {
		s = strings.TrimLeft(s, " \t")
		if s == "" {
			return
		}
		p.TrimNextSpace = false
	}
	if s == "" {
		return
	}
	f := p.textFrame()
	f.segs = append(f.segs, doc.Segment{Text: s, Marks: p.marks})
}

// AddAtom adds an inline atom with the marks currently applied.
func (p *Parser) AddAtom(n *doc.Node) {
	p.TrimNextSpace = false
	f := p.textFrame()
	f.segs = append(f.segs, doc.Segment{Atom: n, Marks: p.marks})
}

// WithMark applies m to the content fn adds.
func (p *Parser) WithMark(m doc.Mark, fn func()) {
	saved := p.marks
	p.marks = p.marks.Add(m)
	fn()
	p.marks = saved
}

// Blocks converts n's children.
func (p *Parser) Blocks(n ast.Node) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		p.Walk(c)
	}
}

// Inlines converts n's inline children; it is Blocks by another name.
func (p *Parser) Inlines(n ast.Node) { p.Blocks(n) }

// Walk converts one node by its rule, or keeps it as raw source.
func (p *Parser) Walk(n ast.Node) {
	if f, ok := p.c.rules.Nodes[n.Kind()]; ok {
		f(p, n)
		return
	}
	if n.Type() == ast.TypeInline {
		// Unknown inline syntax: keep its content.
		p.Inlines(n)
		return
	}
	p.Raw(n)
}

// Raw adds n's source as a raw block, kept verbatim.
func (p *Parser) Raw(n ast.Node) {
	src := p.BlockSource(n)
	if t := p.c.reg.Node("raw"); t != nil {
		p.AddNode(doc.NewTextBlock(t, nil, doc.Text(src)))
		return
	}
	p.AddText(src)
}

// Sub reads src, a fragment of the document, with the same rules, and
// returns its blocks: for syntax that holds Markdown in raw lines, as a
// footnote definition.
func (p *Parser) Sub(src string) []*doc.Node { return p.c.parse(src, true).Children() }

// Lines returns a leaf block's content lines joined, without the final
// newline.
func (p *Parser) Lines(n ast.Node) string {
	var b bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(p.src))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// BlockSource returns the whole lines of source a block spans, as far as
// its descendants' lines tell.
func (p *Parser) BlockSource(n ast.Node) string {
	start, stop := -1, -1
	var visit func(n ast.Node)
	visit = func(n ast.Node) {
		if n.Type() == ast.TypeBlock {
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				if start < 0 || seg.Start < start {
					start = seg.Start
				}
				if seg.Stop > stop {
					stop = seg.Stop
				}
			}
			if h, ok := n.(*ast.HTMLBlock); ok && h.HasClosure() {
				if seg := h.ClosureLine; seg.Stop > stop {
					stop = seg.Stop
				}
			}
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			visit(c)
		}
	}
	visit(n)
	if start < 0 {
		return ""
	}
	for start > 0 && p.src[start-1] != '\n' {
		start--
	}
	for stop < len(p.src) && p.src[stop-1] != '\n' && p.src[stop] != '\n' {
		stop++ // to the end of the last line
	}
	return strings.TrimRight(string(p.src[start:stop]), "\n")
}

// Text returns the plain, decoded text of an inline subtree, as an
// image's alternative text.
func (p *Parser) Text(n ast.Node) string {
	var b strings.Builder
	var visit func(n ast.Node)
	visit = func(n ast.Node) {
		switch t := n.(type) {
		case *ast.Text:
			v := t.Segment.Value(p.src)
			if !t.IsRaw() {
				v = Decode(v)
			}
			b.Write(v)
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte(' ')
			}
			return
		case *ast.String:
			b.Write(t.Value)
			return
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			visit(c)
		}
	}
	visit(n)
	return b.String()
}

// Decode resolves backslash escapes and entities, as goldmark's renderer
// does for text that is not raw.
func Decode(source []byte) []byte {
	if bytes.IndexByte(source, '\\') < 0 && bytes.IndexByte(source, '&') < 0 {
		return source
	}
	var out bytes.Buffer
	for i := 0; i < len(source); i++ {
		if source[i] == '\\' && i+1 < len(source) && util.IsPunct(source[i+1]) {
			i++
			out.WriteByte(source[i])
			continue
		}
		if source[i] == '&' {
			if end := bytes.IndexByte(source[i:], ';'); end > 0 && end < 40 {
				entity := source[i : i+end+1]
				decoded := util.ResolveEntityNames(util.ResolveNumericReferences(entity))
				if !bytes.Equal(decoded, entity) {
					out.Write(decoded)
					i += end
					continue
				}
			}
		}
		out.WriteByte(source[i])
	}
	return out.Bytes()
}
