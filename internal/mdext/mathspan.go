package mdext

import (
	"bytes"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// MathSpan recognizes a raw inline TeX span before Markdown can consume its
// escapes or emphasis markers. Single-dollar delimiters follow Pandoc's rules.
func MathSpan(line []byte) (tex []byte, length int) {
	if len(line) < 3 || line[0] != '$' {
		return nil, 0
	}
	delimiters := 1
	if line[1] == '$' {
		delimiters = 2
	} else if mathSpace(line[1]) {
		return nil, 0
	}
	for i := delimiters; i < len(line); i++ {
		if line[i] == '\n' || line[i] == '\r' {
			break
		}
		if line[i] != '$' || (delimiters == 2 && (i+1 >= len(line) || line[i+1] != '$')) {
			continue
		}
		slashes := 0
		for j := i - 1; j >= delimiters && line[j] == '\\'; j-- {
			slashes++
		}
		if slashes%2 != 0 {
			continue
		}
		if delimiters == 1 && (mathSpace(line[i-1]) || (i+1 < len(line) && line[i+1] >= '0' && line[i+1] <= '9')) {
			continue
		}
		body := line[delimiters:i]
		if len(bytes.TrimSpace(body)) == 0 {
			return nil, 0
		}
		return body, i + delimiters
	}
	return nil, 0
}

func mathSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }

// GFM consumes one slash before each cell pipe, including inside code/math.
// Custom math nodes bypass Goldmark's Text/CodeSpan unescaper, so do it here.
func MathText(parent ast.Node, tex []byte) string {
	for node := parent; node != nil; node = node.Parent() {
		if node.Kind() == east.KindTableCell {
			return string(bytes.ReplaceAll(tex, []byte(`\|`), []byte(`|`)))
		}
	}
	return string(tex)
}
