// Package mdcodec reads Markdown into a doc tree and writes a tree back.
// Every node type and mark brings its own rules through an extension; the
// codec supplies the walking, the writer's line prefixes and escaping, and
// fallbacks that keep anything without rules as raw source.
package mdcodec

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"

	"github.com/Maidang1/corra/internal/doc"
)

// ParseFunc builds doc content for a goldmark node through p.
type ParseFunc func(p *Parser, n ast.Node)

// WriteFunc writes a doc node as Markdown through w.
type WriteFunc func(w *Writer, n *doc.Node)

// MarkSyntax writes a mark's delimiters around the text it covers.
type MarkSyntax struct {
	Open, Close func(m doc.Mark, st Style) string
	// Verbatim marks, as code, wrap each segment on its own and write its
	// text unescaped; Wrap returns the whole written span.
	Verbatim bool
	Wrap     func(m doc.Mark, text string) string
	// Delimited marks, as "**", follow CommonMark's flanking rules; the
	// writer encodes neighboring characters so their delimiters count.
	Delimited bool
	// Alt returns another spelling of the delimiters, used where the
	// usual one would run into a neighbor's ("_" for "*").
	Alt func(m doc.Mark, st Style) (open, close string)
	// PerSegment marks wrap each piece of text on its own: WrapNear gets
	// the escaped text and the characters written just before and after
	// it, so the spelling can depend on them: "^2^", or "<sup>a b</sup>"
	// where a space or a neighboring "^" would break the short form.
	PerSegment bool
	WrapNear   func(m doc.Mark, text string, prev, next rune) string
}

// Rules are one extension's part of the codec.
type Rules struct {
	Extenders     []goldmark.Extender
	BlockParsers  []util.PrioritizedValue
	InlineParsers []util.PrioritizedValue
	// Transforms run over the goldmark tree before it is converted.
	Transforms []func(root ast.Node, src []byte)
	// Preprocess may take a prefix off the source, as front matter, and
	// add what it read to p; it returns the rest.
	Preprocess []func(p *Parser, src string) string

	Nodes  map[ast.NodeKind]ParseFunc
	Writes map[string]WriteFunc
	Marks  map[string]MarkSyntax
	// Escape lists characters the writer escapes in text wherever they
	// appear, as "~" for subscripts or "$" for math.
	Escape string
	// EscapeRuns lists characters escaped where two come together, as "="
	// for "==highlight==".
	EscapeRuns string
	// Loose names block types that always get a blank line before them,
	// even in a tight list, as a thematic break.
	Loose []string
}

// Add merges o into r; o's handlers win where both have one.
func (r *Rules) Add(o Rules) {
	r.Extenders = append(r.Extenders, o.Extenders...)
	r.BlockParsers = append(r.BlockParsers, o.BlockParsers...)
	r.InlineParsers = append(r.InlineParsers, o.InlineParsers...)
	r.Transforms = append(r.Transforms, o.Transforms...)
	r.Preprocess = append(r.Preprocess, o.Preprocess...)
	if r.Nodes == nil {
		r.Nodes = map[ast.NodeKind]ParseFunc{}
	}
	for k, v := range o.Nodes {
		r.Nodes[k] = v
	}
	if r.Writes == nil {
		r.Writes = map[string]WriteFunc{}
	}
	for k, v := range o.Writes {
		r.Writes[k] = v
	}
	if r.Marks == nil {
		r.Marks = map[string]MarkSyntax{}
	}
	for k, v := range o.Marks {
		r.Marks[k] = v
	}
	r.Loose = append(r.Loose, o.Loose...)
	for _, c := range o.EscapeRuns {
		if !containsRune(r.EscapeRuns, c) {
			r.EscapeRuns += string(c)
		}
	}
	for _, c := range o.Escape {
		if !containsRune(r.Escape, c) {
			r.Escape += string(c)
		}
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// Codec converts between Markdown and documents of one schema.
type Codec struct {
	reg   *doc.Registry
	rules Rules
	md    goldmark.Markdown
	Style Style
}

// New returns a codec for reg with rules.
func New(reg *doc.Registry, rules Rules) *Codec {
	opts := []goldmark.Option{goldmark.WithExtensions(rules.Extenders...)}
	var popts []parser.Option
	if len(rules.BlockParsers) > 0 {
		popts = append(popts, parser.WithBlockParsers(rules.BlockParsers...))
	}
	if len(rules.InlineParsers) > 0 {
		popts = append(popts, parser.WithInlineParsers(rules.InlineParsers...))
	}
	if len(popts) > 0 {
		opts = append(opts, goldmark.WithParserOptions(popts...))
	}
	return &Codec{reg: reg, rules: rules, md: goldmark.New(opts...), Style: DefaultStyle()}
}

// Registry returns the codec's schema.
func (c *Codec) Registry() *doc.Registry { return c.reg }
