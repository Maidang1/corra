package richtext

import (
	"encoding/json"
	"strings"
	"testing"
)

func runLinkTitle(t *testing.T, run Run) string {
	t.Helper()
	data, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Title string `json:"linkTitle"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	return wire.Title
}

func TestInlineLinkTitlesAreEditable(t *testing.T) {
	for _, source := range []string{
		`[中文😀](/url "a title")`,
		`[中文😀](/url 'a title')`,
		`[中文😀](/url (a title))`,
	} {
		t.Run(source, func(t *testing.T) {
			doc := Parse(source)
			if len(doc.Paragraphs) != 1 || doc.Paragraphs[0].Kind != "body" {
				t.Fatalf("title link stayed source: %+v", doc)
			}
			run := doc.Paragraphs[0].Runs[0]
			if run.Text != "中文😀" || run.Link != "/url" || runLinkTitle(t, run) != "a title" {
				t.Fatalf("lost link metadata: %+v", run)
			}
			if got := Serialize(doc); got != source {
				t.Fatalf("untouched spelling changed: got %q, want %q", got, source)
			}
		})
	}
}

func TestReferenceLinksResolveAndPreserveDefinitions(t *testing.T) {
	definition := "[ref]: </路径?q=1&amp;x=2> 'a &quot;title&quot;'  \r\n"
	for _, spelling := range []string{"[中文😀][ref]", "[ref][]", "[ref]"} {
		for _, before := range []bool{false, true} {
			source := spelling + " unchanged\r\n\r\n" + definition
			if before {
				source = definition + "\r\n" + spelling + " unchanged\r\n"
			}
			t.Run(source, func(t *testing.T) {
				doc := Parse(source)
				found := false
				for i, p := range doc.Paragraphs {
					if p.Kind != "body" {
						continue
					}
					if p.Runs[0].Link != "/路径?q=1&x=2" || runLinkTitle(t, p.Runs[0]) != `a "title"` {
						t.Fatalf("reference did not resolve: %+v", p)
					}
					doc.Paragraphs[i].Runs[len(p.Runs)-1].Text = " changed"
					found = true
				}
				if !found {
					t.Fatalf("reference remained source: %+v", doc)
				}
				got := Serialize(doc)
				if !strings.Contains(got, definition) || !strings.Contains(got, `](</路径?q=1\&x=2> "a \"title\"") changed`) {
					t.Fatalf("edited reference lost destination/title/definition: %q", got)
				}
				if untouched := Serialize(Parse(source)); untouched != source {
					t.Fatalf("untouched reference spelling changed: got %q, want %q", untouched, source)
				}
			})
		}
	}
}

func TestEditedLinkTitlesRetainEscapedCharacters(t *testing.T) {
	source := "before [中文😀](</url> \"say \\\"yes\\\" &amp; \\\\ path\") after"
	doc := Parse(source)
	if doc.Paragraphs[0].Kind != "body" {
		t.Fatalf("escaped title stayed source: %+v", doc)
	}
	doc.Paragraphs[0].Runs[0].Text = "changed "
	got := Serialize(doc)
	again := Parse(got)
	if len(again.Paragraphs) != 1 || again.Paragraphs[0].Kind != "body" || !sameRuns(doc.Paragraphs[0].Runs, again.Paragraphs[0].Runs) {
		t.Fatalf("link semantics changed: %q -> %+v", got, again)
	}
	if title := runLinkTitle(t, again.Paragraphs[0].Runs[1]); title != "say \"yes\" & \\ path" {
		t.Fatalf("title changed: %q", title)
	}
}

func TestLinkTitleOnlyChangePersists(t *testing.T) {
	doc := Parse(`[label](/url "old")`)
	data, _ := json.Marshal(doc)
	var wire map[string]any
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	wire["paragraphs"].([]any)[0].(map[string]any)["runs"].([]any)[0].(map[string]any)["linkTitle"] = "new"
	data, _ = json.Marshal(wire)
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if got, want := Serialize(doc), `[label](</url> "new")`; got != want {
		t.Fatalf("title change reused stale raw: got %q, want %q", got, want)
	}
}
