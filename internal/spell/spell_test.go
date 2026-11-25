package spell

import (
	"reflect"
	"testing"
)

// TestWords checks how prose splits into the words a dictionary spell
// checks, and that the ranges carry UTF-16 offsets.
func TestWords(t *testing.T) {
	tests := []struct {
		text string
		want []word
	}{
		{"", nil},
		{"the quick brown fox", []word{
			{0, 3, 0, 3}, {4, 9, 4, 9}, {10, 15, 10, 15}, {16, 19, 16, 19},
		}},
		// An apostrophe between letters joins the word.
		{"don't stop", []word{{0, 5, 0, 5}, {6, 10, 6, 10}}},
		{"l'école", []word{{0, 7, 0, 7}}},
		// Apostrophes only count inside a word.
		{"'quoted'", []word{{1, 7, 1, 7}}},
		{"dogs' bones", []word{{0, 4, 0, 4}, {6, 11, 6, 11}}},
		// Digits and punctuation separate words.
		{"abc123def", []word{{0, 3, 0, 3}, {6, 9, 6, 9}}},
		{"e-mail", []word{{0, 1, 0, 1}, {2, 6, 2, 6}}},
		// Scripts written without spaces are never checked.
		{"你好世界", nil},
		{"スパース テスト", nil},
		// They break words rather than joining them.
		{"你好abc世界", []word{{2, 5, 2, 5}}},
		// A surrogate pair is 2 UTF-16 units: the word after it moves.
		{"🙂 hi", []word{{2, 4, 3, 5}}},
	}
	for _, tt := range tests {
		got := words(tt.text)
		if len(got) == 0 && len(tt.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("words(%q) = %+v, want %+v", tt.text, got, tt.want)
		}
	}
}
