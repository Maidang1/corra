//go:build darwin && cgo

package spell

import "testing"

// TestCocoaMisspelled checks text with the real NSSpellChecker. A system
// without spell checking dictionaries reports nothing wrong with anything;
// such an environment can only be skipped.
func TestCocoaMisspelled(t *testing.T) {
	if misspelled := cocoaMisspelled("corra spelchek test"); len(misspelled) == 0 {
		t.Skip("the system spell checker reports nothing; no dictionaries here?")
	}
	if got := cocoaMisspelled("the quick brown fox"); len(got) != 0 {
		t.Errorf("correct words flagged: %v", got)
	}
	// The ranges are UTF-16 offsets, as the editor positions them.
	got := cocoaMisspelled("🙂 spelchek")
	if len(got) != 1 || got[0].Start != 3 || got[0].End != 11 {
		t.Errorf("misspelled range after a surrogate pair = %+v, want [3,11)", got)
	}
}
