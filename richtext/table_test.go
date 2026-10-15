package richtext

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestEditableTableRoundTrip(t *testing.T) {
	for _, source := range []string{
		"| A | B |\n| :--- | ---: |\n| **one** | two |\n",
		"A | B\r\n--- | ---\r\n | \r\n",
		"| A | B |\n|---|---|",
		"| A | B |\n|---|---|\n| a\\|b | `x\\|y` |",
	} {
		doc := Parse(source)
		if len(doc.Paragraphs) != 1 || doc.Paragraphs[0].Kind != "table" {
			t.Errorf("table not editable: %+v", doc)
			continue
		}
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		if got := Serialize(doc); got != source {
			t.Errorf("round trip: %q != %q", got, source)
		}
	}
}

func TestTableCodeSpanRoundTripPreservesLiteralPipeEscapes(t *testing.T) {
	for _, content := range []string{`a|b`, `a\|b`, `a\\|b`, `a\\\|b`, "a`|b", "`\\|`", "``\\\\|`"} {
		t.Run(fmt.Sprintf("%q", content), func(t *testing.T) {
			doc := Parse("| Code | Other |\n| --- | --- |\n| old | intact |")
			doc.Paragraphs[0].Rows[1][0].Runs = []Run{{Text: content, Code: true}}
			encoded := Serialize(doc)
			again := Parse(encoded)
			if len(again.Paragraphs) != 1 || again.Paragraphs[0].Kind != "table" || len(again.Paragraphs[0].Rows) != 2 || len(again.Paragraphs[0].Rows[1]) != 2 {
				t.Fatalf("code broke table structure: %q -> %+v", encoded, again)
			}
			row := again.Paragraphs[0].Rows[1]
			if len(row[0].Runs) != 1 || row[0].Runs[0].Text != content || !row[0].Runs[0].Code {
				t.Errorf("code content changed: %q -> %+v; want %q", encoded, row[0].Runs, content)
			}
			if PlainText(Paragraph{Runs: row[1].Runs}) != "intact" {
				t.Errorf("adjacent cell changed: %q -> %+v", encoded, row[1])
			}
		})
	}
}

func TestTableAdjacentCellEditPreservesCodePipeEscapes(t *testing.T) {
	for slashes := 0; slashes <= 3; slashes++ {
		t.Run(fmt.Sprint(slashes), func(t *testing.T) {
			content := "a" + strings.Repeat(`\`, slashes) + "|b"
			codeSource := "`a" + strings.Repeat(`\`, slashes+1) + "|b`"
			doc := Parse("| Code | Other |\n| --- | --- |\n| " + codeSource + " | before |")
			row := doc.Paragraphs[0].Rows[1]
			if len(row[0].Runs) != 1 || row[0].Runs[0].Text != content || !row[0].Runs[0].Code {
				t.Fatalf("initial GFM code parsing changed: %+v; want %q", row[0], content)
			}
			row[1].Runs = []Run{{Text: "after"}}
			encoded := Serialize(doc)
			if !strings.Contains(encoded, "| "+codeSource+" | after |") {
				t.Errorf("adjacent edit changed code source: %q; want %q", encoded, codeSource)
			}
			again := Parse(encoded)
			row = again.Paragraphs[0].Rows[1]
			if len(row) != 2 || len(row[0].Runs) != 1 || row[0].Runs[0].Text != content || !row[0].Runs[0].Code || PlainText(Paragraph{Runs: row[1].Runs}) != "after" {
				t.Errorf("adjacent edit changed table content: %q -> %+v", encoded, row)
			}
		})
	}
}

func TestTableCellEditPreservesStructure(t *testing.T) {
	doc := Parse("before\n\n| A | B |\n| :--- | ---: |\n| **one** | two |\n\nafter")
	data, _ := json.Marshal(doc)
	var model map[string]any
	json.Unmarshal(data, &model)
	paragraph := model["paragraphs"].([]any)[1].(map[string]any)
	rows, ok := paragraph["rows"].([]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("missing editable rows: %s", data)
	}
	rows[1].([]any)[1].(map[string]any)["runs"] = []any{map[string]any{"text": "new | value", "bold": true}}
	data, _ = json.Marshal(model)
	json.Unmarshal(data, &doc)
	got := Serialize(doc)
	if !strings.HasPrefix(got, "before\n\n") || !strings.HasSuffix(got, "\n\nafter") {
		t.Fatalf("adjacent text changed: %q", got)
	}
	reloaded := Parse(got)
	data, _ = json.Marshal(reloaded)
	if !strings.Contains(string(data), `"text":"new | value","bold":true`) || !strings.Contains(got, ":---") || !strings.Contains(got, "---:") {
		t.Errorf("cell or alignment lost: %q %s", got, data)
	}
}
