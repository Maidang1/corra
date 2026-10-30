package mgdoc

import (
	"strings"
	"testing"
)

const corpus = "# Title\n\nHello **bold** and ==marked== with x^2^ and H~2~O, $a^2+b^2$ and note[^1].\n\n- one\n- two\n\n> quote\n\n| a | b |\n| - | - |\n| 1 | 2 |\n\n$$\nx = 1\n$$\n\n> [!NOTE]\n> careful\n\n[^1]: The note.\n\n---\n\n<div>raw</div>\n\n```go\nfmt.Println(\"$x$ ==y==\")\n```\n\nText with `$code$` and https://x.com/~u/a^b.\n"

func roundTrip(t *testing.T, src string) string {
	t.Helper()
	c := &Codec{}
	return c.Save(c.Load(src))
}

func TestRoundTripPreservesUnsupportedSyntax(t *testing.T) {
	got := roundTrip(t, corpus)
	for _, want := range []string{
		"==marked==", "x^2^", "H~2~O", "$a^2+b^2$", "[^1]", "| a | b |", "| - | - |",
		"$$\nx = 1\n$$", "> [!NOTE]\n> careful", "[^1]: The note.", "---", "<div>raw</div>",
		"```go\nfmt.Println(\"$x$ ==y==\")", "https://x.com/~u/a^b", "`$code$`", "https://x.com/~u/a^b",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lost %q in:\n%s", want, got)
		}
	}
	if again := roundTrip(t, got); again != got {
		t.Errorf("not stable:\n%s\n---\n%s", got, again)
	}
}

func TestTagsAreRuns(t *testing.T) {
	c := &Codec{}
	text := c.Load("a ==b== c\n")
	found := false
	for _, p := range text.Paragraphs() {
		for _, r := range p.Runs {
			if r.Attributes.Tag == TagMark && r.Text == "b" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("highlight run missing")
	}
}
