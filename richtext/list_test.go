package richtext

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

func paragraphDepth(t *testing.T, p Paragraph) int {
	t.Helper()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		Depth int `json:"depth"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatal(err)
	}
	return metadata.Depth
}

func TestNestedListsExposeEditableParagraphs(t *testing.T) {
	source := "3. 中文😀\r\n   + **child**\r\n     - [X] done\r\n   + sibling\r\n4. last\r\n"
	doc := Parse(source)
	wantKinds := []string{"ordered", "bullet", "checklist", "bullet", "ordered"}
	wantDepths := []int{0, 1, 2, 1, 0}
	if len(doc.Paragraphs) != len(wantKinds) {
		t.Fatalf("got %d paragraphs, want %d: %+v", len(doc.Paragraphs), len(wantKinds), doc)
	}
	for i, p := range doc.Paragraphs {
		if p.Kind != wantKinds[i] || paragraphDepth(t, p) != wantDepths[i] {
			t.Errorf("paragraph %d: kind=%s depth=%d, want %s depth=%d", i, p.Kind, paragraphDepth(t, p), wantKinds[i], wantDepths[i])
		}
	}
	if doc.Paragraphs[0].Ordinal != 3 || doc.Paragraphs[4].Ordinal != 4 || !doc.Paragraphs[2].Checked {
		t.Errorf("missing ordinal/check metadata: %+v", doc)
	}
	if PlainText(doc.Paragraphs[0]) != "中文😀" || !doc.Paragraphs[1].Runs[0].Bold {
		t.Errorf("lost rich inline content: %+v", doc)
	}
	encoded, _ := json.Marshal(doc)
	var decoded Document
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if got := Serialize(decoded); got != source {
		t.Errorf("untouched JSON round trip: got %q, want %q", got, source)
	}
}

func TestEditingNestedListLeafPreservesIndentAndSiblings(t *testing.T) {
	for _, source := range []string{
		"3. parent\r\n   + child\r\n   + sibling\r\n4. end\r\n",
		"- parent\n\t- child\n\t- sibling\n- end\n",
		"123. parent\n\n     8) child\n     9) sibling\n124. end\n",
		"- [ ] parent\n\n  3. child\n     * grandchild\n  4. sibling\n- end\n",
	} {
		t.Run(source, func(t *testing.T) {
			doc := Parse(source)
			found := false
			for i, p := range doc.Paragraphs {
				if p.Kind != "source" && PlainText(p) == "child" {
					doc.Paragraphs[i].Runs = []Run{{Text: "edited 中文😀"}}
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("child did not become an editable rich item: %+v", doc)
			}
			got := Serialize(doc)
			// Keep the container's delimiter; changing it could merge this
			// list with a neighboring list at the same depth.
			want := strings.Replace(source, "child", "edited 中文😀", 1)
			if got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
			again := Parse(got)
			if len(again.Paragraphs) != len(doc.Paragraphs) {
				t.Fatalf("edit changed list shape: %+v", again)
			}
			for i, p := range again.Paragraphs {
				if p.Kind != doc.Paragraphs[i].Kind || paragraphDepth(t, p) != paragraphDepth(t, doc.Paragraphs[i]) {
					t.Errorf("item %d changed nesting after edit: %+v", i, again)
				}
			}
		})
	}
}

func TestNestedListDepthEditPersists(t *testing.T) {
	doc := Parse("- parent\n  - child\n  - sibling\n- end\n")
	data, _ := json.Marshal(doc)
	var wire map[string]any
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	wire["paragraphs"].([]any)[2].(map[string]any)["depth"] = float64(2)
	data, _ = json.Marshal(wire)
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	got := Serialize(doc)
	want := "- parent\n  - child\n    - sibling\n- end\n"
	if got != want {
		t.Fatalf("depth edit reused stale raw: got %q, want %q", got, want)
	}
	again := Parse(got)
	if paragraphDepth(t, again.Paragraphs[2]) != 2 {
		t.Fatalf("depth edit did not reparse: %+v", again)
	}
}

func TestNewMixedNestedListsSerializeAsOneTree(t *testing.T) {
	var doc Document
	if err := json.Unmarshal([]byte(`{"paragraphs":[{"kind":"ordered","ordinal":3,"runs":[{"text":"parent"}]},{"kind":"bullet","depth":1,"runs":[{"text":"child"}]},{"kind":"checklist","depth":2,"checked":true,"runs":[{"text":"done"}]},{"kind":"bullet","depth":1,"runs":[{"text":"sibling"}]},{"kind":"ordered","ordinal":4,"runs":[{"text":"end"}]}]}`), &doc); err != nil {
		t.Fatal(err)
	}
	want := "3. parent\n   - child\n     - [x] done\n   - sibling\n4. end"
	if got := Serialize(doc); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestUnsupportedNestedListKeepsEntireSource(t *testing.T) {
	for _, source := range []string{
		"- parent\n  - child\n\n        indented code\n- sibling",
		"- parent\n  > quote\n- sibling",
		"- parent\n  - ![image](image.png)\n- sibling",
		"- parent\n  - \n  - next\n- end",
	} {
		t.Run(source, func(t *testing.T) {
			doc := Parse(source)
			for _, p := range doc.Paragraphs {
				if p.Kind != "source" {
					t.Fatalf("unsupported list was partially flattened: %+v", doc)
				}
			}
			if got := Serialize(doc); got != source {
				t.Fatalf("lost source: got %q, want %q", got, source)
			}
		})
	}
}

func TestNestedListSourceRoundTrips(t *testing.T) {
	for _, source := range []string{
		"  + parent\r\n    - [ ] child\r\n\r\n      7) task\r\n    - next\r\n  + end",
		"- parent\n\n  - child\n\n  - sibling\n\n- end\n\n",
		"009. parent\n     - child\n010. end\n",
		"3. [X] parent\n   - child\n4. [ ] end\n",
		"- parent\n\n  - \n  - next\n- end\n",
		"- parent\n  - [ ] \n  - [x] \n- end\n",
	} {
		t.Run(source, func(t *testing.T) {
			doc := Parse(source)
			for _, p := range doc.Paragraphs {
				if !isList(p) && !emptyParagraph(p) {
					t.Fatalf("list did not become editable: %+v", doc)
				}
			}
			if got := Serialize(doc); got != source {
				t.Fatalf("got %q, want %q", got, source)
			}
		})
	}
}

func TestListStructuralEditsReindentWholeTree(t *testing.T) {
	t.Run("ordered marker grows", func(t *testing.T) {
		doc := Parse("9. parent\n   - child\n10. end\n")
		doc.Paragraphs[0].Ordinal = 10
		want := "10. parent\n    - child\n10. end\n"
		if got := Serialize(doc); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
	t.Run("minimal marker shrinks", func(t *testing.T) {
		doc := Parse("000009. parent\n        - child\n000010. end\n")
		doc.Paragraphs[0].Runs = []Run{{Text: "changed"}}
		got := Serialize(doc)
		again := Parse(got)
		if len(again.Paragraphs) != 3 || again.Paragraphs[1].Kind != "bullet" || again.Paragraphs[1].Depth != 1 {
			t.Fatalf("marker shrinking lost child: %q -> %+v", got, again)
		}
	})
	t.Run("subtree indent and outdent", func(t *testing.T) {
		source := "- first\n- parent\n  - child\n- end\n"
		doc := Parse(source)
		doc.Paragraphs[1].Depth++
		doc.Paragraphs[2].Depth++
		want := "- first\n  - parent\n    - child\n- end\n"
		if got := Serialize(doc); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
		doc.Paragraphs[1].Depth--
		doc.Paragraphs[2].Depth--
		if got := Serialize(doc); got != source {
			t.Fatalf("outdent did not restore exact source: %q", got)
		}
	})
	t.Run("untouched wide parent", func(t *testing.T) {
		for _, source := range []string{"000009. parent\n        - child\n        - sibling\n", "-   parent\n    - child\n    - sibling\n"} {
			doc := Parse(source)
			doc.Paragraphs[2].Depth++
			got := Serialize(doc)
			again := Parse(got)
			if len(again.Paragraphs) != 3 || again.Paragraphs[1].Depth != 1 || again.Paragraphs[2].Depth != 2 {
				t.Fatalf("raw parent marker width lost nesting: %q -> %+v", got, again)
			}
		}
	})
}

func TestNewOrderedNestedEmptyItem(t *testing.T) {
	doc := Parse("3. parent\n\n   7. child\n4. end\n")
	item := Paragraph{Kind: "ordered", Ordinal: 8, Depth: 1, Runs: []Run{}}
	doc.Paragraphs = append(doc.Paragraphs[:2], append([]Paragraph{item}, doc.Paragraphs[2:]...)...)
	want := "3. parent\n\n   7. child\n   8. \n4. end\n"
	if got := Serialize(doc); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNewNestedListBoundaryUsesStandardMarkdown(t *testing.T) {
	for _, child := range []Paragraph{
		{Kind: "ordered", Ordinal: 7, Depth: 1, Runs: []Run{{Text: "child"}}},
		{Kind: "checklist", Ordinal: 7, Depth: 1, Runs: []Run{{Text: "child"}}},
		{Kind: "bullet", Depth: 1, Runs: []Run{}},
	} {
		doc := Document{Paragraphs: []Paragraph{{Kind: "ordered", Ordinal: 3, Runs: []Run{{Text: "parent"}}}, child}}
		got := Serialize(doc)
		if !strings.Contains(got, "parent\n\n") {
			t.Fatalf("missing required blank line: %q", got)
		}
		again := Parse(got)
		if len(again.Paragraphs) != 2 || again.Paragraphs[1].Depth != 1 || again.Paragraphs[1].Kind != child.Kind {
			t.Fatalf("new child did not reparse: %q -> %+v", got, again)
		}
		standard := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader([]byte(got)))
		var depths []int
		if err := ast.Walk(standard, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering && node.Kind() == ast.KindListItem {
				depth := 0
				for parent := node.Parent(); parent != nil; parent = parent.Parent() {
					if parent.Kind() == ast.KindListItem {
						depth++
					}
				}
				depths = append(depths, depth)
			}
			return ast.WalkContinue, nil
		}); err != nil {
			t.Fatal(err)
		}
		if len(depths) != 2 || depths[0] != 0 || depths[1] != 1 {
			t.Fatalf("standard renderer parser lost nested structure: %q -> %v", got, depths)
		}
	}
}

func TestEditingOrderedTaskKeepsOrderedMarker(t *testing.T) {
	doc := Parse("3. [ ] parent\n   - child\n4. [X] end\n")
	if doc.Paragraphs[0].Kind != "checklist" || doc.Paragraphs[0].Ordinal != 3 {
		t.Fatalf("ordered task metadata lost: %+v", doc)
	}
	doc.Paragraphs[0].Checked = true
	want := "3. [x] parent\n   - child\n4. [X] end\n"
	if got := Serialize(doc); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	doc = Parse("3. [ ] task\r\n4. next\r\n")
	doc.Paragraphs[0].Checked = true
	if got, want := Serialize(doc), "3. [x] task\r\n4. next\r\n"; got != want {
		t.Fatalf("ordered task edit changed sibling boundary: got %q, want %q", got, want)
	}
}

func TestNestedListContainersRemainDistinct(t *testing.T) {
	source := "- parent\n\n  3. a\n  4. b\n\n  7) c\n  8) d\n- tail"
	doc := Parse(source)
	if len(doc.Paragraphs) != 6 {
		t.Fatalf("wrong paragraph shape: %+v", doc)
	}
	data, _ := json.Marshal(doc)
	var wire struct {
		Paragraphs []struct {
			Root      string `json:"listId"`
			Container string `json:"listContainer"`
		}
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	for _, p := range wire.Paragraphs {
		if p.Root == "" || p.Root != wire.Paragraphs[0].Root || p.Container == "" {
			t.Fatalf("missing list identity: %+v", wire)
		}
	}
	if wire.Paragraphs[1].Container != wire.Paragraphs[2].Container || wire.Paragraphs[3].Container != wire.Paragraphs[4].Container || wire.Paragraphs[1].Container == wire.Paragraphs[3].Container {
		t.Fatalf("separate nested containers were merged: %+v", wire)
	}
	if got := Serialize(doc); got != source {
		t.Fatalf("untouched containers changed: got %q, want %q", got, source)
	}
	doc.Paragraphs[3].Runs = []Run{{Text: "changed"}}
	got := Serialize(doc)
	want := strings.Replace(source, "7) c", "7) changed", 1)
	if got != want {
		t.Fatalf("container delimiter lost after edit: got %q, want %q", got, want)
	}
	again := Parse(got)
	if len(again.Paragraphs) != 6 || again.Paragraphs[3].Ordinal != 7 || again.Paragraphs[4].Ordinal != 8 {
		t.Fatalf("container numbering changed after edit: %+v", again)
	}
}
