package mdcodec

import (
	"regexp"
	"strings"
)

// Characters that are Markdown syntax anywhere in text, and those written
// as character references at the edges of a piece of text.
const (
	alwaysEscape = "\\*_`[]<~"
	edgeEncode   = "*_~"
)

var (
	lineStartList    = regexp.MustCompile(`^([-+])(\s|$)`)
	lineStartOrdered = regexp.MustCompile(`^(\d{1,9})([.)])(\s|$)`)
	lineStartHeading = regexp.MustCompile(`^#{1,6}(\s|$)`)
	lineStartRule    = regexp.MustCompile(`^(=+|-+)\s*$`)
)

// escape makes s read back as the same text. At the start of a line it
// also escapes what would start a block there, and drops leading spaces,
// which Markdown would drop or read as indented code.
func (w *Writer) escape(s string, lineStart bool) string {
	if lineStart {
		s = strings.TrimLeft(s, " \t")
	}
	extra := w.c.rules.Escape
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case strings.IndexByte(w.c.rules.EscapeRuns, c) >= 0 && i+1 < len(s) && s[i+1] == c:
			b.WriteByte('\\')
			b.WriteByte(c)
		case c == '!' && i == len(s)-1:
			// "!" before a link's "[" would make an image.
			b.WriteString(`\!`)
		case strings.IndexByte(alwaysEscape, c) >= 0 || strings.IndexByte(extra, c) >= 0:
			b.WriteByte('\\')
			b.WriteByte(c)
		case c == '&' && i+1 < len(s) && (isAlnum(s[i+1]) || s[i+1] == '#'):
			b.WriteString(`\&`)
		default:
			b.WriteByte(c)
		}
	}
	out := b.String()
	if !lineStart {
		return out
	}
	switch {
	case lineStartHeading.MatchString(out), strings.HasPrefix(out, ">"), lineStartList.MatchString(out), lineStartRule.MatchString(out):
		return `\` + out
	case lineStartOrdered.MatchString(out):
		m := lineStartOrdered.FindStringSubmatchIndex(out)
		return out[:m[3]] + `\` + out[m[3]:]
	}
	return out
}

func isAlnum(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// Destination writes a link or image destination with an optional title:
// "(url)" or "(url "title")", angle-bracketed when the URL has spaces or
// unbalanced parentheses.
func Destination(url, title string) string {
	if url == "" || strings.ContainsAny(url, " <>\n") || strings.Count(url, "(") != strings.Count(url, ")") {
		url = "<" + strings.NewReplacer("<", `\<`, ">", `\>`, "\n", " ").Replace(url) + ">"
	}
	if title == "" {
		return "(" + url + ")"
	}
	return "(" + url + ` "` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(title) + `")`
}
