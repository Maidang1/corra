package richtext

import (
	"encoding/json"
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
