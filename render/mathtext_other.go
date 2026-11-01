//go:build !darwin || !cgo

package render

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"codeberg.org/go-fonts/liberation/liberationserifbold"
	"codeberg.org/go-fonts/liberation/liberationserifbolditalic"
	"codeberg.org/go-fonts/liberation/liberationserifitalic"
	"codeberg.org/go-fonts/liberation/liberationserifregular"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Without CoreText, MathJax text leaves are outlined in pure Go: Liberation
// Serif (metric-compatible with the Times New Roman the macOS path uses) for
// Latin, then the system's CJK fonts for everything else.

var (
	mathSerifOnce sync.Once
	mathSerif     [4]*sfnt.Font // regular, bold, italic, bold italic
	mathSerifErr  error

	mathFallbackOnce sync.Once
	mathFallback     []*sfnt.Font
)

func mathSerifFont(bold, italic bool) (*sfnt.Font, error) {
	mathSerifOnce.Do(func() {
		for i, data := range [][]byte{liberationserifregular.TTF, liberationserifbold.TTF, liberationserifitalic.TTF, liberationserifbolditalic.TTF} {
			if mathSerif[i], mathSerifErr = sfnt.Parse(data); mathSerifErr != nil {
				return
			}
		}
	})
	if mathSerifErr != nil {
		return nil, fmt.Errorf("math: serif font: %w", mathSerifErr)
	}
	i := 0
	if bold {
		i |= 1
	}
	if italic {
		i |= 2
	}
	return mathSerif[i], nil
}

// mathFallbackFonts opens the first readable system fonts once. Collections
// stay open and are read on demand, so large CJK files are not loaded whole.
func mathFallbackFonts() []*sfnt.Font {
	mathFallbackOnce.Do(func() {
		for _, path := range mathFallbackPaths() {
			mathFallback = append(mathFallback, openSystemFont(path)...)
		}
	})
	return mathFallback
}

func mathFallbackPaths() []string {
	switch runtime.GOOS {
	case "windows":
		dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
		if os.Getenv("WINDIR") == "" {
			dir = `C:\Windows\Fonts`
		}
		var paths []string
		for _, name := range []string{"simsun.ttc", "msyh.ttc", "msjh.ttc", "yumin.ttf", "msgothic.ttc", "malgun.ttf", "seguisym.ttf"} {
			paths = append(paths, filepath.Join(dir, name))
		}
		return paths
	case "darwin":
		return []string{
			"/System/Library/Fonts/Supplemental/Songti.ttc",
			"/System/Library/Fonts/STHeiti Light.ttc",
			"/System/Library/Fonts/Apple Symbols.ttf",
		}
	default:
		paths := []string{
			"/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc",
			"/usr/share/fonts/noto-cjk/NotoSerifCJK-Regular.ttc",
			"/usr/share/fonts/google-noto-cjk/NotoSerifCJK-Regular.ttc",
			"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/google-noto-cjk/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
			"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
			"/usr/share/fonts/wenquanyi/wqy-zenhei/wqy-zenhei.ttc",
			"/usr/share/fonts/wenquanyi/wqy-microhei/wqy-microhei.ttc",
		}
		// Distributions place fonts differently; fontconfig knows where.
		if fc, err := exec.LookPath("fc-match"); err == nil {
			if out, err := exec.Command(fc, "--format=%{file}", "serif:lang=zh-cn").Output(); err == nil && len(out) > 0 {
				paths = append(paths, string(out))
			}
		}
		return append(paths, "/usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf")
	}
}

// openSystemFont opens a font file and returns every face in it, the
// Simplified Chinese one first (Noto CJK puts Japanese first).
func openSystemFont(path string) []*sfnt.Font {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	collection, err := sfnt.ParseCollectionReaderAt(file)
	if err != nil {
		file.Close()
		return nil
	}
	var faces []*sfnt.Font
	var buf sfnt.Buffer
	for i := 0; i < collection.NumFonts(); i++ {
		f, err := collection.Font(i)
		if err != nil {
			continue
		}
		if name, err := f.Name(&buf, sfnt.NameIDFamily); err == nil && strings.Contains(name, " SC") {
			faces = append([]*sfnt.Font{f}, faces...)
		} else {
			faces = append(faces, f)
		}
	}
	if len(faces) == 0 {
		file.Close()
	}
	return faces
}

func nativeMathTextPath(text string, size float64, bold, italic bool) (string, error) {
	serif, err := mathSerifFont(bold, italic)
	if err != nil {
		return "", err
	}
	ppem := fixed.Int26_6(size * 64)
	var buf sfnt.Buffer
	var out strings.Builder
	x := fixed.Int26_6(0)
	var prevFont *sfnt.Font
	var prevGlyph sfnt.GlyphIndex
	for _, r := range text {
		f, glyph := serif, sfnt.GlyphIndex(0)
		if g, err := serif.GlyphIndex(&buf, r); err == nil {
			glyph = g
		}
		if glyph == 0 {
			for _, fallback := range mathFallbackFonts() {
				if g, err := fallback.GlyphIndex(&buf, r); err == nil && g != 0 {
					f, glyph = fallback, g
					break
				}
			}
		}
		if glyph == 0 {
			if unicode.IsSpace(r) {
				// No font maps it; advance like a regular space.
				if g, err := serif.GlyphIndex(&buf, ' '); err == nil && g != 0 {
					if adv, err := serif.GlyphAdvance(&buf, g, ppem, font.HintingNone); err == nil {
						x += adv
					}
				}
				prevFont = nil
				continue
			}
			return "", fmt.Errorf("math: no font has an outline for %q", text)
		}
		if f == prevFont {
			if kern, err := f.Kern(&buf, prevGlyph, glyph, ppem, font.HintingNone); err == nil {
				x += kern
			}
		}
		segments, err := f.LoadGlyph(&buf, glyph, ppem, nil)
		if err != nil {
			return "", fmt.Errorf("math: outline for %q: %w", r, err)
		}
		// sfnt outlines are Y-down with the origin on the baseline, the same
		// frame as the SVG text leaf they replace.
		open := false
		for _, s := range segments {
			switch s.Op {
			case sfnt.SegmentOpMoveTo:
				if open {
					out.WriteByte('Z')
				}
				out.WriteByte('M')
				writeMathPoint(&out, x, s.Args[0])
				open = true
			case sfnt.SegmentOpLineTo:
				out.WriteByte('L')
				writeMathPoint(&out, x, s.Args[0])
			case sfnt.SegmentOpQuadTo:
				out.WriteByte('Q')
				writeMathPoint(&out, x, s.Args[0])
				out.WriteByte(' ')
				writeMathPoint(&out, x, s.Args[1])
			case sfnt.SegmentOpCubeTo:
				out.WriteByte('C')
				writeMathPoint(&out, x, s.Args[0])
				out.WriteByte(' ')
				writeMathPoint(&out, x, s.Args[1])
				out.WriteByte(' ')
				writeMathPoint(&out, x, s.Args[2])
			}
		}
		if open {
			out.WriteByte('Z')
		}
		adv, err := f.GlyphAdvance(&buf, glyph, ppem, font.HintingNone)
		if err != nil {
			return "", fmt.Errorf("math: advance for %q: %w", r, err)
		}
		x += adv
		prevFont, prevGlyph = f, glyph
	}
	return out.String(), nil
}

func writeMathPoint(out *strings.Builder, dx fixed.Int26_6, p fixed.Point26_6) {
	out.WriteString(strconv.FormatFloat(float64(p.X+dx)/64, 'g', 8, 64))
	out.WriteByte(' ')
	out.WriteString(strconv.FormatFloat(float64(p.Y)/64, 'g', 8, 64))
}
