package doctest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Maidang1/corra/internal/doc"
)

// Dump writes n in the builder's notation, as
// doc(p("a ", b("bold")), ul(li(p("x")))).
func Dump(n *doc.Node) string { return dump(n, nil, false) }

// DumpIDs is Dump with each node's ID, for checking identity.
func DumpIDs(n *doc.Node) string { return dump(n, nil, true) }

// DumpState is Dump with the selection's markers in the text: "|" for a
// caret, "[" and "]" for anchor and head. A node selection appends
// " sel(ids)".
func DumpState(s doc.State) string {
	marks := map[doc.NodeID][]marker{}
	switch sel := s.Sel.(type) {
	case doc.TextSelection:
		if sel.Empty() {
			marks[sel.Head.Leaf] = append(marks[sel.Head.Leaf], marker{'|', sel.Head.Offset})
		} else {
			marks[sel.Anchor.Leaf] = append(marks[sel.Anchor.Leaf], marker{'[', sel.Anchor.Offset})
			marks[sel.Head.Leaf] = append(marks[sel.Head.Leaf], marker{']', sel.Head.Offset})
		}
	}
	out := dump(s.Doc, marks, false)
	if sel, ok := s.Sel.(doc.NodeSelection); ok {
		var names []string
		for _, id := range sel.IDs {
			if n := s.Node(id); n != nil {
				names = append(names, n.Type().Name())
			}
		}
		out += " sel(" + strings.Join(names, ",") + ")"
	}
	return out
}

var shortNames = map[string]string{
	"paragraph": "p", "heading": "h", "blockquote": "quote", "bullet_list": "ul",
	"list_item": "li", "code_block": "pre", "image": "img",
}

func name(t *doc.NodeType) string {
	if s, ok := shortNames[t.Name()]; ok {
		return s
	}
	return t.Name()
}

func dump(n *doc.Node, marks map[doc.NodeID][]marker, ids bool) string {
	var b strings.Builder
	var write func(n *doc.Node)
	write = func(n *doc.Node) {
		b.WriteString(name(n.Type()))
		if ids {
			fmt.Fprintf(&b, "#%d", n.ID())
		}
		b.WriteByte('(')
		var args []string
		for _, k := range n.Attrs().Keys() {
			args = append(args, fmt.Sprintf("%s=%v", k, formatAttr(n.Attr(k))))
		}
		switch n.Type().Content() {
		case doc.ContentBlocks:
			for i := 0; i < n.NumChildren(); i++ {
				var sub strings.Builder
				sub.WriteString(dump(n.Child(i), marks, ids))
				args = append(args, sub.String())
			}
		case doc.ContentInline, doc.ContentText:
			args = append(args, inlineArgs(n.Inline(), marks[n.ID()], ids)...)
		}
		b.WriteString(strings.Join(args, ", "))
		b.WriteByte(')')
	}
	write(n)
	return b.String()
}

func formatAttr(v any) string {
	if s, ok := v.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprint(v)
}

// inlineArgs writes content as quoted strings and mark calls, inserting
// selection markers at their offsets.
func inlineArgs(in doc.Inline, marks []marker, ids bool) []string {
	sort.SliceStable(marks, func(i, j int) bool { return marks[i].off < marks[j].off })
	used := make([]bool, len(marks))
	// at returns the unused markers at offset off.
	at := func(off int) string {
		var b strings.Builder
		for i, m := range marks {
			if !used[i] && m.off == off {
				b.WriteByte(m.kind)
				used[i] = true
			}
		}
		return b.String()
	}
	var args []string
	off := 0
	n := in.NumSegments()
	for i := 0; i < n; i++ {
		s := in.Segment(i)
		var body string
		if s.Atom != nil {
			if pre := at(off); pre != "" {
				args = append(args, fmt.Sprintf("%q", pre))
			}
			switch {
			case s.Atom.Type().Name() != "math":
				body = dump(s.Atom, nil, ids)
			case ids:
				body = fmt.Sprintf("math#%d(%q)", s.Atom.ID(), s.Atom.Attrs().String("tex"))
			default:
				body = fmt.Sprintf("math(%q)", s.Atom.Attrs().String("tex"))
			}
			off++
		} else {
			var b strings.Builder
			for _, r := range s.Text {
				b.WriteString(at(off))
				b.WriteRune(r)
				if r >= 0x10000 {
					off += 2
				} else {
					off++
				}
			}
			if i == n-1 {
				b.WriteString(at(off))
			}
			body = fmt.Sprintf("%q", b.String())
		}
		body = wrapMarks(body, s.Marks)
		args = append(args, body)
	}
	if rest := at(off); rest != "" {
		args = append(args, fmt.Sprintf("%q", rest))
	}
	return args
}

func wrapMarks(body string, marks doc.MarkSet) string {
	for j := len(marks) - 1; j >= 0; j-- {
		m := marks[j]
		switch m.Type.Name() {
		case "bold":
			body = "b(" + body + ")"
		case "italic":
			body = "i(" + body + ")"
		case "code":
			body = "c(" + body + ")"
		case "link":
			body = fmt.Sprintf("link(%q, %s)", m.Attrs.String("href"), body)
		default:
			body = m.String() + "(" + body + ")"
		}
	}
	return body
}
