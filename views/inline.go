package views

import (
	"fmt"
	"strings"
)

// wordCount returns Typora-style counts for the document source.
type wordCount struct {
	Words   int // latin words + CJK characters
	Chars   int // all runes, excluding whitespace
	Lines   int
	Minutes int // estimated reading time
}

// countWords counts CJK characters individually and latin runs as words, as
// Typora does for mixed documents.
func countWords(src string) wordCount {
	wc := wordCount{Lines: strings.Count(src, "\n") + 1}
	if strings.TrimSpace(src) == "" {
		wc.Lines = 0
	}
	var latin strings.Builder
	flush := func() {
		if latin.Len() > 0 {
			wc.Words++
			latin.Reset()
		}
	}
	for _, r := range src {
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		case r > 0x2e00: // CJK & full-width: each character is a word
			flush()
			wc.Words++
			wc.Chars++
		default:
			latin.WriteRune(r)
			wc.Chars++
		}
	}
	flush()
	cjk := 0
	for _, r := range src {
		if r > 0x2e00 {
			cjk++
		}
	}
	latinWords := wc.Words - cjk
	wc.Minutes = latinWords/275 + cjk/400
	if wc.Words > 0 && wc.Minutes == 0 {
		wc.Minutes = 1
	}
	return wc
}

func (wc wordCount) String() string {
	if wc.Words == 0 {
		return "0 字"
	}
	return fmt.Sprintf("%d 字 · %d 字符 · %d 行 · 约 %d 分钟", wc.Words, wc.Chars, wc.Lines, wc.Minutes)
}
