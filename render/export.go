package render

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"codeberg.org/go-fonts/liberation/liberationsansregular"
	"github.com/gpdf-dev/gpdf"
	"github.com/gpdf-dev/gpdf/document"
	"github.com/gpdf-dev/gpdf/template"
)

// darkInk is the text color math renders with in exported PDFs.
var darkInk = color.RGBA{0x2b, 0x2f, 0x36, 0xff}

const (
	pdfFont   = "noto"
	pdfMono   = "noto"
	pdfBaseSz = 10.5
)

// liberationTTF returns the embedded Liberation Sans (Latin). Run
// internal/genfonts/genfonts.py for a CJK-capable PDF font (see README).
func liberationTTF() []byte { return liberationsansregular.TTF }

// ExportPDF renders a document to PDF. Diagram, chart and math blocks are
// rasterized and embedded as images.
func ExportPDF(ctx context.Context, doc *Document, title string) ([]byte, error) {
	return ExportPDFOpts(ctx, doc, title, nil)
}

// ExportPDFOpts is ExportPDF with an optional image loader: it receives the
// image URL (already resolved against the document directory by the caller)
// and returns encoded PNG/JPEG bytes. Without a loader, image blocks render
// their alt text.
func ExportPDFOpts(ctx context.Context, doc *Document, title string, loadImage func(url string) ([]byte, error)) ([]byte, error) {
	d := gpdf.NewDocument(
		gpdf.WithFont(pdfFont, liberationTTF()),
		gpdf.WithDefaultFont(pdfFont, pdfBaseSz),
		gpdf.WithPageSize(document.A4),
		gpdf.WithMargins(document.UniformEdges(document.Mm(18))),
		gpdf.WithMetadata(document.DocumentMetadata{Title: title, Producer: "Corra"}),
	)
	page := d.AddPage()
	for i, b := range doc.Blocks {
		// Typora's page break: <div style="page-break-after: always"></div>
		if b.Kind == KindHTML && strings.Contains(strings.ToLower(b.Source), "page-break-after") {
			page = d.AddPage()
			continue
		}
		if i > 0 {
			page.AutoRow(func(r *template.RowBuilder) {
				r.Col(12, func(c *template.ColBuilder) {
					c.Spacer(document.Mm(1.2))
				})
			})
		}
		if err := pdfBlock(ctx, page, &b, doc, loadImage); err != nil {
			// Degrade: render failures show the block source.
			pdfText(page, b.Source, template.FontSize(pdfBaseSz-1))
		}
	}
	if len(doc.Footnotes) > 0 {
		items := make([]string, len(doc.Footnotes))
		for i, fn := range doc.Footnotes {
			items[i] = fmt.Sprintf("[%d] %s", i+1, fn)
		}
		page.AutoRow(func(r *template.RowBuilder) {
			r.Col(12, func(c *template.ColBuilder) {
				c.Text("脚注", template.FontSize(pdfBaseSz), template.Bold())
				c.List(items)
			})
		})
	}
	return d.Generate()
}

func pdfText(page *template.PageBuilder, s string, opts ...template.TextOption) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(12, func(c *template.ColBuilder) {
			c.Text(s, opts...)
		})
	})
}

func pdfBlock(ctx context.Context, page *template.PageBuilder, b *Block, doc *Document, loadImage func(string) ([]byte, error)) error {
	switch b.Kind {
	case KindFrontMatter:
		lines := strings.Split(b.Source, "\n")
		if len(lines) >= 2 {
			lines = lines[1 : len(lines)-1]
		}
		pdfText(page, strings.Join(lines, "  ·  "), template.FontSize(pdfBaseSz-2))
	case KindTOC:
		var items []string
		for _, hb := range doc.Blocks {
			if hb.Kind != KindHeading {
				continue
			}
			items = append(items, strings.Repeat("　", hb.Level-1)+PlainText(hb.Source))
		}
		if len(items) > 0 {
			page.AutoRow(func(r *template.RowBuilder) {
				r.Col(12, func(c *template.ColBuilder) {
					c.Text("目录", template.FontSize(pdfBaseSz), template.Bold())
					c.List(items)
				})
			})
		}
	case KindHeading:
		size := map[int]float64{1: 21, 2: 17, 3: 14, 4: 12.5, 5: 11.5, 6: 11}[b.Level]
		if size == 0 {
			size = 11
		}
		family := pdfFont
		if b.Level <= 2 {
			family = pdfFont + "-bold"
		}
		pdfText(page, PlainText(b.Source), template.FontSize(size), template.FontFamily(family), template.Bold())
	case KindParagraph, KindQuote, KindHTML:
		t := pdfInlineText(b)
		opts := []template.TextOption{template.FontSize(pdfBaseSz)}
		if b.Kind == KindQuote {
			opts = append(opts, template.Italic())
		}
		if b.Kind == KindHTML {
			t = b.Source
			opts = append(opts, template.FontSize(pdfBaseSz-1.5))
		}
		pdfText(page, t, opts...)
	case KindCodeBlock:
		code := strings.TrimSuffix(b.Source, "\n")
		pdfText(page, code, template.FontSize(9.5), template.FontFamily(pdfMono))
	case KindList:
		items := make([]string, 0, len(b.Items))
		for _, it := range b.Items {
			t := PlainText(it.Source)
			if it.Task {
				box := "☐"
				if it.Done {
					box = "☑"
				}
				t = box + " " + t
			}
			items = append(items, t)
		}
		page.AutoRow(func(r *template.RowBuilder) {
			r.Col(12, func(c *template.ColBuilder) {
				if b.Ordered {
					c.OrderedList(items)
				} else {
					c.List(items)
				}
			})
		})
	case KindTable:
		header := make([]string, len(b.Header))
		for i, h := range b.Header {
			header[i] = h.Text
		}
		rows := make([][]string, 0, len(b.Rows))
		for _, row := range b.Rows {
			cells := make([]string, len(row))
			for i, cell := range row {
				cells[i] = cell.Text
			}
			rows = append(rows, cells)
		}
		page.AutoRow(func(r *template.RowBuilder) {
			r.Col(12, func(c *template.ColBuilder) {
				c.Table(header, rows)
			})
		})
	case KindMathBlock:
		img, err := RenderMath(b.Math, true, pdfBaseSz+2, darkInk)
		if err != nil {
			return err
		}
		return pdfImage(page, img)
	case KindDiagram:
		svg, err := RenderDiagram(ctx, b.Diagram, b.DiagramS)
		if err != nil {
			return err
		}
		img, err := RasterizeSVG(svg, 2)
		if err != nil {
			return err
		}
		return pdfImage(page, img)
	case KindChart:
		spec, err := ParseChart(b.ChartS)
		if err != nil {
			return err
		}
		img, err := RenderChartImage(spec, 2)
		if err != nil {
			return err
		}
		return pdfImage(page, img)
	case KindImage:
		if loadImage != nil {
			if data, err := loadImage(b.ImageURL); err == nil {
				img, _, err := image.Decode(bytes.NewReader(data))
				if err == nil {
					return pdfImageSized(page, img, b)
				}
			}
		}
		if b.ImageAlt != "" {
			pdfText(page, "[图片: "+b.ImageAlt+"]", template.FontSize(pdfBaseSz-1.5))
		}
		return nil
	case KindHR:
		page.AutoRow(func(r *template.RowBuilder) {
			r.Col(12, func(c *template.ColBuilder) {
				c.Line()
			})
		})
	}
	return nil
}

// pdfInlineText flattens a paragraph block's inline source to readable text.
func pdfInlineText(b *Block) string {
	ins := ParseInlines(strings.TrimSpace(b.Source))
	return inlinePlainAll(ins)
}

func pdfImage(page *template.PageBuilder, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(12, func(c *template.ColBuilder) {
			c.Image(buf.Bytes(), template.FitWidth(document.Mm(160)), template.WithAlign(document.AlignCenter))
		})
	})
	return nil
}

// pdfImageSized embeds an image honoring the =WxH / =NN% size hints.
func pdfImageSized(page *template.PageBuilder, img image.Image, b *Block) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	width := document.Mm(160) // fit column by default
	w := img.Bounds().Dx()
	switch {
	case b.ImageWidth > 0:
		width = document.Pt(float64(b.ImageWidth))
	case b.ImageZoom > 0 && w > 0:
		width = document.Pt(float64(w) * float64(b.ImageZoom) / 100 * 0.75) // px -> pt
	}
	page.AutoRow(func(r *template.RowBuilder) {
		r.Col(12, func(c *template.ColBuilder) {
			c.Image(buf.Bytes(), template.FitWidth(width), template.WithAlign(document.AlignCenter))
		})
	})
	return nil
}
