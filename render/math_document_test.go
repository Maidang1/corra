package render

import (
	"strings"
	"testing"
)

func TestMathBlocksPreserveSourceRanges(t *testing.T) {
	source := "before\n\n$$\nx+y\n$$\n\n$$z$$\n\nend"
	doc := Parse(source)
	var blocks []Block
	for _, b := range doc.Blocks {
		if b.Kind == KindMathBlock {
			blocks = append(blocks, b)
			if b.End <= b.Start || b.Source != source[b.Start:b.End] || !strings.HasPrefix(b.Source, "$$") {
				t.Errorf("lost equation source range: %+v", b)
			}
		}
	}
	if len(blocks) != 2 || blocks[0].Source == blocks[1].Source {
		t.Errorf("different equations share source identity: %+v", blocks)
	}
	if len(doc.Blocks) != 4 || len(blocks) != 2 || blocks[0].Math != "x+y" || blocks[1].Math != "z" || strings.TrimSpace(doc.Blocks[len(doc.Blocks)-1].Source) != "end" {
		t.Errorf("single-line equation swallowed text or lost TeX: %+v", doc.Blocks)
	}
	opener := Parse("$$ x\ny\n$$")
	if len(opener.Blocks) != 1 || opener.Blocks[0].Math != "x\ny" {
		t.Errorf("opening-line TeX was dropped: %+v", opener.Blocks)
	}
	broken := Parse("before\n\n$$\n\\badcommand")
	if len(broken.Blocks) != 2 || !strings.Contains(broken.Blocks[1].Source, `\badcommand`) {
		t.Errorf("unclosed formula disappeared: %+v", broken)
	}
}

func TestDocumentMathCollectsInlineAndDisplayInOrder(t *testing.T) {
	doc := Parse("See $\\eqref{energy}$.\n\n`$code$`\n\n$$\nE=mc^2\\label{energy}\n$$\n\n| Ref |\n| --- |\n| $\\ref{energy}$ |")
	if len(doc.MathExpressions) != 3 || doc.MathExpressions[0].TeX != `\eqref{energy}` || !doc.MathExpressions[1].Display || doc.MathExpressions[2].TeX != `\ref{energy}` {
		t.Fatalf("lost document context: %+v", doc.MathExpressions)
	}
	index, ok := doc.MathIndex(2)
	if !ok || index != 1 {
		t.Errorf("display block points at wrong batch slot: %d %v", index, ok)
	}
}
