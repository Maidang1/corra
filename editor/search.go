package editor

import "strings"

// Match is one occurrence of the query.
type Match struct {
	Start, End int // rune offsets
}

// runeLen returns the rune count of s.
func runeLen(s string) int { return len([]rune(s)) }

// Find returns all case-insensitive matches of query in src, as rune offsets.
func Find(src, query string) []Match {
	if query == "" {
		return nil
	}
	lq := len([]rune(query))
	if lq == 0 {
		return nil
	}
	lower := strings.ToLower(src)
	q := strings.ToLower(query)
	var out []Match
	start := 0
	for {
		i := strings.Index(lower[start:], q)
		if i < 0 {
			break
		}
		s := runeLen(lower[:start+i])
		out = append(out, Match{Start: s, End: s + lq})
		start = start + i + len(q)
	}
	return out
}

// Replace replaces the match range with repl.
func Replace(src string, m Match, repl string) string {
	rs := []rune(src)
	if m.Start < 0 || m.End > len(rs) || m.Start > m.End {
		return src
	}
	out := make([]rune, 0, len(rs)-(m.End-m.Start)+len([]rune(repl)))
	out = append(out, rs[:m.Start]...)
	out = append(out, []rune(repl)...)
	out = append(out, rs[m.End:]...)
	return string(out)
}

// ReplaceAll replaces every occurrence of query (case-insensitive) with repl.
func ReplaceAll(src, query, repl string) string {
	if query == "" {
		return src
	}
	matches := Find(src, query)
	if len(matches) == 0 {
		return src
	}
	var out []rune
	rs := []rune(src)
	last := 0
	for _, m := range matches {
		out = append(out, rs[last:m.Start]...)
		out = append(out, []rune(repl)...)
		last = m.End
	}
	out = append(out, rs[last:]...)
	return string(out)
}
