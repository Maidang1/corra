package views

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/Maidang1/corra/app"
	"github.com/Maidang1/corra/render"
)

// diagramTimeout bounds WASM and layout engine work.
const diagramTimeout = 20 * time.Second

// block renders one document block in preview mode, read-only. Rich text
// blocks render their inline formatting; structured blocks (code, tables,
// math, diagrams, charts) render as cards and open the source editor on
// click, the way Apple Notes' previews stay out of the way of reading.
func (v *View) block(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()

	switch b.Kind {
	case render.KindHeading:
		v.headingBlock(c, i)
	case render.KindParagraph:
		v.inlineFlow(c, b.Source, t, t.FontSize)
	case render.KindFrontMatter:
		v.frontMatterBlock(c, b)
	case render.KindTOC:
		v.tocBlock(c)
	case render.KindCodeBlock:
		v.codeBlock(c, i)
	case render.KindQuote:
		v.quoteBlock(c, i)
	case render.KindList:
		v.listBlock(c, i)
	case render.KindTable:
		v.tableBlock(c, i)
	case render.KindMathBlock:
		v.mathBlock(c, i)
	case render.KindDiagram:
		v.diagramBlock(c, i)
	case render.KindChart:
		v.chartBlock(c, i)
	case render.KindImage:
		v.imageBlock(c, i)
	case render.KindHR:
		ui.Divider(c).FillWidth().PaddingY(6)
	case render.KindHTML:
		ui.Box(c).FillWidth().Padding(10, 12).Background(t.Surface).
			Border(1, t.Border).Children(func() {
			ui.Text(c, strings.TrimSpace(b.Source)).Font(fontMono).FontSize(11.5).
				TextColor(t.TextMuted)
		})
	default:
		ui.Text(c, b.Source)
	}
}

// headingSize mirrors the native editor's type scale (27/22/18.5/16.5/15.5/15
// on a 16pt body): the first line reads as a note title, not a banner.
func headingSize(level int, base float32) float32 {
	sz := map[int]float32{1: 23.5, 2: 19.25, 3: 16.25, 4: 14.5, 5: 13.5, 6: 13}[level]
	if sz == 0 {
		sz = 13
	}
	return sz * base / 14
}

func (v *View) headingBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	size := headingSize(b.Level, t.FontSize)
	// Strip the leading #'s so they don't show in the preview.
	src := strings.TrimLeft(b.Source, "#")
	if v.App.HeadingNumbering {
		if num := headingNumber(v.App.Doc, i); num != "" {
			src = " " + num + src // src keeps its leading space after #'s
		}
	}
	el := v.inlineFlow(c, src, t, size)
	if b.Level <= 2 {
		el = el.FontWeight(700)
	} else {
		el = el.FontWeight(600)
	}
	if i == v.scrollToBlock {
		el = el.ScrollIntoView()
		v.scrollToBlock = -1
	}
	// Headings belong to what follows: open room above, stay close below.
	if b.Level == 1 {
		el = el.Padding(8, 0, 0, 0)
	} else {
		el = el.Padding(5, 0, 0, 0)
	}
}

func (v *View) quoteBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	if b.Alert != "" {
		v.alertBlock(c, b, t)
		return
	}
	// A rounded bar at the text edge, as the native editor draws quotes.
	ui.Row(c).FillWidth().Gap(10).Padding(2, 0).Children(func() {
		ui.Box(c).Width(3).FillHeight().Background(t.Scrollbar).Radius(1.5)
		ui.Column(c).FillWidth().Gap(6).Padding(4, 0).Children(func() {
			for j := range b.Items {
				v.inlineFlow(c, b.Items[j].Source, t, t.FontSize)
			}
		})
	})
}

// alertBlock draws a GitHub-style callout: a tinted card, an accent bar and the
// label as its title. The marker itself was stripped by the parser.
func (v *View) alertBlock(c *ui.Context, b *render.Block, t *ui.Theme) {
	accent, tint := alertColors(b.Alert, t.Dark)
	ui.Row(c).FillWidth().Gap(10).Padding(4, 0).Children(func() {
		ui.Box(c).Width(3).FillHeight().Background(accent).Radius(1.5)
		ui.Column(c).FillWidth().Gap(4).Padding(8, 12).Radius(6).Background(tint).Children(func() {
			ui.Text(c, render.AlertTitle(b.Alert)).FontSize(t.FontSize - 2.5).
				TextColor(accent).Bold()
			for j := range b.Items {
				v.inlineFlow(c, b.Items[j].Source, t, t.FontSize)
			}
		})
	})
}

// alertColors picks the accent and tint of a callout kind, matching the CSS the
// HTML export writes.
func alertColors(label string, dark bool) (accent, tint ui.Color) {
	type pair struct{ accent, light, dark string }
	palette := map[string]pair{
		"NOTE":      {"#54aeff", "#ddf4ff", "#10233a"},
		"TIP":       {"#4ac26b", "#dafbe1", "#122b1c"},
		"IMPORTANT": {"#b062f5", "#fbefff", "#241a38"},
		"WARNING":   {"#d4a72c", "#fff8c5", "#2e2612"},
		"CAUTION":   {"#ff8182", "#ffebe9", "#33191a"},
	}
	p, ok := palette[label]
	if !ok {
		p = palette["NOTE"]
	}
	accent = ui.Hex(p.accent)
	tint = ui.Hex(p.light)
	if dark {
		tint = ui.Hex(p.dark)
	}
	return accent, tint
}

func (v *View) listBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(1).Children(func() {
		for j := range b.Items {
			item := &b.Items[j]
			idx, depth := j, item.Depth
			ui.Row(c).FillWidth().Gap(10).Center().
				Padding(0, 0, 0, float32(depth)*18).Children(func() {
				if item.Task {
					done := item.Done
					ui.Checkbox(c, &done, "").Label("任务：" + item.Source).Disabled(!v.canEdit()).OnChange(func() {
						v.toggleTask(i, idx)
					})
				} else if b.Ordered {
					ui.Text(c, fmt.Sprintf("%d.", idx+1)).TextColor(t.TextMuted).
						FontSize(t.FontSize - 2).NoWrap()
				} else {
					ui.Text(c, "•").TextColor(t.TextMuted).
						FontSize(t.FontSize + 2).NoWrap()
				}
				v.inlineFlow(c, item.Source, t, t.FontSize).LineHeight(1.35)
			})
		}
	})
}

// toggleTask flips one task list item's checkbox, keeping the rest of the
// block's items as they are.
func (v *View) toggleTask(blockIdx, itemIdx int) {
	if !v.canEdit() || blockIdx < 0 || blockIdx >= len(v.App.Doc.Blocks) {
		return
	}
	b := &v.App.Doc.Blocks[blockIdx]
	if itemIdx < 0 || itemIdx >= len(b.Items) {
		return
	}
	item := b.Items[itemIdx]
	src := v.App.Source
	offset := item.CheckboxOffset
	if !item.Task || offset <= 0 || offset+1 >= len(src) || src[offset-1] != '[' || src[offset+1] != ']' {
		return
	}
	marker := "x"
	if item.Done {
		marker = " "
	}
	v.App.SetSource(src[:offset]+marker+src[offset+1:], offset)
}

func (v *View) codeBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	code := strings.TrimSuffix(b.Source, "\n")
	colors := codeColors(t)
	ui.Box(c).FillWidth().Children(func() {
		ui.Box(c).FillWidth().Background(t.Surface).Border(1, t.Border).
			Padding(12, 14).Children(func() {
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				if b.Language != "" {
					ui.Text(c, b.Language).FontSize(11).TextColor(t.TextMuted)
				}
				v.richLines(c, code, b.Language, colors, t.FontSize-1, v.App.CodeLineNumbers)
			})
		})
	}).OnClick(func() {
		v.openBlockEditor(i, code, false)
	})
}

// richLines renders code as RichText spans with highlight colors; lineNumbers
// adds a gutter of line numbers (Typora's code-fence line-number option).
func (v *View) richLines(c *ui.Context, code, language string, colors map[string]ui.Color, size float32, lineNumbers bool) {
	spans := render.Highlight(code, language)
	uiSpans := make([]ui.Span, 0, len(spans))
	for _, s := range spans {
		uiSpans = append(uiSpans, ui.Span{
			Text:  s.Text,
			Color: colors[s.Class],
			Font:  fontMono,
			Size:  size,
		})
	}
	if len(uiSpans) == 0 {
		uiSpans = []ui.Span{{Text: code, Font: fontMono, Size: size}}
	}
	if lineNumbers {
		t := c.Theme()
		lines := strings.Count(code, "\n") + 1
		var nums strings.Builder
		for n := 1; n <= lines; n++ {
			fmt.Fprintf(&nums, "%d\n", n)
		}
		ui.Row(c).FillWidth().Gap(10).Children(func() {
			ui.Text(c, strings.TrimSuffix(nums.String(), "\n")).Font(fontMono).
				FontSize(size).TextColor(t.TextMuted).LineHeight(1.5).NoWrap()
			ui.RichText(c, uiSpans...).FillWidth().LineHeight(1.5)
		})
		return
	}
	ui.RichText(c, uiSpans...).FillWidth().LineHeight(1.5)
}

func (v *View) tableBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	ui.Box(c).FillWidth().Border(1, t.Border).Padding(2, 2).Children(func() {
		ui.Column(c).FillWidth().Gap(0).Children(func() {
			renderRow := func(cells []render.TableCell, header bool) {
				ui.Row(c).FillWidth().Gap(1).Dividers(1, t.Border).Children(func() {
					for _, cell := range cells {
						ui.Box(c).Grow(1).Padding(6, 10).Background(func() ui.Color {
							if header {
								return t.SurfaceHover
							}
							return t.Surface
						}()).Children(func() {
							txt := cell.Text
							el := ui.Text(c, txt).FontSize(t.FontSize - 1)
							if header {
								el.Bold()
							}
							_ = el
						})
					}
				})
				ui.Divider(c).FillWidth()
			}
			renderRow(b.Header, true)
			for _, row := range b.Rows {
				renderRow(row, false)
			}
		})
	}).OnClick(func() {
		v.openBlockEditor(i, strings.TrimRight(b.Source, "\n"), false)
	})
}

func (v *View) mathBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	fg := color.RGBA{R: t.Text.R, G: t.Text.G, B: t.Text.B, A: t.Text.A}
	results, ready, batchErr := v.App.MathImages(v.App.Doc.MathExpressions, render.MathOptions{Numbering: v.App.MathNumbering}, 15*float64(v.App.Zoom), fg)
	index, exists := v.App.Doc.MathIndex(i)
	var img image.Image
	var mathErr error
	if batchErr != nil {
		mathErr = batchErr
	} else if ready && exists && index < len(results) {
		img, mathErr = results[index].Image, results[index].Error
	}
	ui.Box(c).FillWidth().Padding(10, 12).Center().Children(func() {
		if img != nil {
			ui.Image(c, ui.NewBitmap(img)).Size(float32(img.Bounds().Dx())/2, float32(img.Bounds().Dy())/2)
		} else {
			element := ui.Text(c, b.Math).FontSize(t.FontSize).TextColor(t.TextMuted)
			if mathErr != nil {
				element.Tooltip(mathErr.Error())
			}
		}
	}).OnClick(func() {
		v.openBlockEditor(i, "$$\n"+b.Math+"\n$$", false)
	})
}

func (v *View) diagramBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	kindName := map[render.DiagramKind]string{render.DiagramMermaid: "mermaid", render.DiagramD2: "d2", render.DiagramGoAT: "goat", render.DiagramFlowchartJS: "flow", render.DiagramSequenceJS: "sequence"}[b.Diagram]
	key := app.ImageKey("diag-"+kindName, b, v.App.Zoom, t.Dark)
	img, ok := v.App.BlockImage(key, func() (image.Image, error) {
		ctx, cancel := context.WithTimeout(context.Background(), diagramTimeout)
		defer cancel()
		svg, err := render.RenderDiagram(ctx, b.Diagram, b.DiagramS)
		if err != nil {
			return nil, err
		}
		return render.RasterizeSVG(svg, 2)
	})
	ui.Box(c).FillWidth().Padding(12, 14).Background(t.Surface).Border(1, t.Border).
		Children(func() {
			if ok && img != nil {
				ui.Image(c, ui.NewBitmap(img))
				return
			}
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				ui.Text(c, kindName+" 图表（渲染中…）").FontSize(11).TextColor(t.TextMuted)
			})
		}).OnClick(func() {
		v.openBlockEditor(i, "```"+kindName+"\n"+strings.TrimRight(b.DiagramS, "\n")+"\n```", false)
	})
}

func (v *View) chartBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	key := app.ImageKey("chart", b, v.App.Zoom, t.Dark)
	img, ok := v.App.BlockImage(key, func() (image.Image, error) {
		spec, err := render.ParseChart(b.ChartS)
		if err != nil {
			return nil, err
		}
		return render.RenderChartImage(spec, 2)
	})
	ui.Box(c).FillWidth().Padding(12, 14).Background(t.Surface).Border(1, t.Border).
		Children(func() {
			if ok && img != nil {
				ui.Image(c, ui.NewBitmap(img))
				return
			}
			ui.Text(c, "chart 数据图表（渲染中…）").FontSize(11).TextColor(t.TextMuted)
		}).OnClick(func() {
		v.openBlockEditor(i, "```chart\n"+strings.TrimRight(b.ChartS, "\n")+"\n```", false)
	})
}

func (v *View) imageBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	baseDir := ""
	if v.App.Path != "" {
		baseDir = dirOf(v.App.Path)
	}
	e := v.Images.Get(b.ImageURL, baseDir, v.invalidate)
	ui.Box(c).FillWidth().Padding(8, 0).Children(func() {
		if bmp, ok := e.Result(); ok && bmp != nil {
			img := ui.Image(c, bmp).MaxWidth(680)
			// Typora-style sizing: =WxH / =NN% suffix or <img width height>.
			w, h := bmp.Size()
			switch {
			case b.ImageWidth > 0 && b.ImageHeight > 0:
				img = img.Width(float32(b.ImageWidth) * v.App.Zoom).Height(float32(b.ImageHeight) * v.App.Zoom)
			case b.ImageWidth > 0:
				img = img.Width(float32(b.ImageWidth) * v.App.Zoom)
			case b.ImageHeight > 0:
				img = img.Height(float32(b.ImageHeight) * v.App.Zoom)
			case b.ImageZoom > 0 && w > 0:
				img = img.Width(float32(w) * float32(b.ImageZoom) / 100 * v.App.Zoom)
			}
			_ = h // h reserved for aspect-fit sizing
			if b.ImageAlt != "" {
				ui.Text(c, b.ImageAlt).FontSize(11.5).TextColor(t.TextMuted).Center()
			}
		} else if e.Err() != nil {
			ui.Text(c, "图片加载失败："+b.ImageURL).FontSize(11.5).TextColor(t.Danger)
		} else {
			ui.Text(c, "加载图片…").FontSize(11.5).TextColor(t.TextMuted)
		}
	})
}

func (v *View) footnotes(c *ui.Context) {
	if len(v.App.Doc.Footnotes) == 0 {
		return
	}
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(4).PaddingY(8).Children(func() {
		ui.Divider(c).FillWidth()
		for i, fn := range v.App.Doc.Footnotes {
			idx := i
			ui.RichText(c,
				ui.Span{Text: fmt.Sprintf("[%d] ", idx+1), Weight: 600},
				ui.Span{Text: fn, Color: t.TextMuted},
			).FontSize(12)
		}
	})
}

func dirOf(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return ""
}

// frontMatterBlock renders YAML front matter as a quiet metadata card.
func (v *View) frontMatterBlock(c *ui.Context, b *render.Block) {
	t := c.Theme()
	lines := strings.Split(b.Source, "\n")
	if len(lines) >= 2 {
		lines = lines[1 : len(lines)-1] // drop --- fences
	}
	ui.Box(c).FillWidth().Padding(4, 0).Children(func() {
		ui.Column(c).FillWidth().Gap(2).Padding(8, 12).Radius(8).
			Background(t.SurfaceHover).Children(func() {
			for _, line := range lines {
				ui.Text(c, line).Font(fontMono).FontSize(11).TextColor(t.TextMuted).FillWidth()
			}
		})
	})
}

// tocBlock renders [TOC] as the document's heading outline; entries scroll
// the preview to their heading.
func (v *View) tocBlock(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(3).Padding(4, 0).Children(func() {
		ui.Text(c, "目录").FontSize(11).TextColor(t.TextMuted)
		for i := range v.App.Doc.Blocks {
			hb := &v.App.Doc.Blocks[i]
			if hb.Kind != render.KindHeading {
				continue
			}
			idx := i
			ui.Text(c, hb.HeadingText()).
				FontSize(t.FontSize-1).TextColor(t.Accent).
				Padding(0, 0, 0, float32(hb.Level-1)*16).
				Cursor(ui.CursorPointer).
				OnClick(func() {
					v.scrollToBlock = idx
					v.App.InvalidateUI()
				})
		}
	})
}

// headingNumber computes the 1.2.3-style number of the i-th block when it is
// a heading, from the heading sequence of the document.
func headingNumber(doc *render.Document, i int) string {
	if i < 0 || i >= len(doc.Blocks) || doc.Blocks[i].Kind != render.KindHeading {
		return ""
	}
	var counters [6]int
	for k := 0; k <= i; k++ {
		hb := &doc.Blocks[k]
		if hb.Kind != render.KindHeading {
			continue
		}
		level := hb.Level
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		counters[level-1]++
		for m := level; m < 6; m++ {
			counters[m] = 0
		}
	}
	level := doc.Blocks[i].Level
	if level < 1 {
		level = 1
	}
	var parts []string
	for m := 0; m < level; m++ {
		if counters[m] == 0 {
			counters[m] = 1
		}
		parts = append(parts, fmt.Sprintf("%d", counters[m]))
	}
	return strings.Join(parts, ".") + "."
}
