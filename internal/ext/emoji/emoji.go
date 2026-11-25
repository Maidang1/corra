// Package emoji completes ":shortcode" to emoji as it is typed, and turns
// a typed ":shortcode:" into its emoji.
package emoji

import (
	"regexp"
	"strings"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/mdext"
)

// common are the shortcodes offered first, before any is typed.
var common = []string{
	"smile", "joy", "heart", "+1", "-1", "fire", "rocket", "tada", "white_check_mark", "x",
	"warning", "bulb", "star", "eyes", "thinking", "pray", "clap", "muscle", "sparkles", "zap",
	"memo", "pushpin", "link", "lock", "key", "bug", "wrench", "hammer", "gear", "package",
	"book", "bookmark", "calendar", "clock1", "hourglass", "mag", "chart_with_upwards_trend",
	"bar_chart", "question", "exclamation", "heavy_check_mark", "arrow_right", "arrow_left",
	"point_right", "wave", "smiley", "grin", "wink", "sweat_smile", "sob", "angry", "sunglasses",
	"coffee", "beer", "pizza", "apple", "sunny", "cloud", "umbrella", "snowflake", "rainbow",
	"cat", "dog", "panda_face", "rabbit", "100", "1234", "copyright", "registered", "tm",
}

func Extension() ext.Extension {
	return ext.Extension{
		Name: "emoji",
		Completions: func(reg *doc.Registry) []editor.Completion {
			return []editor.Completion{{
				Trigger: ":",
				Items: func(q string) []editor.MenuItem {
					if q == "" {
						return nil // ":" alone, as in "a: b", opens nothing
					}
					var out []editor.MenuItem
					for _, name := range common {
						if strings.HasPrefix(name, q) {
							if e, ok := mdext.EmojiFor(name); ok {
								out = append(out, editor.MenuItem{Title: e + "  " + name, Hint: ":" + name + ":", Value: e})
							}
						}
					}
					if e, ok := mdext.EmojiFor(q); ok && len(out) == 0 {
						out = append(out, editor.MenuItem{Title: e + "  " + q, Hint: ":" + q + ":", Value: e})
					}
					return out
				},
				Apply: func(it editor.MenuItem) doc.Command { return doc.InsertText(it.Value.(string)) },
			}}
		},
		InputRules: func(reg *doc.Registry) []editor.InputRule {
			re := regexp.MustCompile(`(^|[^\w:]):([a-z0-9_+\-]+):$`)
			return []editor.InputRule{{
				Match: func(before string) []string {
					m := re.FindStringSubmatch(before)
					if m == nil {
						return nil
					}
					if _, ok := mdext.EmojiFor(m[2]); !ok {
						return nil
					}
					return m
				},
				Handle: func(s doc.State, tx *doc.Tx, m []string) bool {
					sel, ok := s.TextSel()
					if !ok || !sel.Empty() {
						return false
					}
					e, _ := mdext.EmojiFor(m[2])
					n := doc.UTF16Len(m[2]) + 2
					leaf := s.Node(sel.Head.Leaf)
					tx.ReplaceText(leaf.ID(), sel.Head.Offset-n, sel.Head.Offset, doc.Text(e, leaf.Inline().MarksAt(sel.Head.Offset)...))
					tx.SetSelection(doc.Caret(doc.Point{Leaf: leaf.ID(), Offset: sel.Head.Offset - n + doc.UTF16Len(e)}))
					return true
				},
			}}
		},
	}
}
