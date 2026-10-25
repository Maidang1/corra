package nativerich

import (
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Maidang1/corra/richtext"
)

// PageCard is what a page card shows of the note it points at.
type PageCard struct {
	Title   string `json:"title,omitempty"`
	Excerpt string `json:"excerpt,omitempty"`
	Missing bool   `json:"missing,omitempty"`
}

// PageInfo reads a note for its page card: the first heading (or the file
// name) as the title and the first prose as the excerpt.
func PageInfo(path string) PageCard {
	data, err := os.ReadFile(path)
	if err != nil {
		return PageCard{Missing: true}
	}
	card := PageCard{}
	for _, p := range richtext.Parse(string(data)).Paragraphs {
		text := strings.TrimSpace(strings.Join(strings.Fields(richtext.PlainText(p)), " "))
		if text == "" {
			continue
		}
		switch p.Kind {
		case "heading":
			if card.Title == "" {
				card.Title = text
			}
		case "body", "quote", "bullet", "ordered", "checklist":
			if card.Excerpt == "" {
				card.Excerpt = text
			}
		}
		if card.Title != "" && card.Excerpt != "" {
			break
		}
	}
	if card.Title == "" {
		card.Title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	card.Excerpt = clip(card.Excerpt, 140)
	return card
}

func clip(text string, runes int) string {
	if utf8.RuneCountInString(text) <= runes {
		return text
	}
	return string([]rune(text)[:runes]) + "…"
}
