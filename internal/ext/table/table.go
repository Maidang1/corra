// Package table is GFM tables: a header row, a delimiter row giving each
// column's alignment, and body rows of inline cells.
package table

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdcodec"
)

// Aligns returns a table's column alignments: "l", "c", "r" or "".
func Aligns(n *doc.Node) []string { return strings.Split(n.Attrs().String("align"), ",") }

func Extension() ext.Extension {
	return ext.Extension{
		Name: "table",
		Nodes: []doc.NodeSpec{
			// align lists each column's alignment, comma separated.
			{Name: "table", Group: "block", Content: doc.ContentBlocks, Allow: []string{"table_row"}, Isolating: true,
				Attrs: map[string]doc.AttrSpec{"align": {Default: ""}}, Normalize: normalizeTable},
			{Name: "table_row", Content: doc.ContentBlocks, Allow: []string{"table_cell"}},
			// A cell is one line of inline content; header marks the first
			// row's.
			{Name: "table_cell", Content: doc.ContentInline, Isolating: true,
				Attrs: map[string]doc.AttrSpec{"header": {Default: false}}, Normalize: noBreaks},
		},
		Markdown: mdcodec.Rules{
			Extenders: []goldmark.Extender{extension.Table},
			Nodes: map[ast.NodeKind]mdcodec.ParseFunc{
				east.KindTable: func(p *mdcodec.Parser, n ast.Node) {
					var aligns []string
					for _, a := range n.(*east.Table).Alignments {
						aligns = append(aligns, map[east.Alignment]string{east.AlignLeft: "l", east.AlignCenter: "c", east.AlignRight: "r"}[a])
					}
					p.Open(p.Node("table"), doc.A{"align": strings.Join(aligns, ",")})
					p.Blocks(n)
					p.Close()
				},
				east.KindTableHeader: row(true),
				east.KindTableRow:    row(false),
				east.KindTableCell: func(p *mdcodec.Parser, n ast.Node) {
					header := n.Parent() != nil && n.Parent().Kind() == east.KindTableHeader
					p.Open(p.Node("table_cell"), doc.A{"header": header})
					p.Inlines(n)
					p.Close()
				},
			},
			Writes: map[string]mdcodec.WriteFunc{"table": write},
		},
		Views: map[string]editor.ViewSpec{
			"table": {Container: func(n *doc.Node, env *editor.Env) editor.ContainerLook {
				return editor.ContainerLook{Before: env.Font.Size * 0.4, After: env.Font.Size, Paint: func(p *ui.Painter, r ui.Rect, env *editor.Env) {
					p.Stroke(r, env.Theme.Border, 6, 1)
				}}
			}},
			"table_row": {Container: func(n *doc.Node, env *editor.Env) editor.ContainerLook {
				cols := n.NumChildren()
				header := cols > 0 && n.Child(0).Attrs().Bool("header")
				return editor.ContainerLook{Horizontal: true, Paint: func(p *ui.Painter, r ui.Rect, env *editor.Env) {
					if header {
						// Inside the table's rounded rim.
						p.FillRadii(ui.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 1}, [4]float32{5, 5, 0, 0}, env.Tint())
					}
					p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, env.Theme.Border, 0)
					for i := 1; i < cols; i++ {
						x := r.X + r.W*float32(i)/float32(cols)
						p.Fill(ui.Rect{X: x, Y: r.Y, W: 1, H: r.H}, env.Theme.Border, 0)
					}
				}}
			}},
			"table_cell": {Text: func(n *doc.Node, env *editor.Env) editor.TextLook {
				f := env.Font
				f.LineHeight = 1.5
				if n.Attrs().Bool("header") {
					f.Weight = 600
				}
				return editor.TextLook{Font: f, Color: env.Theme.Text, PadX: 10, PadY: 6}
			}},
		},
		Slash: []editor.SlashItem{{Title: "表格", Keywords: "table grid", Command: "table"}},
		Commands: func(reg *doc.Registry) map[string]doc.Command {
			return map[string]doc.Command{
				"table":          Insert(reg, 3, 3),
				"tableRowBelow":  AddRow(reg, 1),
				"tableRowAbove":  AddRow(reg, 0),
				"tableColRight":  AddColumn(reg, 1),
				"tableColLeft":   AddColumn(reg, 0),
				"tableDeleteRow": DeleteRow,
				"tableDeleteCol": DeleteColumn,
			}
		},
		Chains: func(reg *doc.Registry) map[string][]doc.Command {
			return map[string][]doc.Command{
				"Tab":       {NextCell(reg, 1)},
				"Shift-Tab": {NextCell(reg, -1)},
				"Enter":     {CellBelow(reg)},
			}
		},
	}
}

func row(header bool) mdcodec.ParseFunc {
	return func(p *mdcodec.Parser, n ast.Node) {
		p.Open(p.Node("table_row"), nil)
		p.Blocks(n)
		p.Close()
	}
}

// normalizeTable gives every row as many cells as the widest, marks the
// first row's cells as the header, and keeps one alignment per column.
func normalizeTable(n *doc.Node, fix *doc.Fixer) {
	reg := n.Type().Registry()
	// Rules run before the schema's repairs: wait until the table holds
	// only rows of cells.
	for i := 0; i < n.NumChildren(); i++ {
		r := n.Child(i)
		if r.Type().Name() != "table_row" {
			return
		}
		for j := 0; j < r.NumChildren(); j++ {
			if r.Child(j).Type().Name() != "table_cell" {
				return
			}
		}
	}
	cols := 0
	for i := 0; i < n.NumChildren(); i++ {
		cols = max(cols, n.Child(i).NumChildren())
	}
	for i := 0; i < n.NumChildren(); i++ {
		r := n.Child(i)
		if missing := cols - r.NumChildren(); missing > 0 {
			var cells []*doc.Node
			for j := 0; j < missing; j++ {
				cells = append(cells, doc.NewTextBlock(reg.Node("table_cell"), doc.A{"header": i == 0}, doc.Inline{}))
			}
			fix.ReplaceChildren(r.ID(), r.NumChildren(), r.NumChildren(), cells...)
			return
		}
		for j := 0; j < r.NumChildren(); j++ {
			if c := r.Child(j); c.Attrs().Bool("header") != (i == 0) {
				fix.SetAttr(c.ID(), "header", i == 0)
				return
			}
		}
	}
	aligns := Aligns(n)
	if len(aligns) != cols {
		aligns = append(aligns, make([]string, max(0, cols-len(aligns)))...)[:cols]
		fix.SetAttr(n.ID(), "align", strings.Join(aligns, ","))
	}
}

// noBreaks turns line breaks in a cell into spaces: a cell is one line.
func noBreaks(n *doc.Node, fix *doc.Fixer) {
	in := n.Inline()
	for i, off := 0, 0; i < in.NumSegments(); i++ {
		seg := in.Segment(i)
		if seg.Atom != nil && seg.Atom.Type().Name() == "hard_break" {
			fix.ReplaceText(n.ID(), off, off+1, doc.Text(" ", seg.Marks...))
			return
		}
		if seg.Atom != nil {
			off++
		} else {
			off += doc.UTF16Len(seg.Text)
		}
	}
}

func write(w *mdcodec.Writer, n *doc.Node) {
	aligns := Aligns(n)
	line := func(cells []string) {
		w.Raw("| " + strings.Join(cells, " | ") + " |")
		w.Newline()
	}
	for i := 0; i < n.NumChildren(); i++ {
		r := n.Child(i)
		var cells []string
		for j := 0; j < r.NumChildren(); j++ {
			// A cell is one line, and "|" ends it: escape it, in code too
			// (GFM reads "\|" back as "|" there).
			s := strings.ReplaceAll(w.InlineString(r.Child(j).Inline()), "|", `\|`)
			cells = append(cells, strings.ReplaceAll(s, "\n", " "))
		}
		line(cells)
		if i == 0 {
			var delims []string
			for j := range cells {
				a := ""
				if j < len(aligns) {
					a = aligns[j]
				}
				delims = append(delims, map[string]string{"l": ":---", "c": ":---:", "r": "---:"}[a])
				if delims[j] == "" {
					delims[j] = "---"
				}
			}
			line(delims)
		}
	}
}

// cellAt returns the cell holding the caret, its row and table, and the
// indexes of the row and the column.
func cellAt(s doc.State) (cell, row, table *doc.Node, r, c int, ok bool) {
	sel, ok := s.TextSel()
	if !ok {
		return nil, nil, nil, 0, 0, false
	}
	anc := s.Ancestors(sel.Head.Leaf)
	if len(anc) < 3 || anc[len(anc)-1].Type().Name() != "table_cell" {
		return nil, nil, nil, 0, 0, false
	}
	cell, row, table = anc[len(anc)-1], anc[len(anc)-2], anc[len(anc)-3]
	return cell, row, table, table.ChildIndex(row.ID()), row.ChildIndex(cell.ID()), true
}

func emptyRow(reg *doc.Registry, cols int) *doc.Node {
	var cells []*doc.Node
	for i := 0; i < cols; i++ {
		cells = append(cells, doc.NewTextBlock(reg.Node("table_cell"), nil, doc.Inline{}))
	}
	return doc.NewBlock(reg.Node("table_row"), nil, cells...)
}

// Insert inserts an empty table of rows by cols, the first row its header.
func Insert(reg *doc.Registry, rows, cols int) doc.Command {
	return func(s doc.State, tx *doc.Tx) bool {
		var rs []*doc.Node
		for i := 0; i < rows; i++ {
			rs = append(rs, emptyRow(reg, cols))
		}
		t := doc.NewBlock(reg.Node("table"), doc.A{"align": strings.Repeat(",", cols-1)}, rs...)
		return doc.InsertBlocks(t)(s, tx)
	}
}

// NextCell moves to the next cell (dir 1) or the previous one (-1); after
// the last cell it adds a row.
func NextCell(reg *doc.Registry, dir int) doc.Command {
	return func(s doc.State, tx *doc.Tx) bool {
		cell, _, table, _, _, ok := cellAt(s)
		if !ok {
			return false
		}
		var cells []*doc.Node
		table.Walk(func(n *doc.Node) bool {
			if n.Type().Name() == "table_cell" {
				cells = append(cells, n)
			}
			return true
		})
		i := 0
		for j, c := range cells {
			if c == cell {
				i = j
			}
		}
		switch next := i + dir; {
		case next < 0:
			return true
		case next >= len(cells):
			r := emptyRow(reg, table.Child(0).NumChildren())
			tx.ReplaceChildren(table.ID(), table.NumChildren(), table.NumChildren(), r)
			tx.SetSelection(doc.Caret(doc.Point{Leaf: r.Child(0).ID()}))
		default:
			c := cells[next]
			tx.SetSelection(doc.TextSelection{Anchor: doc.Point{Leaf: c.ID()}, Head: doc.Point{Leaf: c.ID(), Offset: c.TextLen()}})
		}
		return true
	}
}

// CellBelow moves to the cell below, adding a row after the last.
func CellBelow(reg *doc.Registry) doc.Command {
	return func(s doc.State, tx *doc.Tx) bool {
		_, _, table, r, c, ok := cellAt(s)
		if !ok {
			return false
		}
		if r+1 < table.NumChildren() {
			below := table.Child(r + 1).Child(c)
			tx.SetSelection(doc.Caret(doc.Point{Leaf: below.ID(), Offset: below.TextLen()}))
			return true
		}
		row := emptyRow(reg, table.Child(0).NumChildren())
		tx.ReplaceChildren(table.ID(), r+1, r+1, row)
		tx.SetSelection(doc.Caret(doc.Point{Leaf: row.Child(c).ID()}))
		return true
	}
}

// AddRow adds a row above (at 0) or below (at 1) the caret's.
func AddRow(reg *doc.Registry, at int) doc.Command {
	return func(s doc.State, tx *doc.Tx) bool {
		_, _, table, r, c, ok := cellAt(s)
		if !ok || (at == 0 && r == 0) {
			return false // nothing goes above the header
		}
		row := emptyRow(reg, table.Child(0).NumChildren())
		tx.ReplaceChildren(table.ID(), r+at, r+at, row)
		tx.SetSelection(doc.Caret(doc.Point{Leaf: row.Child(c).ID()}))
		return true
	}
}

// AddColumn adds a column left (at 0) or right (at 1) of the caret's.
func AddColumn(reg *doc.Registry, at int) doc.Command {
	return func(s doc.State, tx *doc.Tx) bool {
		_, _, table, _, c, ok := cellAt(s)
		if !ok {
			return false
		}
		for i := 0; i < table.NumChildren(); i++ {
			r := table.Child(i)
			tx.ReplaceChildren(r.ID(), c+at, c+at, doc.NewTextBlock(reg.Node("table_cell"), doc.A{"header": i == 0}, doc.Inline{}))
		}
		aligns := Aligns(table)
		aligns = append(aligns[:min(c+at, len(aligns))], append([]string{""}, aligns[min(c+at, len(aligns)):]...)...)
		tx.SetAttr(table.ID(), "align", strings.Join(aligns, ","))
		return true
	}
}

// DeleteRow deletes the caret's row, or the table with its last body row.
func DeleteRow(s doc.State, tx *doc.Tx) bool {
	_, _, table, r, _, ok := cellAt(s)
	if !ok || r == 0 {
		return false
	}
	tx.ReplaceChildren(table.ID(), r, r+1)
	return true
}

// DeleteColumn deletes the caret's column; the last one deletes the table.
func DeleteColumn(s doc.State, tx *doc.Tx) bool {
	_, _, table, _, c, ok := cellAt(s)
	if !ok {
		return false
	}
	if table.Child(0).NumChildren() == 1 {
		p, _ := s.Parent(table.ID())
		i := s.Index(table.ID())
		tx.ReplaceChildren(p, i, i+1)
		return true
	}
	for i := 0; i < table.NumChildren(); i++ {
		r := table.Child(i)
		tx.ReplaceChildren(r.ID(), c, c+1)
	}
	aligns := Aligns(table)
	if c < len(aligns) {
		aligns = append(aligns[:c], aligns[c+1:]...)
		tx.SetAttr(table.ID(), "align", strings.Join(aligns, ","))
	}
	return true
}
