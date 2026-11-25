//go:build linux

package spell

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/egoist/mygo/ui"
)

// hunspellChecker checks words with the system's hunspell, if installed:
// the library is loaded on first use, so Corra runs fine without it.
type hunspellChecker struct {
	once   sync.Once
	handle uintptr
	spell  func(handle uintptr, word string) int
}

func newChecker() Checker { return &hunspellChecker{} }

// Misspelled reports a paragraph's misspelled words. Hunspell judges one
// word at a time, so the paragraph is split here; the words carry their
// UTF-16 ranges.
func (h *hunspellChecker) Misspelled(text string) []ui.TextInputRange {
	h.once.Do(h.load)
	if h.spell == nil {
		return nil
	}
	var out []ui.TextInputRange
	rs := []rune(text)
	for _, w := range words(text) {
		if h.spell(h.handle, string(rs[w.I:w.J])) == 0 {
			out = append(out, ui.TextInputRange{Start: w.From16, End: w.To16})
		}
	}
	return out
}

// load finds the hunspell library and a dictionary, or gives up quietly:
// a system without them keeps editing, just unchecked.
func (h *hunspellChecker) load() {
	lib, err := openLibrary("libhunspell-1.7.so.0", "libhunspell-1.7.so",
		"libhunspell-1.6.so.0", "libhunspell-1.6.so", "libhunspell.so.1", "libhunspell.so")
	if err != nil {
		return
	}
	var create func(aff, dic string) uintptr
	createSym, err := purego.Dlsym(lib, "Hunspell_create")
	if err != nil {
		return
	}
	purego.RegisterFunc(&create, createSym)
	spellSym, err := purego.Dlsym(lib, "Hunspell_spell")
	if err != nil {
		return
	}
	purego.RegisterFunc(&h.spell, spellSym)
	aff, dic, ok := dictionaries()
	if !ok {
		return
	}
	if h.handle = create(aff, dic); h.handle == 0 {
		h.spell = nil
	}
}

// openLibrary dlopens the first library that exists.
func openLibrary(names ...string) (uintptr, error) {
	var err error
	for _, n := range names {
		var lib uintptr
		if lib, err = purego.Dlopen(n, purego.RTLD_NOW|purego.RTLD_LOCAL); err == nil {
			return lib, nil
		}
	}
	return 0, err
}

// dictionaries finds an affix file and its dictionary on the usual distro
// paths, preferring the user's language.
func dictionaries() (aff, dic string, ok bool) {
	langs := candidateLanguages()
	dirs := []string{
		"/usr/share/hunspell",
		"/usr/share/myspell/dicts",
		"/usr/share/myspell",
		"/usr/lib/hunspell",
		"/usr/local/share/hunspell",
	}
	for _, dir := range dirs {
		for _, lang := range langs {
			a, d := filepath.Join(dir, lang+".aff"), filepath.Join(dir, lang+".dic")
			if fileExists(a) && fileExists(d) {
				return a, d, true
			}
		}
		if a, d, found := anyDictionary(dir); found {
			return a, d, true
		}
	}
	return "", "", false
}

// candidateLanguages lists the languages to look a dictionary up for: the
// locale's, its parent, then the common English ones.
func candidateLanguages() []string {
	lang := os.Getenv("LC_ALL")
	if lang == "" {
		lang = os.Getenv("LC_MESSAGES")
	}
	if lang == "" {
		lang = os.Getenv("LANG")
	}
	lang, _, _ = strings.Cut(lang, ".")
	lang, _, _ = strings.Cut(lang, "@")
	if lang == "" || lang == "C" || lang == "POSIX" {
		lang = "en_US"
	}
	var out []string
	if lang != "en_US" {
		out = append(out, lang)
		if parent, _, _ := strings.Cut(lang, "_"); parent != lang {
			out = append(out, parent)
		}
	}
	return append(out, "en_US", "en_GB")
}

// anyDictionary falls back to the first pair of files in a directory.
func anyDictionary(dir string) (aff, dic string, ok bool) {
	dics, err := filepath.Glob(filepath.Join(dir, "*.dic"))
	if err != nil || len(dics) == 0 {
		return "", "", false
	}
	for _, d := range dics {
		a := strings.TrimSuffix(d, ".dic") + ".aff"
		if fileExists(a) {
			return a, d, true
		}
	}
	return "", "", false
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
