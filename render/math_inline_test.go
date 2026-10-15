package render

import "testing"

func TestInlineMathEscapedDollarAndCurrency(t *testing.T) {
	for _, sample := range []struct{ source, tex string }{
		{`before $x+\$5$ after`, `x+\$5`},
		{`price $5 and $6 remain text`, ``},
		{`escaped \$x$ remains text`, ``},
		{`$ x$ is not math`, ``},
	} {
		var expressions []string
		for _, in := range ParseInlines(sample.source) {
			if in.Kind == InlineMath {
				expressions = append(expressions, in.Text)
			}
		}
		if sample.tex == "" {
			if len(expressions) != 0 {
				t.Errorf("literal became math: %q -> %q", sample.source, expressions)
			}
			continue
		}
		if len(expressions) != 1 || expressions[0] != sample.tex {
			t.Errorf("TeX changed: %q -> %q", sample.source, expressions)
		}
	}
}
