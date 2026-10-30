// Package mgdoc maps Markdown onto MyGo's rich-text model without losing what
// that model cannot show. Text MyGo handles is edited in place; everything
// else (tables, math, footnotes, callouts, HTML, diagrams) is kept as a
// protected block whose source is written back byte for byte.
package mgdoc

import (
	"regexp"
	"strings"
)

var (
	fenceOpen = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")
	mathOpen  = regexp.MustCompile(`^ {0,3}\$\$`)
)

// Split cuts source into blocks apart by blank lines. Fenced code and $$
// math keep their inner blank lines.
func Split(source string) []string {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	var segs []string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			segs = append(segs, strings.Join(cur, "\n"))
			cur = nil
		}
	}
	lines := strings.Split(source, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if len(cur) == 0 {
			if m := fenceOpen.FindStringSubmatch(line); m != nil {
				fence := m[1]
				cur = append(cur, line)
				for i++; i < len(lines); i++ {
					cur = append(cur, lines[i])
					if closesFence(lines[i], fence) {
						break
					}
				}
				flush()
				continue
			}
			if mathOpen.MatchString(line) && !strings.Contains(strings.TrimSpace(line)[2:], "$$") {
				cur = append(cur, line)
				for i++; i < len(lines); i++ {
					cur = append(cur, lines[i])
					if strings.Contains(lines[i], "$$") {
						break
					}
				}
				flush()
				continue
			}
		}
		cur = append(cur, line)
	}
	flush()
	return segs
}

func closesFence(line, fence string) bool {
	t := strings.TrimSpace(line)
	return len(t) >= len(fence) && strings.Trim(t, fence[:1]) == ""
}
