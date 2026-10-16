package mdext

import (
	"regexp"
	"strings"
)

// GitHub renders a blockquote whose first line is [!NOTE] (or TIP, IMPORTANT,
// WARNING, CAUTION) as a callout instead of a plain quote. richtext, the
// preview, the exporters and the native editor all read the label through
// here so they cannot disagree about what counts as a marker.

var alertMarkerRe = regexp.MustCompile(`(?i)^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]`)

// AlertLabels lists the callout kinds in the order menus show them.
var AlertLabels = []string{"NOTE", "TIP", "IMPORTANT", "WARNING", "CAUTION"}

// IsAlertLabel reports whether label names one of the callout kinds.
func IsAlertLabel(label string) bool {
	for _, known := range AlertLabels {
		if strings.EqualFold(label, known) {
			return true
		}
	}
	return false
}

// AlertLabel reports the callout label carried by one blockquote line, or ""
// when the line is not a marker. The line may keep its ">" markers. The marker
// counts both on its own and with text after it, because goldmark merges
// "> [!NOTE]\n> body" into a single paragraph.
func AlertLabel(line string) string {
	label, _ := splitAlertMarker(strings.TrimLeft(strings.TrimRight(line, "\r"), "> \t"), true)
	return label
}

// AlertLabelPrefix reports the callout label at the start of a callout's first
// paragraph text together with the body that follows it. Unlike AlertLabel it
// accepts text glued straight onto the marker, because a paragraph's text
// merges the marker line with the lines under it.
func AlertLabelPrefix(text string) (label, body string) {
	return splitAlertMarker(strings.TrimLeft(text, "> \t"), false)
}

func splitAlertMarker(text string, needSpace bool) (label, body string) {
	m := alertMarkerRe.FindStringSubmatch(text)
	if m == nil {
		return "", text
	}
	rest := text[len(m[0]):]
	if needSpace && rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return "", text
	}
	return strings.ToUpper(m[1]), strings.TrimLeft(rest, " \t")
}
