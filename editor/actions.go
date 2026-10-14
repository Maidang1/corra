package editor

import (
	"fmt"
	"strings"
)

// The actions below compute Edit values for the formatting toolbar and
// shortcuts. Selection arguments are rune offsets; a collapsed selection is
// a caret. Actions are pure functions of (source, selection) so they unit
// test cleanly.

// lineBounds returns the rune range of the line containing pos.
func lineBounds(rs []rune, pos int) (start, end int) {
	if pos > len(rs) {
		pos = len(rs)
	}
	start = pos
	for start > 0 && rs[start-1] != '\n' {
		start--
	}
	end = pos
	for end < len(rs) && rs[end] != '\n' {
		end++
	}
	return
}

// selOrWord returns the selection, or the word around the caret when the
// selection is collapsed.
func selOrWord(rs []rune, start, end int) (int, int) {
	if start != end {
		return start, end
	}
	isWord := func(r rune) bool {
		return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
	}
	s, e := start, end
	for s > 0 && isWord(rs[s-1]) {
		s--
	}
	for e < len(rs) && isWord(rs[e]) {
		e++
	}
	if s == e {
		return start, end
	}
	return s, e
}

// ToggleWrap wraps (or unwraps) the selection in marker: **bold**, *italic*…
func ToggleWrap(src, marker string, start, end int) Edit {
	rs := []rune(src)
	s, e := selOrWord(rs, start, end)
	if s == e {
		// Nothing selected: insert markers around the caret.
		return Edit{Start: s, End: e, Text: marker + marker, Caret: s + len([]rune(marker))}
	}
	inner := string(rs[s:e])
	if strings.HasPrefix(inner, marker) && strings.HasSuffix(inner, marker) && len(inner) >= 2*len(marker) {
		un := inner[len(marker) : len(inner)-len(marker)]
		return Edit{Start: s, End: e, Text: un, Caret: s + len([]rune(un))}
	}
	wrapped := marker + inner + marker
	return Edit{Start: s, End: e, Text: wrapped, Caret: s + len([]rune(wrapped))}
}

// ToggleHeading cycles the heading level of the spanned lines:
// paragraph -> h1 -> h2 ... -> h6 -> paragraph.
func ToggleHeading(src string, start, end int) Edit {
	rs := []rune(src)
	ls, _ := lineBounds(rs, start)
	_, le := lineBounds(rs, end)
	lines := strings.Split(string(rs[ls:le]), "\n")
	out := make([]string, len(lines))
	for i, l := range lines {
		trim := strings.TrimLeft(l, " \t")
		indent := l[:len(l)-len(trim)]
		level := 0
		for level < len(trim) && trim[level] == '#' {
			level++
		}
		body := trim
		if level < len(trim) && trim[level] == ' ' {
			body = trim[level+1:]
		} else if level > 0 {
			level = 0 // "#x" is not a heading
		}
		switch {
		case level == 0:
			out[i] = indent + "# " + body
		case level >= 6:
			out[i] = indent + body
		default:
			out[i] = indent + strings.Repeat("#", level+1) + " " + body
		}
	}
	newText := strings.Join(out, "\n")
	return Edit{Start: ls, End: le, Text: newText, Caret: ls + len([]rune(newText))}
}

// ToggleListMarker prefixes (or removes) a list marker on the spanned lines.
// ordered selects "1." over "-".
func ToggleListMarker(src string, ordered bool, start, end int) Edit {
	rs := []rune(src)
	ls, _ := lineBounds(rs, start)
	_, le := lineBounds(rs, end)
	lines := strings.Split(string(rs[ls:le]), "\n")
	marker := "- "
	if ordered {
		marker = "1. "
	}
	allMarked := true
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if !hasListMarker(l) {
			allMarked = false
			break
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			out[i] = l
			continue
		}
		if allMarked {
			out[i] = stripListMarkerRunes(l)
		} else {
			indent := l[:len(l)-len(strings.TrimLeft(l, " \t"))]
			out[i] = indent + marker + strings.TrimLeft(l, " \t")
		}
	}
	newText := strings.Join(out, "\n")
	return Edit{Start: ls, End: le, Text: newText, Caret: ls + len([]rune(newText))}
}

func hasListMarker(l string) bool {
	t := strings.TrimLeft(l, " \t")
	if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") || strings.HasPrefix(t, "+ ") {
		return true
	}
	i := 0
	for i < len(t) && t[i] >= '0' && t[i] <= '9' {
		i++
	}
	return i > 0 && i+1 < len(t) && t[i] == '.' && t[i+1] == ' '
}

func stripListMarkerRunes(l string) string {
	t := strings.TrimLeft(l, " \t")
	indent := l[:len(l)-len(t)]
	if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") || strings.HasPrefix(t, "+ ") {
		return indent + t[2:]
	}
	i := 0
	for i < len(t) && t[i] >= '0' && t[i] <= '9' {
		i++
	}
	if i > 0 && i+1 < len(t) && t[i] == '.' && t[i+1] == ' ' {
		return indent + t[i+2:]
	}
	return l
}

// ToggleQuote prefixes (or removes) "> " on the spanned lines.
func ToggleQuote(src string, start, end int) Edit {
	rs := []rune(src)
	ls, _ := lineBounds(rs, start)
	_, le := lineBounds(rs, end)
	lines := strings.Split(string(rs[ls:le]), "\n")
	allQuoted := true
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if !strings.HasPrefix(strings.TrimLeft(l, " \t"), ">") {
			allQuoted = false
			break
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			out[i] = l
			continue
		}
		if allQuoted {
			t := strings.TrimLeft(l, " \t")
			indent := l[:len(l)-len(t)]
			t = strings.TrimPrefix(t, "> ")
			if strings.HasPrefix(t, ">") {
				t = strings.TrimPrefix(t, ">")
			}
			out[i] = indent + t
		} else {
			out[i] = "> " + l
		}
	}
	newText := strings.Join(out, "\n")
	return Edit{Start: ls, End: le, Text: newText, Caret: ls + len([]rune(newText))}
}

// InsertLink wraps the selection as [text](url), or inserts a template.
func InsertLink(src, url string, start, end int) Edit {
	rs := []rune(src)
	s, e := selOrWord(rs, start, end)
	label := string(rs[s:e])
	if label == "" {
		label = "链接文字"
	}
	if url == "" {
		url = "https://"
	}
	text := fmt.Sprintf("[%s](%s)", label, url)
	return Edit{Start: s, End: e, Text: text, Caret: s + len([]rune(text))}
}

// InsertTable inserts a 3x3 table at the caret.
func InsertTable(src string, caret int) Edit {
	const table = "\n| 列1 | 列2 | 列3 |\n| --- | --- | --- |\n|  |  |  |\n|  |  |  |\n"
	return Edit{Start: caret, End: caret, Text: table, Caret: caret + len([]rune(table))}
}

// InsertCodeBlock inserts a fenced code block at the caret.
func InsertCodeBlock(src, language string, caret int) Edit {
	text := "\n```" + language + "\n\n```\n"
	return Edit{Start: caret, End: caret, Text: text, Caret: caret + len([]rune("\n```"+language+"\n"))}
}

// InsertTask inserts a task list item.
func InsertTask(src string, caret int) Edit {
	return Edit{Start: caret, End: caret, Text: "\n- [ ] ", Caret: caret + len([]rune("\n- [ ] "))}
}

// InsertMath inserts a display math block.
func InsertMath(src string, caret int) Edit {
	return Edit{Start: caret, End: caret, Text: "\n$$\n\n$$\n", Caret: caret + 4}
}
