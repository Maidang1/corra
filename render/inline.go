package render

// Inline is an inline segment of a paragraph, heading, quote or list item.
type InlineKind int

const (
	InlineText          InlineKind = iota
	InlineBold                     // **x** or __x__
	InlineItalic                   // *x* or _x__
	InlineCode                     // `x`
	InlineStrikethrough            // ~~x~~
	InlineLink                     // [x](url)
	InlineImage                    // ![alt](url)
	InlineMath                     // $x$
	InlineHardBreak                // trailing two spaces / backslash newline
	InlineHighlight                // ==x==
	InlineUnderline                // <u>x</u> or <ins>x</ins>
	InlineSuperscript              // ^x^ or <sup>x</sup>
	InlineSubscript                // ~x~ or <sub>x</sub>
	InlineKbd                      // <kbd>x</kbd>
)

// Inline is one run of inline content with uniform formatting.
type Inline struct {
	Kind  InlineKind
	Text  string
	URL   string
	Title string
	// Children holds nested formatting (e.g. bold inside italic).
	Children []Inline
}

// PlainText renders inlines as plain text (used for outline, titles).
func PlainText(src string) string {
	ins := ParseInlines(src)
	var s string
	for _, in := range ins {
		s += inlinePlain(in)
	}
	return s
}

func inlinePlain(in Inline) string {
	if len(in.Children) > 0 {
		var s string
		for _, ch := range in.Children {
			s += inlinePlain(ch)
		}
		return s
	}
	return in.Text
}
