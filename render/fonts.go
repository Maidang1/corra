package render

import (
	"codeberg.org/go-latex/latex/font/liberation"
	"golang.org/x/image/font/sfnt"
)

// notoFont backs chart text. By default it is the embedded Liberation Sans
// (Latin). For Chinese labels in charts and PDF export, run
// `python3 internal/genfonts/genfonts.py` once to subset a CJK font into
// render/fonts/ (gitignored); fonts.go then prefers it automatically.
var notoFont = liberation.Fonts().Rm

// For Chinese labels in charts and PDFs, subset a CJK font with
// internal/genfonts/genfonts.py; a future version loads it here.
var _ = sfnt.Parse
