// The notes library: a folder of Markdown files presented the way Apple
// Notes presents its storage — folders in the sidebar, dated cards in the
// list. Everything here is pure file/string logic, unit-testable headless.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Maidang1/corra/render"
)

// TrashName is the library's trash subfolder, the "最近删除" of the
// sidebar. ScanLibrary never lists it as a regular folder.
const TrashName = ".trash"

// TrashDir returns the trash folder of a library root.
func TrashDir(root string) string { return filepath.Join(root, TrashName) }

// Note is one Markdown file of the library.
type Note struct {
	Path    string    // absolute path of the .md file
	Folder  string    // folder relative to the library root ("" = root)
	Title   string    // first heading or first line
	Snippet string    // a short plain-text preview of the body
	ModTime time.Time // last modification, what the list sorts and groups by
}

// Library holds the scanned state of the notes folder.
type Library struct {
	Root  string
	Notes []Note // sorted by ModTime, most recent first
}

// mdExt reports whether path is a Markdown file the library manages.
func mdExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdx", ".txt":
		return true
	}
	return false
}

// ScanLibrary walks root recursively and collects every Markdown file.
// Missing or unreadable roots yield an empty library, not an error.
func ScanLibrary(root string) *Library {
	lib := &Library{Root: root}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			if d.Name() == TrashName {
				return filepath.SkipDir // the trash is its own sidebar row
			}
			return nil
		}
		if !mdExt(path) {
			return nil
		}
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		folder, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil || folder == "." {
			folder = ""
		}
		title, snippet := NoteMeta(string(data))
		lib.Notes = append(lib.Notes, Note{
			Path:    path,
			Folder:  filepath.ToSlash(folder),
			Title:   title,
			Snippet: snippet,
			ModTime: info.ModTime(),
		})
		return nil
	})
	sort.Slice(lib.Notes, func(i, j int) bool {
		if !lib.Notes[i].ModTime.Equal(lib.Notes[j].ModTime) {
			return lib.Notes[i].ModTime.After(lib.Notes[j].ModTime)
		}
		return lib.Notes[i].Path < lib.Notes[j].Path
	})
	return lib
}

// Folders returns the distinct folder names of the library's notes,
// sorted, "" (root) excluded — root notes show under "全部".
func (l *Library) Folders() []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range l.Notes {
		if n.Folder == "" || seen[n.Folder] {
			continue
		}
		seen[n.Folder] = true
		out = append(out, n.Folder)
	}
	sort.Strings(out)
	return out
}

// Filter returns the notes of a folder ("" = all) whose title or snippet
// contains query (case-insensitive, empty matches all), keeping the
// ModTime ordering.
func (l *Library) Filter(folder, query string) []Note {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]Note, 0, len(l.Notes))
	for _, n := range l.Notes {
		if folder != "" && n.Folder != folder {
			continue
		}
		if q != "" &&
			!strings.Contains(strings.ToLower(n.Title), q) &&
			!strings.Contains(strings.ToLower(n.Snippet), q) {
			continue
		}
		out = append(out, n)
	}
	return out
}

// NoteGroup is a dated section of the notes list.
type NoteGroup struct {
	Label string
	Notes []Note
}

// GroupByDate buckets notes the way Apple Notes does: 今天, 昨天,
// 最近 7 天, 最近 30 天, 更早. now is injectable for tests.
func GroupByDate(notes []Note, now time.Time) []NoteGroup {
	today := dateOf(now)
	groups := []NoteGroup{
		{Label: "今天"},
		{Label: "昨天"},
		{Label: "最近 7 天"},
		{Label: "最近 30 天"},
		{Label: "更早"},
	}
	bucket := func(n Note) int {
		d := dateOf(n.ModTime)
		switch {
		case d.Equal(today):
			return 0
		case d.Equal(today.AddDate(0, 0, -1)):
			return 1
		case !d.Before(today.AddDate(0, 0, -7)):
			return 2
		case !d.Before(today.AddDate(0, 0, -30)):
			return 3
		default:
			return 4
		}
	}
	for _, n := range notes {
		i := bucket(n)
		groups[i].Notes = append(groups[i].Notes, n)
	}
	out := groups[:0]
	for _, g := range groups {
		if len(g.Notes) > 0 {
			out = append(out, g)
		}
	}
	return out
}

// dateOf truncates a time to its calendar day.
func dateOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// isDigits reports whether s is all ASCII digits.
func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isDigitByte(s[i]) {
			return false
		}
	}
	return len(s) > 0
}

func isDigitByte(b byte) bool { return b >= '0' && b <= '9' }

// NoteMeta derives a note's title and snippet from its source: the title
// is the first heading (markers stripped) or the first non-empty line;
// the snippet is the following text, flattened to one line.
func NoteMeta(src string) (title, snippet string) {
	lines := strings.Split(src, "\n")
	inFence := false
	var body []string
	for _, ln := range lines {
		s := strings.TrimSpace(ln)
		if strings.HasPrefix(s, "```") {
			inFence = !inFence
			continue
		}
		if s == "" || inFence {
			continue
		}
		if strings.HasPrefix(s, "#") {
			if title == "" {
				title = strings.TrimSpace(strings.TrimLeft(s, "#"))
			}
			continue
		}
		// Strip the common block markers so the preview reads as prose,
		// then flatten inline markdown (**bold**, `code`, [links]…).
		s = strings.TrimSpace(strings.TrimPrefix(s, ">"))
		if strings.HasPrefix(s, "- [ ]") || strings.HasPrefix(s, "- [x]") || strings.HasPrefix(s, "- [X]") {
			s = strings.TrimSpace(s[5:])
		}
		// A leading date token (2026/10/5, 2026-10-05…) never makes a
		// good preview; drop it, but only when it really is a date (a
		// bare number like "2026年销量" stays).
		if len(s) >= 6 && isDigits(s[:4]) {
			i, sawSep := 4, false
			for i < len(s) && (isDigitByte(s[i]) || s[i] == '/' || s[i] == '-' || s[i] == '.') {
				if !isDigitByte(s[i]) {
					sawSep = true
				}
				i++
			}
			if sawSep {
				s = strings.TrimSpace(s[i:])
			}
		}
		s = strings.TrimLeft(s, "-*|0123456789. ")
		if s == "" {
			continue
		}
		s = strings.TrimSpace(render.PlainText(s))
		if title == "" {
			title = s
			continue
		}
		body = append(body, s)
		if len([]rune(strings.Join(body, " "))) > 60 {
			break
		}
	}
	if title == "" {
		title = "新备忘录"
	}
	snippet = strings.Join(body, " ")
	r := []rune(snippet)
	if len(r) > 60 {
		snippet = string(r[:59]) + "…"
	}
	return title, snippet
}

// CreateNote makes a new note file in folder (relative to root, "" for
// root) and returns its path. The name is a timestamp, the way Apple
// Notes names notes internally; the title comes from the content.
func CreateNote(root, folder string) (string, error) {
	dir := root
	if folder != "" {
		dir = filepath.Join(root, filepath.FromSlash(folder))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("备忘录 %s.md", time.Now().Format("2006-01-02 15.04.05"))
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		// Same-second collision: bump the name.
		name = fmt.Sprintf("备忘录 %s %d.md", time.Now().Format("2006-01-02 15.04.05"), time.Now().Nanosecond()/1e6)
		path = filepath.Join(dir, name)
	}
	if err := os.WriteFile(path, []byte("# 新备忘录\n\n"), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// DeleteNote moves a note into the library's trash; a note already in
// the trash is removed permanently. The caller rescans the library.
func DeleteNote(root, path string) error {
	if !mdExt(path) {
		return fmt.Errorf("不是 Markdown 文件: %s", path)
	}
	clean := filepath.Clean(path)
	if filepath.Dir(clean) == TrashDir(root) {
		return os.Remove(clean)
	}
	if err := os.MkdirAll(TrashDir(root), 0o755); err != nil {
		return err
	}
	ext := filepath.Ext(clean)
	base := strings.TrimSuffix(filepath.Base(clean), ext)
	dest := filepath.Join(TrashDir(root), filepath.Base(clean))
	for i := 2; ; i++ {
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			break
		}
		dest = filepath.Join(TrashDir(root), fmt.Sprintf("%s (%d)%s", base, i, ext))
	}
	return os.Rename(clean, dest)
}

// TrashCount returns how many notes the trash holds.
func TrashCount(root string) int {
	entries, err := os.ReadDir(TrashDir(root))
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && mdExt(e.Name()) {
			n++
		}
	}
	return n
}

// SanitizeFileName maps a title to a safe file name (used when saving a
// note under a new name).
func SanitizeFileName(title string) string {
	r := []rune(title)
	var b strings.Builder
	lastDash := false
	for _, ch := range r {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '.' || ch == '-' || ch == '_' {
			b.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(b.String(), "-.")
	if name == "" {
		name = "note"
	}
	if len([]rune(name)) > 60 {
		name = string([]rune(name)[:60])
	}
	return name + ".md"
}
