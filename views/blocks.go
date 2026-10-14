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

// block renders one document block in single-pane mode.
func (v *View) block(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	key := i // stable identity across reparses as long as possible

	wrap := ui.Box(c).Key(key).FillWidth()
	if v.jumpTo == i {
		wrap.ScrollIntoView()
		v.jumpTo = -1
	}
	wrap.Children(func() {
		switch b.Kind {
		case render.KindHeading:
			v.headingBlock(c, i)
		case render.KindParagraph:
			v.paragraphBlock(c, i)
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
	})
}

func headingSize(level int, base float32) float32 {
	sz := map[int]float32{1: 26, 2: 21.5, 3: 18, 4: 16, 5: 14.5, 6: 13.5}[level]
	if sz == 0 {
		sz = 13.5
	}
	return sz * base / 13.5
}

func (v *View) headingBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	size := headingSize(b.Level, t.FontSize)
	el := ui.EditableText(c, &v.editBufs[i]).
		FontSize(size).LineHeight(1.3).FillWidth()
	if b.Level <= 2 {
		el = el.Bold()
	}
	if b.Level == 1 {
		el = el.PaddingY(6)
	}
	if el.Changed() {
		text := strings.TrimSpace(v.editBufs[i])
		if text == "" {
			v.App.ReplaceBlockRange(i, "")
			return
		}
		prefix := strings.Repeat("#", b.Level) + " "
		if !strings.HasPrefix(text, "#") {
			v.App.ReplaceBlockRange(i, prefix+text)
		} else {
			v.App.ReplaceBlockRange(i, text)
		}
	}
}

func (v *View) paragraphBlock(c *ui.Context, i int) {
	t := c.Theme()
	el := ui.EditableText(c, &v.editBufs[i]).
		FontSize(t.FontSize).LineHeight(1.65).FillWidth()
	if el.Changed() {
		v.commitTextBlock(i)
	}
}

// commitTextBlock writes an edited paragraph/quote source back.
func (v *View) commitTextBlock(i int) {
	text := v.editBufs[i]
	if strings.TrimSpace(text) == "" {
		v.App.ReplaceBlockRange(i, "")
		return
	}
	// Trigger conversions: typing "# ", "- ", "> " at the start changes
	// the block kind naturally on re-parse.
	v.App.ReplaceBlockRange(i, text)
}

func (v *View) quoteBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	ui.Box(c).FillWidth().Padding(2, 0, 2, 14).Children(func() {
		ui.Box(c).FillWidth().Padding(10, 14).Background(t.Surface).
			Border(1, t.Border).Children(func() {
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				for j := range b.Items {
					item := &b.Items[j]
					el := ui.EditableText(c, &item.Source).
						FontSize(t.FontSize).LineHeight(1.65).Italic().FillWidth()
					if el.Changed() {
						v.commitQuoteItem(i, j)
					}
				}
			})
		})
	})
}

func (v *View) commitQuoteItem(blockIdx, itemIdx int) {
	b := &v.App.Doc.Blocks[blockIdx]
	lines := make([]string, len(b.Items))
	for j := range b.Items {
		src := strings.TrimSpace(b.Items[j].Source)
		if !strings.HasPrefix(src, ">") {
			src = "> " + src
		}
		lines[j] = src
	}
	v.App.ReplaceBlockRange(blockIdx, strings.Join(lines, "\n"))
}

func (v *View) listBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		for j := range b.Items {
			item := &b.Items[j]
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				marker := "•"
				if b.Ordered {
					marker = fmt.Sprintf("%d.", j+1)
				}
				if item.Task {
					done := item.Done
					ui.Checkbox(c, &done, "").OnChange(func() {
						v.toggleTask(i, j)
					})
				} else {
					ui.Text(c, marker).TextColor(t.TextMuted).FontSize(t.FontSize)
				}
				el := ui.EditableText(c, &item.Source).
					FontSize(t.FontSize).LineHeight(1.6).FillWidth()
				if el.Changed() {
					v.commitListItem(i)
					return
				}
			})
		}
	})
}

// commitListItem rebuilds the list block from its item bindings.
func (v *View) commitListItem(i int) {
	b := &v.App.Doc.Blocks[i]
	marker := "- "
	if b.Ordered {
		marker = "1. "
	}
	lines := make([]string, 0, len(b.Items))
	for j := range b.Items {
		item := &b.Items[j]
		src := strings.TrimSpace(item.Source)
		if src == "" {
			continue
		}
		line := marker + src
		if item.Task {
			box := "[ ]"
			if item.Done {
				box = "[x]"
			}
			line = "- " + box + " " + src
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		v.App.ReplaceBlockRange(i, "")
		return
	}
	v.App.ReplaceBlockRange(i, strings.Join(lines, "\n"))
}

func (v *View) toggleTask(blockIdx, itemIdx int) {
	b := &v.App.Doc.Blocks[blockIdx]
	marker := "- "
	if b.Ordered {
		marker = "1. "
	}
	lines := make([]string, 0, len(b.Items))
	for j := range b.Items {
		item := &b.Items[j]
		done := item.Done
		if j == itemIdx {
			done = !done
		}
		src := strings.TrimSpace(item.Source)
		if src == "" {
			continue
		}
		if item.Task {
			box := "[ ]"
			if done {
				box = "[x]"
			}
			lines = append(lines, "- "+box+" "+src)
		} else {
			lines = append(lines, marker+src)
		}
	}
	if len(lines) == 0 {
		v.App.ReplaceBlockRange(blockIdx, "")
		return
	}
	v.App.ReplaceBlockRange(blockIdx, strings.Join(lines, "\n"))
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
				v.richLines(c, code, b.Language, colors, t.FontSize-1)
				ui.Text(c, "双击编辑代码").FontSize(10.5).TextColor(t.TextMuted).OnClick(func() {
					v.openBlockEditor(i, code, false)
				})
			})
		})
	}).OnClick(func() {
		v.openBlockEditor(i, code, false)
	})
}

// richLines renders code as RichText spans with highlight colors.
func (v *View) richLines(c *ui.Context, code, language string, colors map[string]ui.Color, size float32) {
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
	ui.RichText(c, uiSpans...).FillWidth().LineHeight(1.5)
}

func (v *View) tableBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	ui.Box(c).FillWidth().Border(1, t.Border).Padding(2, 2).Children(func() {
		ui.Column(c).FillWidth().Gap(0).Children(func() {
			renderRow := func(cells []render.TableCell, header bool) {
				ui.Row(c).FillWidth().Gap(0).Children(func() {
					for _, cell := range cells {
						ui.Box(c).Grow(1).Padding(8, 10).Background(func() ui.Color {
							if header {
								return t.SurfacePressed
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
	key := app.ImageKey("math", b, v.App.Zoom, t.Dark)
	img, ok := v.App.BlockImage(key, func() (image.Image, error) {
		return render.RenderMath(b.Math, true, 15, fg)
	})
	ui.Box(c).FillWidth().Padding(10, 12).Center().Children(func() {
		if ok && img != nil {
			ui.Image(c, ui.NewBitmap(img))
		} else {
			ui.Text(c, b.Math).FontSize(t.FontSize).TextColor(t.TextMuted)
		}
	}).OnClick(func() {
		v.openBlockEditor(i, "$$\n"+b.Math+"\n$$", false)
	})
}

func (v *View) diagramBlock(c *ui.Context, i int) {
	b := &v.App.Doc.Blocks[i]
	t := c.Theme()
	kindName := map[render.DiagramKind]string{render.DiagramMermaid: "mermaid", render.DiagramD2: "d2", render.DiagramGoAT: "goat"}[b.Diagram]
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
			ui.Image(c, bmp).MaxWidth(680)
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

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func dirOf(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return ""
}
