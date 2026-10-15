// Package render turns Markdown source into a block model: a UI-agnostic
// intermediate representation that views paint and exporters serialize.
// Keeping it free of package ui makes the whole pipeline unit-testable.
package render

// Kind identifies the kind of a top-level block.
type Kind int

const (
	KindHeading     Kind = iota // # .. ######
	KindParagraph               // plain text with inline formatting
	KindCodeBlock               // fenced or indented code
	KindQuote                   // >
	KindList                    // -, *, 1. (flat or nested)
	KindTable                   // GFM pipe table
	KindMathBlock               // $$ ... $$
	KindDiagram                 // mermaid / d2 / goat fenced block
	KindChart                   // ```chart YAML data
	KindImage                   // standalone image paragraph
	KindHR                      // ---
	KindHTML                    // raw HTML block, shown as source
	KindFrontMatter             // YAML front matter, shown as a meta card
	KindTOC                     // [TOC] table-of-contents marker
)

// ListItem is one item of a list block.
type ListItem struct {
	// Source is the item's own markdown source, without the list marker.
	Source string
	// Task is - [ ] / - [x]; Done is the checkbox state.
	Task  bool
	Done  bool
	Depth int
	// CheckboxOffset is the absolute source byte inside [ ]/[x], or -1.
	CheckboxOffset int
}

// TableCell is one cell of a table row.
type TableCell struct {
	Text  string
	Align int // 0 left, 1 center, 2 right
}

// DiagramKind selects the diagram engine.
type DiagramKind int

const (
	DiagramMermaid DiagramKind = iota
	DiagramD2
	DiagramGoAT
)

// Block is one top-level document block.
type Block struct {
	Kind   Kind
	Source string // exact markdown source of the block (for re-editing)
	// Start and End are byte offsets of Source within the parsed document,
	// used to splice edits back into the whole.
	Start, End int

	// KindHeading
	Level int

	// KindCodeBlock
	Language string

	// KindList
	Ordered bool
	Items   []ListItem

	// KindTable
	Header []TableCell
	Rows   [][]TableCell

	// KindMathBlock
	Math string

	// KindDiagram
	Diagram  DiagramKind
	DiagramS string // diagram source

	// KindChart
	ChartS string // chart YAML source

	// KindImage
	ImageURL    string
	ImageAlt    string
	ImageTitle  string
	ImageWidth  int // pixels, 0 = auto (from =WxH or <img width>)
	ImageHeight int
	ImageZoom   int // percent, 0 = none (from =NN%)
}

// HeadingText returns the plain text of a heading block (for the outline).
func (b *Block) HeadingText() string {
	if b.Kind != KindHeading {
		return ""
	}
	return PlainText(b.Source)
}
