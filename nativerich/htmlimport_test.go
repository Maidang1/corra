package nativerich

import (
	"strings"
	"testing"
)

func TestHTMLToMarkdown(t *testing.T) {
	src := `<h2>Title</h2><p>Hello <strong>bold</strong> and <a href="https://x.dev">link</a>.</p>` +
		`<ul><li>one</li><li>two<ul><li>nested</li></ul></li></ul>` +
		`<blockquote><p>quote</p></blockquote>` +
		`<pre><code class="language-go">fmt.Println(1)</code></pre>` +
		`<table><tr><th>A</th><th>B</th></tr><tr><td>1</td><td>2</td></tr></table>` +
		`<p><img src="p.png" width="200" alt="pic"></p>`
	md, err := HTMLToMarkdown(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Title", "**bold**", "[link](https://x.dev)",
		"- one", "  - nested", "> quote", "```go", "fmt.Println(1)",
		"| A | B |", "| --- | --- |", "| 1 | 2 |", "![pic](<p.png =200x>)"} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in:\n%s", want, md)
		}
	}
}
