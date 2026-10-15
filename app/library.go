// The notes library: a folder of Markdown files presented the way Apple
// Notes presents its storage — folders in the sidebar, dated cards in the
// list. Everything here is pure file/string logic, unit-testable headless.
package app

import (
	"fmt"
	"io/fs"
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
	Source  string    // full Markdown source, indexed for search
	ModTime time.Time // last modification, what the list sorts and groups by
}

// Library holds the scanned state of the notes folder.
type Library struct {
	Root       string
	Notes      []Note // sorted by ModTime, most recent first
	trashNotes []Note
	folders    []string
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
	if root == "" {
		return lib
	}
	dir, err := os.OpenRoot(root)
	if err != nil {
		return lib
	}
	defer dir.Close()
	_ = fs.WalkDir(dir.FS(), ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		inTrash := rel == TrashName || strings.HasPrefix(rel, TrashName+"/")
		if d.IsDir() {
			if d.Name() == TrashName && rel != TrashName {
				return fs.SkipDir
			}
			if rel != "." && !inTrash {
				lib.folders = append(lib.folders, rel)
			}
			return nil
		}
		if !d.Type().IsRegular() || !mdExt(rel) {
			return nil
		}
		data, err := dir.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		folder := filepath.Dir(filepath.FromSlash(rel))
		if folder == "." {
			folder = ""
		}
		if inTrash {
			folder = TrashName
		}
		title, snippet := NoteMeta(string(data))
		note := Note{
			Path:    filepath.Join(root, filepath.FromSlash(rel)),
			Folder:  filepath.ToSlash(folder),
			Title:   title,
			Snippet: snippet,
			Source:  string(data),
			ModTime: info.ModTime(),
		}
		if inTrash {
			lib.trashNotes = append(lib.trashNotes, note)
		} else {
			lib.Notes = append(lib.Notes, note)
		}
		return nil
	})
	for _, notes := range [][]Note{lib.Notes, lib.trashNotes} {
		sort.Slice(notes, func(i, j int) bool {
			if !notes[i].ModTime.Equal(notes[j].ModTime) {
				return notes[i].ModTime.After(notes[j].ModTime)
			}
			return notes[i].Path < notes[j].Path
		})
	}
	return lib
}

// Folders returns the distinct folder names, including empty directories,
// sorted, "" (root) excluded — root notes show under "全部".
func (l *Library) Folders() []string {
	seen := map[string]bool{}
	var out []string
	for _, folder := range l.folders {
		seen[folder] = true
		out = append(out, folder)
	}
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

// Filter returns the notes of a folder ("" = all active notes) whose source
// contains query (case-insensitive, empty matches all), keeping the
// ModTime ordering.
func (l *Library) Filter(folder, query string) []Note {
	q := strings.ToLower(strings.TrimSpace(query))
	notes := l.Notes
	if folder == TrashName {
		notes = l.trashNotes
	}
	out := make([]Note, 0, len(notes))
	for _, n := range notes {
		if folder != "" && n.Folder != folder {
			continue
		}
		if q != "" &&
			!strings.Contains(strings.ToLower(n.Source), q) &&
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
	rel, err := libraryFolder(folder, true)
	if err != nil {
		return "", err
	}
	dir, err := openLibraryRoot(root, true)
	if err != nil {
		return "", err
	}
	defer dir.Close()
	if err := checkLibraryPath(dir, rel, true); err != nil {
		return "", err
	}
	if err := dir.MkdirAll(rel, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("备忘录 %s.md", time.Now().Format("2006-01-02 15.04.05"))
	for i := 1; ; i++ {
		path := filepath.Join(rel, numberedName(name, i))
		f, err := dir.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, writeErr := f.WriteString("# 新备忘录\n\n")
		closeErr := f.Close()
		if writeErr != nil {
			_ = dir.Remove(path)
			return "", writeErr
		}
		if closeErr != nil {
			_ = dir.Remove(path)
			return "", closeErr
		}
		return filepath.Join(root, path), nil
	}
}

// DeleteNote moves a note into the library's trash; a note already in
// the trash is removed permanently. The caller rescans the library.
func DeleteNote(root, path string) error {
	dir, err := openLibraryRoot(root, false)
	if err != nil {
		return err
	}
	defer dir.Close()
	rel, err := libraryNotePath(dir, root, path)
	if err != nil {
		return err
	}
	if isTrashPath(rel) {
		return dir.Remove(rel)
	}
	if err := checkLibraryPath(dir, TrashName, true); err != nil {
		return err
	}
	if err := dir.MkdirAll(TrashName, 0o755); err != nil {
		return err
	}
	_, err = moveNoteWithAttachments(dir, root, rel, TrashName)
	return err
}

// TrashCount returns how many notes the trash holds.
func TrashCount(root string) int {
	dir, err := openLibraryRoot(root, false)
	if err != nil {
		return 0
	}
	defer dir.Close()
	count := 0
	_ = fs.WalkDir(dir.FS(), TrashName, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() && mdExt(path) {
			count++
		}
		return nil
	})
	return count
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
