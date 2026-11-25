// Package spell connects the editor to the operating system's spell
// checker: NSSpellChecker on macOS, the Windows Spell Checking API
// (ISpellChecker) on Windows and hunspell on Linux. New returns nil where
// the platform has no checker to offer, as Linux without hunspell
// installed.
package spell

import (
	"unicode"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"
)

// Checker is what the editor takes as its SpellChecker: the ranges, in
// UTF-16 units, of a text's misspelled words.
type Checker interface {
	Misspelled(text string) []ui.TextInputRange
}

// New returns the system's spell checker, or nil where there is none.
func New() Checker { return newChecker() }

// A word a dictionary spell checks: a run of letters, apostrophes allowed
// inside ("don't", "l'école"). From and To are UTF-16 offsets, I and J rune
// indexes into the text.
type word struct {
	I, J   int
	From16 int
	To16   int
}

// words splits prose into the words a dictionary spell checks, with their
// UTF-16 ranges. Digits, punctuation and everything that is not a letter
// separate words. Scripts written without spaces between words, as Chinese
// and Japanese, break words too: a dictionary has no way to judge half a
// sentence, and checking every run of Han characters would underline all of
// it.
func words(text string) []word {
	rs := []rune(text)
	u16 := make([]int, len(rs)+1)
	for i, r := range rs {
		u16[i+1] = u16[i] + utf16.RuneLen(r)
	}
	var out []word
	for i := 0; i < len(rs); {
		if !wordRune(rs[i]) {
			i++
			continue
		}
		j := i + 1
		for j < len(rs) && (wordRune(rs[j]) || apostrophe(rs[j]) && j+1 < len(rs) && wordRune(rs[j+1])) {
			j++
		}
		out = append(out, word{I: i, J: j, From16: u16[i], To16: u16[j]})
		i = j
	}
	return out
}

// wordRune reports whether r is a letter a dictionary knows words of.
func wordRune(r rune) bool {
	return unicode.IsLetter(r) && !spaceless(r)
}

// apostrophe reports whether r joins the words around it into one.
func apostrophe(r rune) bool {
	return r == '\'' || r == '’'
}

// spaceless reports whether r belongs to a script that writes whole
// sentences without spaces, which a word list cannot check. The prolonged
// sound mark and the iteration marks carry the script Common but belong
// to those scripts' writing.
func spaceless(r rune) bool {
	switch r {
	case 'ー', '々', '〆', 'ゝ', 'ゞ', 'ヽ', 'ヾ':
		return true
	}
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hangul, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Thai, r) ||
		unicode.Is(unicode.Lao, r) ||
		unicode.Is(unicode.Khmer, r) ||
		unicode.Is(unicode.Myanmar, r)
}
