package doc

// Offsets inside text count UTF-16 code units, the unit of MyGo's
// TextInputClient and TextLayout, so the editor converts nothing per key.

// UTF16Len returns the number of UTF-16 code units in s. An invalid byte
// counts as one unit, as it decodes to U+FFFD.
func UTF16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// byteOffset returns the byte index in s of UTF-16 offset off. An offset
// inside a surrogate pair rounds down to the rune's start; offsets beyond
// s clamp to its end.
func byteOffset(s string, off int) int {
	if off <= 0 {
		return 0
	}
	n := 0
	for i, r := range s {
		w := 1
		if r >= 0x10000 {
			w = 2
		}
		if n+w > off {
			return i
		}
		n += w
	}
	return len(s)
}
