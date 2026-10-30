package mgdoc

import (
	"regexp"
	"strings"

	"github.com/Maidang1/corra/internal/mdext"
)

// Private-use runes stand in for syntax MyGo's Markdown reader cannot carry:
// they survive parsing and escaping as ordinary text, and are swapped back
// for attachments or tagged runs after the read and for source on save.
const (
	markOpen, markClose = '\uE001', '\uE002'
	supOpen, supClose   = '\uE003', '\uE004'
	subOpen, subClose   = '\uE005', '\uE006'
	rawBase             = rune(0xF0000)
)

type blockKind int

const (
	blockNone blockKind = iota
	blockTable
	blockMath
	blockDiagram
	blockAlert
	blockFootnote
	blockHTML
	blockRule
	blockImage
	blockOther
)

var (
	tableSep    = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)
	ruleLine    = regexp.MustCompile(`^ {0,3}(-( *-){2,}|\*( *\*){2,}|_( *_){2,}) *$`)
	footnoteDef = regexp.MustCompile(`^ {0,3}\[\^[^\]]+\]:`)
	linkDef     = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:\s*\S`)
	alertMark   = regexp.MustCompile(`(?i)^ {0,3}>\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]`)
	htmlBlock   = regexp.MustCompile(`(?i)^ {0,3}(<!--|</?(div|p|details|summary|table|thead|tbody|tr|td|th|img|center|pre|section|article|iframe|video|audio|figure|figcaption|h[1-6]|ul|ol|li|blockquote|hr|br|a|span|kbd)[\s>/])`)
	loneImage   = regexp.MustCompile(`^ {0,3}!\[[^\]]*\]\([^)]*\)\s*$`)
	inlineTag   = regexp.MustCompile(`</?([a-zA-Z][a-zA-Z0-9]*)`)
	prefixQuote = regexp.MustCompile(`^[ >]*(?:[-*+]|\d+[.)])?\s*`)
)

var safeTags = map[string]bool{
	"u": true, "ins": true, "s": true, "strike": true, "del": true,
	"b": true, "strong": true, "i": true, "em": true, "br": true,
}

// classify reports whether a segment must stay a protected block.
func classify(seg string) (blockKind, string) {
	lines := strings.Split(seg, "\n")
	first := lines[0]
	if m := fenceOpen.FindStringSubmatch(first); m != nil {
		lang := ""
		if f := strings.Fields(strings.ToLower(m[2])); len(f) > 0 {
			lang = f[0]
		}
		switch lang {
		case "mermaid", "d2", "goat", "flow", "sequence":
			return blockDiagram, lang
		case "math", "latex", "tex":
			return blockMath, ""
		}
		return blockNone, ""
	}
	switch {
	case mathOpen.MatchString(first):
		return blockMath, ""
	case len(lines) >= 2 && strings.Contains(first, "|") && tableSep.MatchString(lines[1]) && strings.Contains(lines[1], "-"):
		return blockTable, ""
	case footnoteDef.MatchString(first):
		return blockFootnote, ""
	case len(lines) == 1 && ruleLine.MatchString(first):
		return blockRule, ""
	case alertMark.MatchString(first):
		return blockAlert, ""
	case len(lines) == 1 && loneImage.MatchString(first):
		return blockImage, ""
	case htmlBlock.MatchString(first) && !strings.HasPrefix(strings.TrimSpace(first), "<a ") && !strings.HasPrefix(strings.TrimSpace(first), "<span") && !strings.HasPrefix(strings.TrimSpace(first), "<kbd"):
		return blockHTML, ""
	case linkDef.MatchString(first):
		return blockOther, ""
	}
	if hasUnsafeContent(lines) {
		return blockOther, ""
	}
	return blockNone, ""
}

// hasUnsafeContent spots lists and quotes that nest what the text model
// would flatten or drop.
func hasUnsafeContent(lines []string) bool {
	inFence := false
	for _, l := range lines {
		body := prefixQuote.ReplaceAllString(l, "")
		if fenceOpen.MatchString(body) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		switch {
		case mathOpen.MatchString(body), footnoteDef.MatchString(body), alertMark.MatchString(l):
			return true
		case tableSep.MatchString(body) && strings.Contains(body, "|") && strings.Contains(body, "-"):
			return true
		case htmlBlock.MatchString(body) && !safeBlockHTML(body):
			return true
		}
		for _, m := range inlineTag.FindAllStringSubmatch(l, -1) {
			if !safeTags[strings.ToLower(m[1])] {
				return true
			}
		}
	}
	return false
}

func safeBlockHTML(body string) bool {
	for _, m := range inlineTag.FindAllStringSubmatch(body, -1) {
		if !safeTags[strings.ToLower(m[1])] {
			return false
		}
	}
	return true
}

// tokenize swaps inline syntax MyGo cannot read for private-use runes. It
// leaves code, link destinations and bare URLs alone.
func tokenize(seg string) (string, []string) {
	var raws []string
	var out strings.Builder
	inFence := false
	lines := strings.Split(seg, "\n")
	for li, line := range lines {
		if li > 0 {
			out.WriteByte('\n')
		}
		if fenceOpen.MatchString(prefixQuote.ReplaceAllString(line, "")) {
			inFence = !inFence
			out.WriteString(line)
			continue
		}
		if inFence {
			out.WriteString(line)
			continue
		}
		tokenizeLine(line, &out, &raws)
	}
	return out.String(), raws
}

func tokenizeLine(line string, out *strings.Builder, raws *[]string) {
	b := []byte(line)
	var inMark, inSup, inSub bool
	emitRaw := func(s string) {
		out.WriteRune(rawBase + rune(len(*raws)))
		*raws = append(*raws, s)
	}
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch {
		case c == '\\' && i+1 < len(b):
			out.Write(b[i : i+2])
			i++
		case c == '`':
			n := 1
			for i+n < len(b) && b[i+n] == '`' {
				n++
			}
			end := closingTicks(b, i+n, n)
			if end < 0 {
				out.Write(b[i : i+n])
				i += n - 1
				continue
			}
			out.Write(b[i : end+n])
			i = end + n - 1
		case c == ']' && i+1 < len(b) && b[i+1] == '(':
			end := indexByteFrom(b, i+2, ')')
			if end < 0 {
				out.WriteByte(c)
				continue
			}
			out.Write(b[i : end+1])
			i = end
		case c == 'h' && (hasPrefixAt(b, i, "http://") || hasPrefixAt(b, i, "https://")):
			j := i
			for j < len(b) && b[j] != ' ' && b[j] != ')' && b[j] != '>' && b[j] != '\t' {
				j++
			}
			out.Write(b[i:j])
			i = j - 1
		case c == '<':
			end := indexByteFrom(b, i+1, '>')
			if end < 0 {
				out.WriteByte(c)
				continue
			}
			out.Write(b[i : end+1])
			i = end
		case c == '$':
			if tex, n := mdext.MathSpan(b[i:]); tex != nil && n > 0 {
				emitRaw(string(b[i : i+n]))
				i += n - 1
				continue
			}
			out.WriteByte(c)
		case c == '[' && i+1 < len(b) && b[i+1] == '^':
			end := indexByteFrom(b, i+2, ']')
			if end > i+2 && !strings.ContainsAny(string(b[i+2:end]), " \t") && (end+1 >= len(b) || b[end+1] != ':') {
				emitRaw(string(b[i : end+1]))
				i = end
				continue
			}
			out.WriteByte(c)
		case c == '=' && i+1 < len(b) && b[i+1] == '=':
			switch {
			case inMark && i > 0 && !isSpace(b[i-1]):
				out.WriteRune(markClose)
				inMark = false
				i++
			case !inMark && i+2 < len(b) && !isSpace(b[i+2]) && b[i+2] != '=' && closesMark(b, i+2):
				out.WriteRune(markOpen)
				inMark = true
				i++
			default:
				out.WriteString("==")
				i++
			}
		case c == '^':
			switch {
			case inSup && i > 0 && !isSpace(b[i-1]):
				out.WriteRune(supClose)
				inSup = false
			case !inSup && i+1 < len(b) && !isSpace(b[i+1]) && closesSimple(b, i+1, '^'):
				out.WriteRune(supOpen)
				inSup = true
			default:
				out.WriteByte(c)
			}
		case c == '~' && !(i+1 < len(b) && b[i+1] == '~') && !(i > 0 && b[i-1] == '~'):
			switch {
			case inSub && i > 0 && !isSpace(b[i-1]):
				out.WriteRune(subClose)
				inSub = false
			case !inSub && i+1 < len(b) && !isSpace(b[i+1]) && closesSimple(b, i+1, '~'):
				out.WriteRune(subOpen)
				inSub = true
			default:
				out.WriteByte(c)
			}
		default:
			out.WriteByte(c)
		}
	}
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' }

func hasPrefixAt(b []byte, i int, p string) bool {
	return len(b)-i >= len(p) && string(b[i:i+len(p)]) == p
}

func indexByteFrom(b []byte, from int, c byte) int {
	for i := from; i < len(b); i++ {
		if b[i] == c {
			return i
		}
	}
	return -1
}

func closingTicks(b []byte, from, n int) int {
	for i := from; i < len(b); i++ {
		if b[i] != '`' {
			continue
		}
		j := i
		for j < len(b) && b[j] == '`' {
			j++
		}
		if j-i == n {
			return i
		}
		i = j - 1
	}
	return -1
}

// closesMark reports whether a closing == follows with no space before it.
func closesMark(b []byte, from int) bool {
	for i := from; i+1 < len(b); i++ {
		if b[i] == '=' && b[i+1] == '=' && !isSpace(b[i-1]) {
			return true
		}
	}
	return false
}

// closesSimple reports whether a closing ^ or ~ follows without spaces.
func closesSimple(b []byte, from int, c byte) bool {
	for i := from; i < len(b); i++ {
		if isSpace(b[i]) {
			return false
		}
		if b[i] == c {
			return true
		}
	}
	return false
}

// fenceInfos lists the info strings of the fences opened in a segment, in
// order; the text model drops them.
func fenceInfos(seg string) []string {
	var infos []string
	fence := ""
	for _, l := range strings.Split(seg, "\n") {
		body := prefixQuote.ReplaceAllString(l, "")
		if fence == "" {
			if m := fenceOpen.FindStringSubmatch(body); m != nil {
				fence = m[1]
				infos = append(infos, strings.TrimSpace(m[2]))
			}
			continue
		}
		if closesFence(body, fence) {
			fence = ""
		}
	}
	return infos
}
