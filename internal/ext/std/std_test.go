package std_test

import (
	"testing"

	"github.com/Maidang1/corra/internal/doc/doctest"
	"github.com/Maidang1/corra/internal/ext/exttest"
	"github.com/Maidang1/corra/internal/ext/std"
)

func TestReadWrite(t *testing.T) {
	k := std.Kit()
	cases := []struct{ in, want string }{
		{"# Title\n\nHello **bold** and *it* ~~gone~~ `co*de` [link](http://x.y \"T\").\n", "# Title\n\nHello **bold** and *it* ~~gone~~ `co*de` [link](http://x.y \"T\").\n"},
		{"Setext\n======\n\n* a\n* b\n", "# Setext\n\n- a\n- b\n"},
		{"1) one\n2) two\n", "1. one\n2. two\n"},
		{"- [ ] todo\n- [x] done\n", "- [ ] todo\n- [x] done\n"},
		{"> quote\n>\n> - item\n", "> quote\n>\n> - item\n"},
		{"~~~js\nlet a = 1\n~~~\n", "```js\nlet a = 1\n```\n"},
		{"    indented\n", "```\nindented\n```\n"},
		{"a\nb  \nc\\\nd\n", "a\nb\\\nc\\\nd\n"},
		{"para\n\n---\n\n- a\n\n  b\n", "para\n\n---\n\n- a\n\n  b\n"},
		{"![alt *x*](img.png \"t\")\n", "![alt x](img.png \"t\")\n"},
		{"<div>\nhtml\n</div>\n\ntext <span>x</span>\n", "<div>\nhtml\n</div>\n\ntext <span>x</span>\n"},
		{"\\*not em\\* 1\\. & &amp;\n", "\\*not em\\* 1. & &\n"},
		{"- a\n\n\n* b\n", "- a\n\n* b\n"},
	}
	for _, c := range cases {
		d := k.Codec.Parse(c.in)
		if got := k.Codec.Write(d); got != c.want {
			t.Errorf("in   %q\ngot  %q\nwant %q\ndoc  %s", c.in, got, c.want, doctest.Dump(d))
		}
	}
}

func TestMarkdownStable(t *testing.T) {
	exttest.CheckMarkdown(t, std.Kit(),
		"# Title\n\nHello **bold *both*** and `x`.\n",
		"- a\n  - b\n    1. c\n- d\n",
		"> > nested\n>\n> back\n",
		"[*a* **b**](u) <http://auto.link> plain https://x.y\n",
		"1. x\n\n   ```go\n   code\n   ```\n\n2. y\n",
		"Line  \nbreak and soft\nbreak\n",
		"[ref]: http://x.y\n\nUse [ref].\n",
		"| not | a table |\n",
	)
}

func TestRandomDocumentsRoundTrip(t *testing.T) {
	k := std.Kit()
	n := doctest.Iterations(200)
	for seed := uint64(1); seed <= 5; seed++ {
		exttest.CheckRandom(t, k, seed, n)
	}
}

func TestRandomCommands(t *testing.T) {
	k := std.Kit()
	n := doctest.Iterations(30)
	for seed := uint64(1); seed <= 3; seed++ {
		exttest.CheckCommands(t, k, seed, n, 30)
	}
}

func TestExtendedSyntax(t *testing.T) {
	k := std.Kit()
	cases := []struct{ in, want string }{
		{"Note[^1] and[^n].\n\n[^1]: First.\n    Second line.\n\n[^n]: Other.\n", "Note[^1] and[^n].\n\n[^1]: First.\n    Second line.\n\n[^n]: Other.\n"},
		{"$a_b$ and $$x$$\n\n$$\n\\frac{1}{2}\n$$\n", "$a_b$ and $x$\n\n$$\n\\frac{1}{2}\n$$\n"},
		{"==hi== x^2^ H~2~O <u>u</u> <kbd>K</kbd> <span style=\"color:#d44c47\">red</span>\n", "==hi== x^2^ H~2~O <u>u</u> <kbd>K</kbd> <span style=\"color:#d44c47\">red</span>\n"},
		{"> [!WARNING]\n> careful\n", "> [!WARNING]\n> careful\n"},
		{"| a | b |\n|:-|-:|\n| `x\\|y` | **2** |\n| 3 |\n", "| a | b |\n| :--- | ---: |\n| `x\\|y` | **2** |\n| 3 |  |\n"},
		{"cost \\$5 and a ~ b\n", "cost \\$5 and a \\~ b\n"},
		{"![a](p.png =300x200)\n\n![b](q.png =50%)\n", "![a](p.png =300x200)\n\n![b](q.png =50%)\n"},
		{"---\ntitle: T\ntags: [a]\n---\n\n[TOC]\n\n# T\n", "---\ntitle: T\ntags: [a]\n---\n\n[TOC]\n\n# T\n"},
	}
	for _, c := range cases {
		d := k.Codec.Parse(c.in)
		if got := k.Codec.Write(d); got != c.want {
			t.Errorf("in   %q\ngot  %q\nwant %q\ndoc  %s", c.in, got, c.want, doctest.Dump(d))
		}
	}
}
