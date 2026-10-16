package mdext

import (
	"strings"
	"unicode"
)

// Typora sizes a figure by hanging a suffix off the end of the destination:
//
//	![alt](photo.png =300x200)
//	![alt](photo.png =50%)
//	![alt](photo.png =400x)
//	![alt](photo.png =x200)
//
// goldmark only accepts a destination holding a space once it is wrapped in
// angle brackets, so the bare form above arrives as plain text. richtext, the
// preview and the exporters all read the suffix through here so they cannot
// disagree about what counts as a size.

// ImageSizeValid reports whether spec names a size a figure may be drawn at: a
// percentage, a width, a height, or both. An empty or malformed suffix is not a
// size, so "photo.png =tall" stays ordinary text.
func ImageSizeValid(spec string) bool {
	if spec == "" {
		return false
	}
	if percent, ok := strings.CutSuffix(spec, "%"); ok {
		return imageDigits(percent)
	}
	width, height, _ := strings.Cut(spec, "x")
	if strings.Contains(height, "x") {
		return false
	}
	if width != "" && !imageDigits(width) {
		return false
	}
	if height != "" && !imageDigits(height) {
		return false
	}
	return width != "" || height != ""
}

// imageDigits accepts a plain positive count with no sign, separator or
// leading zero padding beyond five digits, which is past any real figure.
func imageDigits(s string) bool {
	if s == "" || len(s) > 5 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// SizedImage recognizes a paragraph that holds nothing but one image, with or
// without a size suffix. It returns the alternative text, the destination and
// the size suffix, which is "" when the figure is drawn at its natural size.
func SizedImage(text string) (alt, url, spec string, ok bool) {
	alt, url, _, spec, ok = imageParts(text)
	return alt, url, spec, ok
}

// imageParts takes a figure's source apart into its alternative text, its URL,
// its quoted title and its size suffix.
func imageParts(text string) (alt, url, title, spec string, ok bool) {
	line := strings.TrimSpace(text)
	if !strings.HasPrefix(line, "![") {
		return "", "", "", "", false
	}
	rest := line[2:]
	alt, destination, found := strings.Cut(rest, "](")
	if !found || strings.ContainsAny(alt, "[]") {
		return "", "", "", "", false
	}
	if !strings.HasSuffix(destination, ")") {
		return "", "", "", "", false
	}
	url, title, spec, ok = SplitImageDestination(destination[:len(destination)-1])
	// A "=" cannot open a URL, so "=300x200" on its own is not a figure.
	if !ok || url == "" || url[0] == '=' {
		return "", "", "", "", false
	}
	return alt, url, strings.Trim(title, `"'`), spec, true
}

// SplitImageDestination takes an image destination apart into its URL, its
// quoted title and its size suffix. The suffix only counts when it follows the
// URL and any title after whitespace, so the "=" of a query string is left
// alone. ok is false when the destination is not understood at all, which
// leaves the caller's own reading of it standing.
func SplitImageDestination(destination string) (url, title, spec string, ok bool) {
	rest := strings.TrimSpace(destination)
	if rest == "" {
		return "", "", "", false
	}
	if rest[0] == '<' {
		// An angle-bracketed destination keeps its spaces, so the size suffix
		// is whatever follows the closing bracket.
		closing := strings.IndexByte(rest, '>')
		if closing < 0 {
			return "", "", "", false
		}
		url, rest = rest[1:closing], strings.TrimSpace(rest[closing+1:])
	} else {
		url, rest = imageField(rest)
	}
	if url == "" || strings.ContainsAny(url, "<>") {
		return "", "", "", false
	}
	if rest != "" && (rest[0] == '"' || rest[0] == '\'') {
		quote := rest[0]
		closing := strings.IndexByte(rest[1:], quote)
		if closing < 0 {
			return "", "", "", false
		}
		title = rest[:closing+2] // the quotes are kept, so it can be written back
		rest = strings.TrimLeft(rest[closing+2:], " \t")
	}
	if rest == "" {
		return url, title, "", true
	}
	if rest[0] != '=' {
		return "", "", "", false
	}
	if spec = strings.TrimSpace(rest[1:]); !ImageSizeValid(spec) {
		return "", "", "", false
	}
	return url, title, spec, true
}

// imageField splits a destination at its first whitespace run: the URL ends
// there and the rest carries the title and the size suffix.
func imageField(destination string) (url, rest string) {
	i := strings.IndexFunc(destination, unicode.IsSpace)
	if i < 0 {
		return destination, ""
	}
	return destination[:i], strings.TrimLeft(destination[i:], " \t\r\n")
}
