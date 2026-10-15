// Pandoc-backed import/export, as Typora offers for docx/EPUB/LaTeX etc.
// Pandoc stays an optional external tool: when it is not installed the menu
// items disable themselves.
package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PandocPath returns the pandoc binary path, or "" when unavailable.
func PandocPath() string {
	if p, err := exec.LookPath("pandoc"); err == nil {
		return p
	}
	for _, p := range []string{"/opt/homebrew/bin/pandoc", "/usr/local/bin/pandoc"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// PandocFormats are the export formats offered beyond the built-in PDF/HTML.
var PandocFormats = []struct {
	Label, Ext, PandocArg string
}{
	{"Word 文档 (.docx)", "docx", "docx"},
	{"EPUB 电子书 (.epub)", "epub", "epub"},
	{"LaTeX 源码 (.tex)", "tex", "latex"},
	{"富文本 (.rtf)", "rtf", "rtf"},
	{"OpenOffice (.odt)", "odt", "odt"},
	{"reStructuredText (.rst)", "rst", "rst"},
}

// ExportPandoc converts markdown source to the format with pandoc.
func ExportPandoc(src, pandocArg, title string) ([]byte, error) {
	pandoc := PandocPath()
	if pandoc == "" {
		return nil, fmt.Errorf("未找到 pandoc，请先安装（brew install pandoc）")
	}
	args := []string{"-f", "gfm+footnotes+tex_math_dollars", "-t", pandocArg, "--standalone", "--metadata", "title=" + title}
	cmd := exec.Command(pandoc, args...)
	cmd.Stdin = strings.NewReader(src)
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("pandoc: %s", strings.TrimSpace(string(exit.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

// ImportFormats are the file types pandoc can turn into Markdown.
var ImportFormats = []struct {
	Label string
	Exts  []string
}{
	{"Word 文档", []string{"docx"}},
	{"EPUB 电子书", []string{"epub"}},
	{"HTML 文件", []string{"html", "htm"}},
	{"LaTeX 源码", []string{"tex"}},
	{"OpenOffice 文档", []string{"odt"}},
	{"reStructuredText", []string{"rst"}},
	{"Org-mode 文档", []string{"org"}},
}

// Importable reports whether pandoc import is available for this file.
func Importable(path string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	for _, f := range ImportFormats {
		for _, e := range f.Exts {
			if e == ext {
				return PandocPath() != ""
			}
		}
	}
	return false
}

// ImportPandoc converts a file to markdown with pandoc.
func ImportPandoc(path string) (string, error) {
	pandoc := PandocPath()
	if pandoc == "" {
		return "", fmt.Errorf("未找到 pandoc，请先安装（brew install pandoc）")
	}
	cmd := exec.Command(pandoc, "-t", "gfm-raw_html", "--wrap=none", path)
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("pandoc: %s", strings.TrimSpace(string(exit.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}
