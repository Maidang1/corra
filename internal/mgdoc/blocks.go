package mgdoc

import (
	"context"
	"image"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"

	"github.com/Maidang1/corra/render"
)

const blockWidth = 640

var blockLabel = map[blockKind]string{
	blockTable: "表格", blockMath: "公式", blockDiagram: "图表", blockAlert: "提示块",
	blockFootnote: "脚注", blockHTML: "HTML", blockRule: "分隔线", blockImage: "图片", blockOther: "源码块",
}

const editHint = "在源码模式中编辑"

// blockParagraph makes the one-character paragraph that holds a protected
// block as an attachment.
func (c *Codec) blockParagraph(kind blockKind, info, seg string) richtext.Paragraph {
	at := &richtext.Attachment{Name: blockLabel[kind], Data: rawRef{raw: seg, block: true}, Width: blockWidth}
	lines := strings.Split(seg, "\n")
	switch kind {
	case blockTable:
		rows := parseTable(lines)
		at.Height = float32(len(rows))*28 + 8
		at.Draw = func(p *ui.Painter, r ui.Rect) { drawTable(p, r, rows) }
	case blockMath:
		it := &mathItem{tex: strings.Trim(strings.TrimSpace(seg), "$`"), display: true, at: at}
		if strings.HasPrefix(strings.TrimSpace(seg), "```") || strings.HasPrefix(strings.TrimSpace(seg), "~~~") {
			it.tex = strings.Join(trimFence(lines), "\n")
		}
		c.math = append(c.math, it)
		at.Height = 56
		at.Draw = func(p *ui.Painter, r ui.Rect) { c.drawMathBlock(p, r, it, seg) }
	case blockDiagram:
		d := startDiagram(info, strings.Join(trimFence(lines), "\n"), c.Invalidate)
		at.Height = 280
		at.Draw = func(p *ui.Painter, r ui.Rect) { d.draw(p, r, seg) }
	case blockRule:
		at.Height = 20
		at.Draw = func(p *ui.Painter, r ui.Rect) {
			p.Line(r.X, r.Y+r.H/2, r.X+r.W, r.Y+r.H/2, 1, p.Theme().Border)
		}
	case blockAlert:
		label, body := alertParts(lines)
		n := 0
		for _, l := range body {
			n += len([]rune(l))/60 + 1
		}
		at.Height = float32(n)*22 + 44
		at.Draw = func(p *ui.Painter, r ui.Rect) { drawAlert(p, r, label, body) }
	case blockImage:
		return c.imageBlock(at, seg)
	default:
		n := min(len(lines), 8)
		at.Height = float32(n)*18 + 40
		label := blockLabel[kind]
		at.Draw = func(p *ui.Painter, r ui.Rect) { drawSource(p, r, label, lines) }
	}
	return richtext.Paragraph{Runs: []richtext.Run{{
		Text:       string(richtext.ObjectReplacement),
		Attributes: richtext.Attributes{Attachment: at},
	}}}
}

var imageParts = regexp.MustCompile(`^ {0,3}!\[([^\]]*)\]\(([^)]*)\)`)

// imageBlock shows a picture on its own line; unreadable ones get a chip.
func (c *Codec) imageBlock(at *richtext.Attachment, seg string) richtext.Paragraph {
	m := imageParts.FindStringSubmatch(seg)
	label, url := "", ""
	if m != nil {
		label, url = m[1], strings.Fields(m[2] + " ")[0]
	}
	at.Height = 28
	at.Draw = func(p *ui.Painter, r ui.Rect) { drawChip(p, r, "图片", firstNonEmpty(label, url)) }
	if c.LoadImage != nil && url != "" && !strings.Contains(url, "://") {
		if data, err := c.LoadImage(url, c.BaseDir); err == nil {
			if bmp, err := ui.DecodeBitmap(data); err == nil {
				w, h := bmp.Size()
				fw, fh := float32(w), float32(h)
				if fw > blockWidth {
					fh, fw = fh*blockWidth/fw, blockWidth
				}
				at.Image, at.Width, at.Height, at.Draw = bmp, fw, fh, nil
			}
		}
	}
	return richtext.Paragraph{Runs: []richtext.Run{{
		Text:       string(richtext.ObjectReplacement),
		Attributes: richtext.Attributes{Attachment: at},
	}}}
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

func trimFence(lines []string) []string {
	if len(lines) > 0 {
		lines = lines[1:]
	}
	if n := len(lines); n > 0 && closesFence(lines[n-1], "```") {
		lines = lines[:n-1]
	}
	return lines
}

func (c *Codec) drawMathBlock(p *ui.Painter, r ui.Rect, it *mathItem, raw string) {
	c.mu.Lock()
	bmp := it.bmp
	c.mu.Unlock()
	if bmp != nil {
		w, h := bmp.Size()
		fw, fh := float32(w), float32(h)
		if fw > r.W-16 {
			fh, fw = fh*(r.W-16)/fw, r.W-16
		}
		p.Image(bmp, ui.Rect{X: r.X + (r.W-fw)/2, Y: r.Y + (r.H-fh)/2, W: fw, H: fh}, ui.Contain)
		return
	}
	drawSource(p, r, "公式", strings.Split(raw, "\n"))
}

func alertParts(lines []string) (string, []string) {
	m := alertMark.FindStringSubmatch(lines[0])
	label := strings.ToUpper(m[1])
	var body []string
	for i, l := range lines {
		l = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), ">"))
		if i == 0 {
			l = strings.TrimSpace(alertMark.ReplaceAllString(lines[0], ""))
		}
		if l != "" {
			body = append(body, l)
		}
	}
	return label, body
}

func alertColor(label string, dark bool) ui.Color {
	switch label {
	case "TIP":
		return ui.Hex("#2da44e")
	case "IMPORTANT":
		return ui.Hex("#8250df")
	case "WARNING":
		return ui.Hex("#bf8700")
	case "CAUTION":
		return ui.Hex("#cf222e")
	}
	return ui.Hex("#0969da")
}

func drawAlert(p *ui.Painter, r ui.Rect, label string, body []string) {
	t := p.Theme()
	col := alertColor(label, t.Dark)
	p.Fill(r, col.Alpha(0.08), 6)
	p.Fill(ui.Rect{X: r.X, Y: r.Y, W: 3, H: r.H}, col, 1.5)
	p.Text(r.X+14, r.Y+8, label, 12, col)
	p.Clip(r, 0, func() {
		p.RichText(r.X+14, r.Y+28, r.W-28, ui.Span{Text: strings.Join(body, "\n"), Size: 14, Color: t.Text})
	})
}

func drawChip(p *ui.Painter, r ui.Rect, kind, text string) {
	t := p.Theme()
	p.Fill(r, t.Surface, 6)
	p.Stroke(r, t.Border, 6, 1)
	p.Text(r.X+8, r.Y+4, kind+" · "+text, 12, t.TextMuted)
}

func drawSource(p *ui.Painter, r ui.Rect, label string, lines []string) {
	t := p.Theme()
	p.Fill(r, t.Surface, 6)
	p.Stroke(r, t.Border, 6, 1)
	p.Text(r.X+10, r.Y+6, label+" · "+editHint, 11, t.TextMuted)
	p.Clip(r, 6, func() {
		for i, l := range lines {
			if i >= 8 {
				break
			}
			p.RichText(r.X+10, r.Y+26+float32(i)*18, r.W-20, ui.Span{Text: l, Font: "monospace", Size: 12, Color: t.Text})
		}
	})
}

type tableRows [][]string

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	var cells []string
	var cur strings.Builder
	for i := 0; i < len(line); i++ {
		if line[i] == '\\' && i+1 < len(line) && line[i+1] == '|' {
			cur.WriteByte('|')
			i++
			continue
		}
		if line[i] == '|' {
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(line[i])
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

var cellMarks = strings.NewReplacer("**", "", "__", "", "`", "", "~~", "")

func parseTable(lines []string) tableRows {
	var rows tableRows
	for i, l := range lines {
		if i == 1 {
			continue
		}
		cells := splitRow(l)
		for j := range cells {
			cells[j] = cellMarks.Replace(cells[j])
		}
		rows = append(rows, cells)
	}
	return rows
}

func drawTable(p *ui.Painter, r ui.Rect, rows tableRows) {
	t := p.Theme()
	cols := 0
	for _, row := range rows {
		cols = max(cols, len(row))
	}
	if cols == 0 {
		return
	}
	widths := make([]float32, cols)
	var total float32
	for j := range widths {
		for _, row := range rows {
			if j < len(row) {
				w, _ := p.MeasureText(0, ui.Span{Text: row[j], Size: 14})
				widths[j] = max(widths[j], w+20)
			}
		}
		widths[j] = max(widths[j], 48)
		total += widths[j]
	}
	if total > r.W {
		for j := range widths {
			widths[j] *= r.W / total
		}
		total = r.W
	}
	grid := ui.Rect{X: r.X, Y: r.Y + 2, W: total, H: float32(len(rows)) * 28}
	p.Stroke(grid, t.Border, 4, 1)
	p.Clip(grid, 4, func() {
		for i, row := range rows {
			y := grid.Y + float32(i)*28
			if i == 0 {
				p.Fill(ui.Rect{X: grid.X, Y: y, W: grid.W, H: 28}, t.Surface, 0)
			}
			if i > 0 {
				p.Line(grid.X, y, grid.X+grid.W, y, 1, t.Border)
			}
			x := grid.X
			for j, cell := range row {
				if j > 0 {
					p.Line(x, y, x, y+28, 1, t.Border)
				}
				col := t.Text
				size := float32(14)
				if i == 0 {
					size = 14
				}
				p.Clip(ui.Rect{X: x, Y: y, W: widths[j], H: 28}, 0, func() {
					p.Text(x+10, y+6, cell, size, col)
				})
				x += widths[j]
			}
		}
	})
}

// diagrams are rendered once per source in the background and shared.
type diagram struct {
	mu   sync.Mutex
	done bool
	err  error
	img  image.Image
	bmp  *ui.Bitmap
}

var (
	diagramMu    sync.Mutex
	diagramCache = map[string]*diagram{}
)

func startDiagram(lang, src string, invalidate func()) *diagram {
	key := lang + "\x00" + src
	diagramMu.Lock()
	if d, ok := diagramCache[key]; ok {
		diagramMu.Unlock()
		return d
	}
	d := &diagram{}
	diagramCache[key] = d
	diagramMu.Unlock()
	kinds := map[string]render.DiagramKind{
		"mermaid": render.DiagramMermaid, "d2": render.DiagramD2, "goat": render.DiagramGoAT,
		"flow": render.DiagramFlowchartJS, "sequence": render.DiagramSequenceJS,
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		svg, err := render.RenderDiagram(ctx, kinds[lang], src)
		var img image.Image
		if err == nil {
			img, err = render.RasterizeSVG(svg, 2)
		}
		d.mu.Lock()
		d.img, d.err, d.done = img, err, true
		d.mu.Unlock()
		if invalidate != nil {
			invalidate()
		}
	}()
	return d
}

func (d *diagram) draw(p *ui.Painter, r ui.Rect, raw string) {
	d.mu.Lock()
	done, err := d.done, d.err
	if done && err == nil && d.bmp == nil && d.img != nil {
		d.bmp = ui.NewBitmap(d.img)
	}
	bmp := d.bmp
	d.mu.Unlock()
	t := p.Theme()
	switch {
	case bmp != nil:
		p.Fill(r, ui.Hex("#ffffff"), 6)
		p.Stroke(r, t.Border, 6, 1)
		p.Image(bmp, ui.Rect{X: r.X + 8, Y: r.Y + 8, W: r.W - 16, H: r.H - 16}, ui.Contain)
	case done:
		drawSource(p, r, "图表（渲染失败）", strings.Split(raw, "\n"))
	default:
		drawChip(p, r, "图表", "渲染中…")
	}
}
