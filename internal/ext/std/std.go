// Package std lists the built-in extensions.
package std

import (
	"github.com/Maidang1/corra/internal/ext"
	"github.com/Maidang1/corra/internal/ext/blockquote"
	"github.com/Maidang1/corra/internal/ext/callout"
	"github.com/Maidang1/corra/internal/ext/code"
	"github.com/Maidang1/corra/internal/ext/core"
	"github.com/Maidang1/corra/internal/ext/diagram"
	"github.com/Maidang1/corra/internal/ext/emoji"
	"github.com/Maidang1/corra/internal/ext/emphasis"
	"github.com/Maidang1/corra/internal/ext/footnote"
	"github.com/Maidang1/corra/internal/ext/frontmatter"
	"github.com/Maidang1/corra/internal/ext/heading"
	"github.com/Maidang1/corra/internal/ext/hr"
	"github.com/Maidang1/corra/internal/ext/image"
	"github.com/Maidang1/corra/internal/ext/list"
	"github.com/Maidang1/corra/internal/ext/marks"
	"github.com/Maidang1/corra/internal/ext/math"
	"github.com/Maidang1/corra/internal/ext/table"
	"github.com/Maidang1/corra/internal/ext/toc"
)

// Extensions returns the built-in extensions.
func Extensions() []ext.Extension {
	return []ext.Extension{
		core.Extension(),
		heading.Extension(),
		blockquote.Extension(),
		callout.Extension(),
		list.Extension(),
		code.Extension(),
		hr.Extension(),
		emphasis.Extension(),
		image.Extension(),
		marks.Extension(),
		math.Extension(),
		footnote.Extension(),
		frontmatter.Extension(),
		toc.Extension(),
		table.Extension(),
		diagram.Extension(),
		emoji.Extension(),
	}
}

// Kit builds the built-in extensions.
func Kit() *ext.Kit { return ext.MustBuild(Extensions()...) }
