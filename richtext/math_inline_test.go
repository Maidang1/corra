package richtext

import "testing"

func TestInlineMathKeepsTeXEscapesAndMarkdownTokens(t *testing.T) {
	for _, expression := range []string{`\{x\}`, `x\_1`, `a * b * c`, `\text{**literal**}`, `\text{&amp;}`, `x+\$5`} {
		source := "before $" + expression + "$ after"
		doc := Parse(source)
		var math []Run
		for _, p := range doc.Paragraphs {
			for _, run := range p.Runs {
				if run.Math != "" {
					math = append(math, run)
				}
			}
		}
		if len(math) != 1 || math[0].Math != expression || math[0].Text != "$"+expression+"$" {
			t.Errorf("TeX changed before rendering: %q -> %+v", expression, math)
		}
		if got := Serialize(doc); got != source {
			t.Errorf("source changed: %q", got)
		}
	}
}

func TestEditedTableMathPreservesPipes(t *testing.T) {
	for _, expression := range []string{`|x|`, `\|x\|`, `\text{a|b}`} {
		doc := Parse("| Math | Other |\n| --- | --- |\n| $x$ | ok |")
		doc.Paragraphs[0].Rows[1][0].Runs = []Run{{Text: "$" + expression + "$", Math: expression}}
		got := Serialize(doc)
		parsed := Parse(got)
		if len(parsed.Paragraphs) != 1 || parsed.Paragraphs[0].Kind != "table" || len(parsed.Paragraphs[0].Rows[1]) != 2 {
			t.Errorf("equation broke table: %q", got)
			continue
		}
		row := parsed.Paragraphs[0].Rows[1]
		if len(row[0].Runs) != 1 || row[0].Runs[0].Math != expression || row[1].Runs[0].Text != "ok" {
			t.Errorf("equation or adjacent cell changed: %q %+v", got, row)
		}
	}
}
