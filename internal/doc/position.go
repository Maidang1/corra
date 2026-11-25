package doc

import "fmt"

// Point is a place in a text block: its ID and a UTF-16 offset into its
// content. Edits in other blocks never move it.
type Point struct {
	Leaf   NodeID
	Offset int
}

func (p Point) String() string { return fmt.Sprintf("%d:%d", p.Leaf, p.Offset) }

// Selection is a TextSelection, NodeSelection or CellSelection.
type Selection interface{ isSelection() }

// TextSelection runs from Anchor, where it started, to Head, where the
// caret is. They are equal for a caret.
type TextSelection struct{ Anchor, Head Point }

// Caret returns a collapsed selection at p.
func Caret(p Point) TextSelection { return TextSelection{p, p} }

func (s TextSelection) Empty() bool { return s.Anchor == s.Head }

// NodeSelection selects whole blocks, as an image, a formula or several
// blocks at once.
type NodeSelection struct{ IDs []NodeID }

// CellRef addresses a table cell.
type CellRef struct{ Row, Col int }

// CellSelection selects a rectangle of a table's cells.
type CellSelection struct {
	Table        NodeID
	Anchor, Head CellRef
}

func (TextSelection) isSelection() {}
func (NodeSelection) isSelection() {}
func (CellSelection) isSelection() {}

// LeafRange is the part [From, To) of one text block a selection covers.
type LeafRange struct {
	Leaf     NodeID
	From, To int
}
