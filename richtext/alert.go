package richtext

import (
	"strings"

	"github.com/Maidang1/corra/internal/mdext"
)

// GitHub renders a blockquote whose first line is [!NOTE] (or TIP, IMPORTANT,
// WARNING, CAUTION) as a callout instead of a plain quote. Corra keeps the
// marker in Raw and Runs so an untouched alert still round-trips byte for
// byte, and records the label here so the native editor and the exporters can
// style the block without re-reading the source.
//
// The marker rule itself lives in mdext so richtext, render and the native
// editor cannot disagree about what counts as a callout.

// quoteAlert reports the callout label carried by one quote paragraph's first
// line, or "" when it is an ordinary quote. Any nesting depth counts: the
// marker line of "> > [!NOTE]" is still a marker.
func quoteAlert(p Paragraph) string {
	if p.Kind != "quote" {
		return ""
	}
	return mdext.AlertLabel(firstLine(p.Raw))
}

// alertGroupContinues reports whether a quote paragraph still belongs to the
// callout opened by the paragraph before it. A blank ">" line keeps one
// blockquote together; a blank source line starts a new one.
func alertGroupContinues(previous Paragraph) bool {
	if previous.Kind != "quote" || previous.Alert == "" {
		return false
	}
	return strings.Contains(strings.ReplaceAll(previous.Separator, "\r", ""), ">")
}

// applyAlerts labels every paragraph of a blockquote once its first line opens
// a callout, so a multi-paragraph alert keeps its styling after a blank ">"
// line instead of decaying into a plain quote.
func applyAlerts(paragraphs []Paragraph) []Paragraph {
	for i := range paragraphs {
		if paragraphs[i].Kind != "quote" {
			continue
		}
		if i > 0 && alertGroupContinues(paragraphs[i-1]) {
			paragraphs[i].Alert = paragraphs[i-1].Alert
			continue
		}
		paragraphs[i].Alert = quoteAlert(paragraphs[i])
	}
	return paragraphs
}

func firstLine(source string) string {
	if i := strings.IndexByte(source, '\n'); i >= 0 {
		return source[:i]
	}
	return source
}
