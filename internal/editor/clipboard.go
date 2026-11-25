package editor

import (
	"github.com/Maidang1/corra/internal/doc"
)

// clip returns the selection as clipboard content.
func (e *Editor) clip() (Clip, bool) {
	s := e.state
	var frag []*doc.Node
	switch sel := s.Sel.(type) {
	case doc.TextSelection:
		if sel.Empty() {
			return Clip{}, false
		}
		frag = s.Fragment(sel)
	case doc.NodeSelection:
		for _, id := range sel.IDs {
			if n := s.Node(id); n != nil {
				frag = append(frag, doc.CloneFresh(n))
			}
		}
	}
	if len(frag) == 0 {
		return Clip{}, false
	}
	c := Clip{Plain: doc.PlainText(frag), Fragment: frag}
	if e.cfg.Codec != nil {
		c.Markdown = e.cfg.Codec.Write(doc.NewBlock(e.cfg.Registry.Root(), nil, frag...))
	}
	return c, true
}

// Copy puts the selection on the clipboard.
func (e *Editor) Copy() {
	if c, ok := e.clip(); ok && e.cfg.Clipboard != nil {
		e.cfg.Clipboard.Write(c)
	}
}

// Cut copies the selection and deletes it.
func (e *Editor) Cut() {
	if c, ok := e.clip(); ok && e.cfg.Clipboard != nil {
		e.cfg.Clipboard.Write(c)
		e.Run(doc.DeleteSelection, "delete")
	}
}

// Paste inserts the clipboard: the editor's own content as it was, else
// HTML or Markdown read as Markdown, else plain text; in a plain-text
// block, as code, always as plain text.
func (e *Editor) Paste() {
	if e.cfg.Clipboard == nil {
		return
	}
	e.PasteClip(e.cfg.Clipboard.Read())
}

// PasteClip inserts c as Paste does.
func (e *Editor) PasteClip(c Clip) {
	e.endComposition()
	sel, ok := e.state.TextSel()
	if !ok {
		return
	}
	if e.state.Node(sel.Head.Leaf).Type().Content() == doc.ContentText {
		text := c.Plain
		if text == "" && len(c.Fragment) > 0 {
			text = doc.PlainText(c.Fragment)
		}
		e.Run(doc.InsertText(text), "paste")
		return
	}
	if len(c.Fragment) > 0 {
		e.Run(doc.InsertFragment(c.Fragment), "paste")
		return
	}
	if len(c.Image) > 0 && c.Plain == "" && c.HTML == "" && e.cfg.OnPasteImage != nil {
		md, err := e.cfg.OnPasteImage(c.Image)
		if err != nil {
			return
		}
		c = Clip{Markdown: md}
	}
	md := c.Markdown
	if md == "" && c.HTML != "" && e.cfg.HTMLToMarkdown != nil {
		if converted, err := e.cfg.HTMLToMarkdown(c.HTML); err == nil {
			md = converted
		}
	}
	if md == "" {
		md = c.Plain
	}
	if md == "" {
		return
	}
	if e.cfg.Codec == nil {
		e.Run(doc.InsertText(md), "paste")
		return
	}
	parsed := e.cfg.Codec.Parse(md)
	e.Run(doc.InsertFragment(parsed.Children()), "paste")
}
