// Package list is bullet, ordered and task lists.
package list

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
)

func positive(v any) error {
	if n, ok := v.(int); !ok || n < 0 {
		return fmt.Errorf("list start %v", v)
	}
	return nil
}

func Extension() ext.Extension {
	return ext.Extension{
		Name:     "list",
		Priority: 10,
		Nodes: []doc.NodeSpec{
			{Name: "bullet_list", Group: "block list", Content: doc.ContentBlocks, Allow: []string{"list_item"},
				Attrs: map[string]doc.AttrSpec{"tight": {Default: true}}},
			{Name: "ordered_list", Group: "block list", Content: doc.ContentBlocks, Allow: []string{"list_item"},
				Attrs: map[string]doc.AttrSpec{"tight": {Default: true}, "start": {Default: 1, Validate: positive}}},
			// A task item has a checkbox; checked says whether it is ticked.
			{Name: "list_item", Content: doc.ContentBlocks, Allow: []string{"block"}, Defining: true,
				Attrs:     map[string]doc.AttrSpec{"task": {Default: false}, "checked": {Default: false}},
				Normalize: normalizeItem},
		},
		Markdown: mdcodec.Rules{
			Extenders: []goldmark.Extender{extension.TaskList},
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				ast.KindList:          parseList,
				ast.KindListItem:      parseItem,
				east.KindTaskCheckBox: func(p *mdcodec.Parser, n ast.Node) { p.TrimNextSpace = true },
			},
			Writes: map[string]mdcodec.WriteFunc{
				"bullet_list":  writeList,
				"ordered_list": writeList,
			},
		},
		Slash: []editor.SlashItem{
			{Title: "无序列表", Hint: "-", Keywords: "bullet list ul", Command: "bulletList"},
			{Title: "有序列表", Hint: "1.", Keywords: "ordered numbered list ol", Command: "orderedList"},
			{Title: "任务列表", Hint: "[ ]", Keywords: "task todo checklist", Command: "taskList"},
		},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{
				"bulletList":  Toggle(reg, "bullet_list", false),
				"orderedList": Toggle(reg, "ordered_list", false),
				"taskList":    Toggle(reg, "bullet_list", true),
				"indent":      Sink,
				"outdent":     LiftItem,
				"toggleTask":  ToggleTask,
			}
		},
		InputRules: func(reg *doc.Registry) []editor.InputRule {
			return []editor.InputRule{
				editor.PrefixRule(regexp.MustCompile(`^[-*+] $`), func([]string) doc.Command { return Toggle(reg, "bullet_list", false) }),
				editor.PrefixRule(regexp.MustCompile(`^(\d{1,9})[.)] $`), func(m []string) doc.Command {
					start, _ := strconv.Atoi(m[1])
					return doc.Chain(Toggle(reg, "ordered_list", false), setStart(start))
				}),
				editor.PrefixRule(regexp.MustCompile(`^\[( |x)\] $`), func(m []string) doc.Command {
					return doc.Chain(Toggle(reg, "bullet_list", true), func(s doc.State, tx *doc.Tx) bool {
						if m[1] == "x" {
							return ToggleTask(s, tx)
						}
						return true
					})
				}),
			}
		},
		Chains: func(reg *doc.Registry) map[string][]doc.Command {
			return map[string][]doc.Command{
				"Enter":     {SplitItem},
				"Tab":       {Sink},
				"Shift-Tab": {LiftItem},
				"Backspace": {backspaceAtItemStart},
			}
		},
	}
}

// normalizeItem drops the checkbox of a task whose first block is not a
// paragraph: Markdown can only write a checkbox before paragraph text.
func normalizeItem(n *doc.Node, fix *doc.Fixer) {
	task := n.Attrs().Bool("task")
	if task && (n.NumChildren() == 0 || n.Child(0).Type().Name() != "paragraph") {
		fix.SetAttrs(n.ID(), doc.A{"task": false, "checked": false})
	} else if !task && n.Attrs().Bool("checked") {
		fix.SetAttr(n.ID(), "checked", false)
	}
}

func parseList(p *mdcodec.Parser, n ast.Node) {
	l := n.(*ast.List)
	if l.IsOrdered() {
		p.Open(p.Node("ordered_list"), doc.A{"tight": l.IsTight, "start": l.Start})
	} else {
		p.Open(p.Node("bullet_list"), doc.A{"tight": l.IsTight})
	}
	p.Blocks(n)
	p.Close()
}

func parseItem(p *mdcodec.Parser, n ast.Node) {
	attrs := doc.A{}
	if first := n.FirstChild(); first != nil {
		if cb, ok := first.FirstChild().(*east.TaskCheckBox); ok {
			attrs["task"], attrs["checked"] = true, cb.IsChecked
		}
	}
	p.Open(p.Node("list_item"), attrs)
	p.Blocks(n)
	p.Close()
}

// tight reports whether a list can be written tight. Under a single
// newline a paragraph or raw source would continue the block before it,
// and a quote would join a quote before it; after its first block, a tight
// item holds only lists, code, headings and atoms.
func tight(n *doc.Node) bool {
	if !n.Attrs().Bool("tight") {
		return false
	}
	for i := 0; i < n.NumChildren(); i++ {
		item := n.Child(i)
		for j := 1; j < item.NumChildren(); j++ {
			// Only blocks known to start a line on their own may follow
			// under a single newline; other types, extensions' included,
			// make the list loose.
			t := item.Child(j).Type()
			switch {
			case t.InGroup("list"), t.Name() == "code_block", t.Name() == "heading", t.Name() == "horizontal_rule":
			default:
				return false
			}
			// A list may interrupt a paragraph only with a first item that
			// has content and, if numbered, starts at 1.
			// Anything not known to end its own line (an extension's block
			// may be written as paragraph text, as "[TOC]") counts as a
			// paragraph here.
			prev := item.Child(j - 1).Type()
			paragraphLike := !(prev.InGroup("list") || prev.Name() == "code_block" || prev.Name() == "heading" || prev.Name() == "horizontal_rule")
			if t.InGroup("list") && paragraphLike {
				l := item.Child(j)
				if emptyItem(l.Child(0)) || (t.Name() == "ordered_list" && l.Attrs().Int("start") != 1) {
					return false
				}
			}
		}
	}
	return true
}

func writeList(w *mdcodec.Writer, n *doc.Node) {
	ordered := n.Type().Name() == "ordered_list"
	bullet, delim := w.Style.Bullet, w.Style.OrderedDelim
	// Two lists of one kind in a row merge unless their markers differ.
	if prev := w.Prev(); prev != nil && prev.Type() == n.Type() {
		bullet = map[string]string{"-": "*", "*": "-", "+": "-"}[bullet]
		delim = map[string]string{".": ")", ")": "."}[delim]
	}
	t := tight(n)
	w.Tight(t, func() {
		for i := 0; i < n.NumChildren(); i++ {
			item := n.Child(i)
			marker := bullet + " "
			if ordered {
				marker = fmt.Sprintf("%d%s ", n.Attrs().Int("start")+i, delim)
			}
			first := marker
			// A checkbox needs paragraph text after it on its line; with an
			// empty first paragraph and more blocks, the next block would
			// land there instead, so the checkbox is not written.
			lone := item.NumChildren() > 1 && item.Child(0).TextLen() == 0
			if item.Attrs().Bool("task") && !lone {
				if item.Attrs().Bool("checked") {
					first += "[x] "
				} else {
					first += "[ ] "
				}
			}
			w.Section(func() {
				w.WithPrefix(first, strings.Repeat(" ", len(marker)), func() { w.Children(item, t) })
			})
			if emptyItem(item) {
				w.NoBlankAfter()
			}
		}
	})
}

// itemAt returns the list item directly holding the caret's text block,
// its list, and the block's index in the item.
func itemAt(s doc.State) (item, list *doc.Node, index int, ok bool) {
	sel, ok := s.TextSel()
	if !ok {
		return nil, nil, 0, false
	}
	anc := s.Ancestors(sel.Head.Leaf)
	if len(anc) < 3 {
		return nil, nil, 0, false
	}
	item, list = anc[len(anc)-2], anc[len(anc)-3]
	if item.Type().Name() != "list_item" {
		return nil, nil, 0, false
	}
	return item, list, item.ChildIndex(sel.Head.Leaf), true
}

// SplitItem handles Enter in a list item: an empty item leaves the list,
// otherwise the item splits at the caret, taking the blocks after it.
func SplitItem(s doc.State, tx *doc.Tx) bool {
	sel, ok := s.TextSel()
	if !ok || !sel.Empty() {
		return false
	}
	item, list, i, ok := itemAt(s)
	if !ok {
		return false
	}
	leaf := item.Child(i)
	if leaf.Type().Content() == doc.ContentText {
		return false // code inside the item takes the newline
	}
	if leaf.TextLen() == 0 && item.NumChildren() == 1 {
		return LiftItem(s, tx)
	}
	off := sel.Head.Offset
	tail := leaf.Inline().Slice(off, leaf.TextLen())
	rest := item.Children()[i+1:]
	tx.ReplaceText(leaf.ID(), off, leaf.TextLen(), doc.Inline{})
	if len(rest) > 0 {
		tx.ReplaceChildren(item.ID(), i+1, item.NumChildren())
	}
	para := doc.NewTextBlock(leaf.Type(), nil, tail)
	attrs := doc.A{"task": item.Attrs().Bool("task")}
	next := doc.NewBlock(item.Type(), attrs, append([]*doc.Node{para}, rest...)...)
	tx.ReplaceChildren(list.ID(), list.ChildIndex(item.ID())+1, list.ChildIndex(item.ID())+1, next)
	tx.SetSelection(doc.Caret(doc.Point{Leaf: para.ID()}))
	return true
}

// Sink nests the caret's item in the item before it, as Tab does.
func Sink(s doc.State, tx *doc.Tx) bool {
	item, list, _, ok := itemAt(s)
	if !ok {
		return false
	}
	i := list.ChildIndex(item.ID())
	if i == 0 {
		return false
	}
	prev := list.Child(i - 1)
	sel := s.Sel
	tx.ReplaceChildren(list.ID(), i, i+1)
	if last := prev.Child(prev.NumChildren() - 1); last.Type() == list.Type() {
		tx.ReplaceChildren(last.ID(), last.NumChildren(), last.NumChildren(), item)
	} else {
		nested := doc.NewBlock(list.Type(), list.Attrs().Map(), item)
		tx.ReplaceChildren(prev.ID(), prev.NumChildren(), prev.NumChildren(), nested)
	}
	tx.SetSelection(sel)
	return true
}

// LiftItem moves the caret's item out one level, as Shift-Tab does: into
// the enclosing list after its parent item, or out of the list entirely,
// splitting it.
func LiftItem(s doc.State, tx *doc.Tx) bool {
	item, list, _, ok := itemAt(s)
	if !ok {
		return false
	}
	sel := s.Sel
	anc := s.Ancestors(list.ID())
	parent := anc[len(anc)-2]
	i := list.ChildIndex(item.ID())
	after := list.Children()[i+1:]
	if parent.Type().Name() == "list_item" {
		// Nested: the item goes after its parent item, taking the items
		// after it as its own nested list.
		outer := anc[len(anc)-3]
		tx.ReplaceChildren(list.ID(), i, list.NumChildren())
		moved := item
		if len(after) > 0 {
			kids := append(item.Children(), doc.NewBlock(list.Type(), list.Attrs().Map(), after...))
			// Removing item detached its children; they move to a new item.
			moved = doc.NewBlock(item.Type(), item.Attrs().Map(), kids...)
		}
		at := outer.ChildIndex(parent.ID()) + 1
		tx.ReplaceChildren(outer.ID(), at, at, moved)
		removeIfEmpty(tx, list.ID())
	} else {
		// Top level: the item's blocks leave the list, which splits.
		tx.ReplaceChildren(list.ID(), i, list.NumChildren())
		blocks := item.Children()
		if len(after) > 0 {
			blocks = append(blocks, doc.NewBlock(list.Type(), list.Attrs().Map(), after...))
		}
		at := parent.ChildIndex(list.ID()) + 1
		tx.ReplaceChildren(parent.ID(), at, at, blocks...)
		removeIfEmpty(tx, list.ID())
	}
	tx.SetSelection(sel)
	return true
}

func removeIfEmpty(tx *doc.Tx, id doc.NodeID) {
	s := tx.State()
	n := s.Node(id)
	if n == nil || n.NumChildren() > 0 {
		return
	}
	p, _ := s.Parent(id)
	pn := s.Node(p)
	i := pn.ChildIndex(id)
	tx.ReplaceChildren(p, i, i+1)
}

// backspaceAtItemStart lifts an item when Backspace is pressed at the
// start of its first block.
func backspaceAtItemStart(s doc.State, tx *doc.Tx) bool {
	sel, ok := s.TextSel()
	if !ok || !sel.Empty() || sel.Head.Offset != 0 {
		return false
	}
	_, _, i, ok := itemAt(s)
	if !ok || i != 0 {
		return false
	}
	return LiftItem(s, tx)
}

// ToggleTask ticks or unticks the task holding the caret.
func ToggleTask(s doc.State, tx *doc.Tx) bool {
	item, _, _, ok := itemAt(s)
	if !ok || !item.Attrs().Bool("task") {
		return false
	}
	tx.SetAttr(item.ID(), "checked", !item.Attrs().Bool("checked"))
	return true
}

// Toggle wraps the selected blocks in a list of type name, one item each;
// in such a list already, it lifts them out; in another kind of list, it
// changes the list's kind. task makes the items tasks.
func Toggle(reg *doc.Registry, name string, task bool) doc.Command {
	return func(s doc.State, tx *doc.Tx) bool {
		sel, ok := s.TextSel()
		if !ok {
			return false
		}
		t := ext.Node(reg, name)
		if item, list, _, ok := itemAt(s); ok {
			if list.Type() == t && item.Attrs().Bool("task") == task {
				return LiftItem(s, tx)
			}
			if list.Type() != t {
				tx.SetType(list.ID(), t, doc.A{"tight": list.Attrs().Bool("tight")})
			}
			for i := 0; i < list.NumChildren(); i++ {
				tx.SetAttr(list.Child(i).ID(), "task", task)
			}
			return true
		}
		from, to := s.Ordered(sel)
		a, b := s.Ancestors(from.Leaf), s.Ancestors(to.Leaf)
		d := 0
		// Stop above the text blocks: within one block the range is that
		// block in its parent.
		for d < len(a)-2 && d < len(b)-2 && a[d+1].ID() == b[d+1].ID() {
			d++
		}
		parent := a[d]
		if !parent.Type().Allows(t) {
			return false
		}
		i, j := parent.ChildIndex(a[d+1].ID()), parent.ChildIndex(b[d+1].ID())
		blocks := slices.Clone(parent.Children()[i : j+1])
		var items []*doc.Node
		for _, blk := range blocks {
			items = append(items, doc.NewBlock(ext.Node(reg, "list_item"), doc.A{"task": task}, blk))
		}
		tx.ReplaceChildren(parent.ID(), i, j+1, doc.NewBlock(t, nil, items...))
		tx.SetSelection(sel)
		return true
	}
}

// setStart sets the start number of the ordered list holding the caret.
func setStart(start int) doc.Command {
	return func(s doc.State, tx *doc.Tx) bool {
		_, list, _, ok := itemAt(s)
		if !ok {
			return false
		}
		if start != list.Attrs().Int("start") {
			tx.SetAttr(list.ID(), "start", start)
		}
		return true
	}
}

// emptyItem reports whether an item holds nothing but an empty paragraph.
func emptyItem(item *doc.Node) bool {
	return item.NumChildren() == 1 && item.Child(0).Type().IsTextBlock() && item.Child(0).TextLen() == 0
}
