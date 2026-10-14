package render

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// HlSpan is one syntax-highlighted run of code. Class is a coarse token
// category that views map to theme colors, keeping this package ui-free.
type HlSpan struct {
	Text  string
	Class string
}

// Token classes, stable across chroma token types.
const (
	ClassText     = "text"
	ClassKeyword  = "keyword"
	ClassString   = "string"
	ClassNumber   = "number"
	ClassComment  = "comment"
	ClassOperator = "operator"
	ClassType     = "type"
	ClassLiteral  = "literal"
	ClassName     = "name"
)

// Highlight tokenizes code with chroma and buckets tokens into classes.
// An unknown or missing language falls back to plain text spans.
func Highlight(code, language string) []HlSpan {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return []HlSpan{{Text: code, Class: ClassText}}
	}
	var spans []HlSpan
	for {
		t := it()
		if t == chroma.EOF {
			break
		}
		spans = append(spans, HlSpan{Text: t.Value, Class: bucket(t.Type)})
	}
	return spans
}

func bucket(tt chroma.TokenType) string {
	switch {
	case tt == chroma.Text:
		return ClassText
	case tt.InCategory(chroma.Keyword):
		return ClassKeyword
	case tt == chroma.LiteralString,
		tt == chroma.LiteralStringEscape,
		tt == chroma.LiteralStringInterpol,
		tt == chroma.LiteralStringDouble,
		tt == chroma.LiteralStringSingle:
		return ClassString
	case tt.InCategory(chroma.Literal):
		return ClassLiteral
	case tt.InCategory(chroma.Number):
		return ClassNumber
	case tt.InCategory(chroma.Comment):
		return ClassComment
	case tt.InCategory(chroma.Operator):
		return ClassOperator
	case tt == chroma.Name, tt == chroma.NameFunction, tt == chroma.NameClass,
		tt == chroma.NameBuiltin, tt == chroma.NameVariable, tt == chroma.NameOther:
		return ClassName
	case tt.InCategory(chroma.Name):
		return ClassName
	default:
		return ClassText
	}
}

// ChromaStyles lists the builtin style names the app offers for code.
func ChromaStyles() []string {
	names := styles.Names()
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, n)
	}
	return out
}
