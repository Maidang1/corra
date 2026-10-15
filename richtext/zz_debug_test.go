package richtext

import (
	"fmt"
	"testing"
)

func TestDebugImageKind(t *testing.T) {
	doc := Parse("![截图](shot.png)\n\n- 列表")
	for i, p := range doc.Paragraphs {
		fmt.Printf("%d kind=%q raw=%q\n", i, p.Kind, p.Raw)
	}
}
