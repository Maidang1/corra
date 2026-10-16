package render

import (
	"strings"
	"testing"
)

// flowchart.js / sequence.js fences translate to mermaid; these tests pin the
// translation, which is where the compatibility lives.

func TestFlowchartToMermaid(t *testing.T) {
	src := "st=>start: Start\n" +
		"e=>end: End\n" +
		"op=>operation: My Operation\n" +
		"cond=>condition: Yes or No?\n" +
		"st->op->cond\n" +
		"cond(yes)->e\n" +
		"cond(no)->op\n"
	out, err := flowchartToMermaid(src)
	if err != nil {
		t.Fatalf("flowchartToMermaid: %v", err)
	}
	// Node labels stay quoted: that is the only form surviving ":" and "#".
	for _, want := range []string{
		"flowchart TD",
		`nst(["Start"])`,
		`ne(["End"])`,
		`nop["My Operation"]`,
		`ncond{"Yes or No?"}`,
		"nst --> nop",
		"nop --> ncond",
		"ncond -->|yes| ne",
		"ncond -->|no| nop",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// A URL in the label is a click target, not text to render.
	if strings.Contains(out, "click") {
		t.Errorf("click statement leaked into output:\n%s", out)
	}
}

func TestFlowchartToMermaidEdgeLabel(t *testing.T) {
	out, err := flowchartToMermaid("a=>start: A\nb=>end: B\na(yes)->b\n")
	if err != nil {
		t.Fatalf("flowchartToMermaid: %v", err)
	}
	// Unquoted: |"yes"| renders the quotes as part of the label.
	if !strings.Contains(out, "na -->|yes| nb") {
		t.Errorf("edge label wrong:\n%s", out)
	}
}

func TestFlowchartToMermaidRejects(t *testing.T) {
	for _, src := range []string{"", "not a diagram at all", "   \n  "} {
		if out, err := flowchartToMermaid(src); err == nil {
			t.Errorf("flowchartToMermaid(%q) = %q, want error", src, out)
		}
	}
}

func TestSequenceToMermaid(t *testing.T) {
	src := "Alice->Bob: Hello Bob, how are you?\n" +
		"Note right of Bob: Bob thinks\n" +
		"Bob-->Alice: I am good thanks!\n"
	out, err := sequenceToMermaid(src)
	if err != nil {
		t.Fatalf("sequenceToMermaid: %v", err)
	}
	for _, want := range []string{
		"sequenceDiagram",
		"participant Alice",
		"participant Bob",
		"Alice ->> Bob: Hello Bob, how are you?",
		"Note right of Bob: Bob thinks",
		"Bob -->> Alice: I am good thanks!",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestSequenceToMermaidParticipantAlias(t *testing.T) {
	out, err := sequenceToMermaid("participant A as Alice\nA->B: hi\n")
	if err != nil {
		t.Fatalf("sequenceToMermaid: %v", err)
	}
	if !strings.Contains(out, "participant A as Alice") {
		t.Errorf("participant alias lost:\n%s", out)
	}
	if !strings.Contains(out, "A ->> B: hi") {
		t.Errorf("message lost:\n%s", out)
	}
}

func TestSequenceToMermaidRejects(t *testing.T) {
	for _, src := range []string{"", "   ", "just words"} {
		if out, err := sequenceToMermaid(src); err == nil {
			t.Errorf("sequenceToMermaid(%q) = %q, want error", src, out)
		}
	}
}

// The fence spelling decides the diagram kind, and the kind decides the
// translation, so a typo in the fence name silently changes engines.
func TestDiagramKindForFence(t *testing.T) {
	cases := map[string]DiagramKind{
		"flow": DiagramFlowchartJS, "sequence": DiagramSequenceJS,
		"mermaid": DiagramMermaid, "d2": DiagramD2, "goat": DiagramGoAT,
	}
	for fence, want := range cases {
		doc := Parse("```" + fence + "\nst=>start: Start\n```\n")
		if len(doc.Blocks) != 1 {
			t.Fatalf("fence %q: got %d blocks", fence, len(doc.Blocks))
		}
		b := doc.Blocks[0]
		if b.Kind != KindDiagram || b.Diagram != want {
			t.Errorf("fence %q: kind=%v diagram=%v, want KindDiagram/%v", fence, b.Kind, b.Diagram, want)
		}
		if strings.TrimRight(b.DiagramS, "\n") != "st=>start: Start" {
			t.Errorf("fence %q: source %q", fence, b.DiagramS)
		}
	}
}

// Both new engines reuse the mermaid renderer, so a translation failure must
// surface as an error rather than a blank diagram.
func TestRenderDiagramFlowAndSequence(t *testing.T) {
	ctx := t.Context()
	for _, tc := range []struct {
		kind DiagramKind
		src  string
	}{
		{DiagramFlowchartJS, "st=>start: Start\ne=>end: End\nst->e\n"},
		{DiagramSequenceJS, "Alice->Bob: Hello Bob, how are you?\nNote right of Bob: Bob thinks\nBob-->Alice: I am good thanks!\n"},
	} {
		svg, err := RenderDiagram(ctx, tc.kind, tc.src)
		if err != nil {
			t.Errorf("RenderDiagram(%v): %v", tc.kind, err)
			continue
		}
		if !strings.Contains(string(svg), "<svg") {
			t.Errorf("RenderDiagram(%v): no svg in output", tc.kind)
		}
	}
}
