package nativerich

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/color"
	"image/png"

	"github.com/Maidang1/corra/render"
	"github.com/Maidang1/corra/richtext"
)

type nativeMathSlot struct {
	render.MathExpression
	Visible bool
}

type nativeMathResult struct {
	Image  string `json:"image"`
	Number string `json:"number"`
	Error  string `json:"error"`
}

// Source being edited still contributes its equations to document numbering,
// but only visible math runs receive attachments. Code and unsupported source
// stay literal. Never recurse through the source fallback of a second parse.
func nativeMathPlan(doc richtext.Document) []nativeMathSlot {
	var slots []nativeMathSlot
	appendParagraph := func(p richtext.Paragraph, visible bool) {
		if p.Kind == "math" {
			slots = append(slots, nativeMathSlot{render.MathExpression{TeX: p.Raw, Display: true}, visible})
			return
		}
		appendRuns := func(runs []richtext.Run) {
			for _, run := range runs {
				if run.Source {
					for _, parsed := range richtext.ParseInline(run.Text, p.Kind == "table").Paragraphs {
						if parsed.Kind != "body" {
							continue
						}
						for _, expanded := range parsed.Runs {
							if expanded.Math != "" {
								slots = append(slots, nativeMathSlot{render.MathExpression{TeX: expanded.Math}, false})
							}
						}
					}
					continue
				}
				if run.Math != "" {
					slots = append(slots, nativeMathSlot{render.MathExpression{TeX: run.Math}, visible})
				}
			}
		}
		if p.Kind == "table" {
			for _, row := range p.Rows {
				for _, cell := range row {
					appendRuns(cell.Runs)
				}
			}
		} else {
			appendRuns(p.Runs)
		}
	}
	for i := 0; i < len(doc.Paragraphs); {
		p := doc.Paragraphs[i]
		if p.Kind != "source" {
			appendParagraph(p, true)
			i++
			continue
		}
		end := i + 1
		for end < len(doc.Paragraphs) && doc.Paragraphs[end].Kind == "source" {
			end++
		}
		source := richtext.Serialize(richtext.Document{Paragraphs: doc.Paragraphs[i:end]})
		for _, parsed := range richtext.Parse(source).Paragraphs {
			appendParagraph(parsed, false)
		}
		i = end
	}
	return slots
}

func (e *Editor) documentMath(doc richtext.Document, dark bool, zoom float64, numbering string) string {
	return e.documentMathContext(doc, richtext.Document{}, richtext.Document{}, dark, zoom, numbering)
}

func (e *Editor) documentMathContext(doc, before, after richtext.Document, dark bool, zoom float64, numbering string) string {
	slots := nativeMathPlan(before)
	for i := range slots {
		slots[i].Visible = false
	}
	slots = append(slots, nativeMathPlan(doc)...)
	for _, slot := range nativeMathPlan(after) {
		slot.Visible = false
		slots = append(slots, slot)
	}
	key, _ := json.Marshal(struct {
		Slots     []nativeMathSlot
		Dark      bool
		Zoom      float64
		Numbering string
	}{slots, dark, zoom, numbering})
	if string(key) == e.mathKey {
		return e.mathResult
	}
	expressions := make([]render.MathExpression, len(slots))
	for i, slot := range slots {
		expressions[i] = slot.MathExpression
	}
	var results []render.MathResult
	var err error
	if len(expressions) > 0 {
		fg := color.RGBA{R: 41, G: 41, B: 41, A: 255}
		if dark {
			fg = color.RGBA{R: 231, G: 233, B: 238, A: 255}
		}
		results, err = render.RenderMathBatch(expressions, render.MathOptions{Numbering: numbering}, 15*zoom, fg)
	}
	visible := make([]nativeMathResult, 0, len(slots))
	for i, slot := range slots {
		if !slot.Visible {
			continue
		}
		result := nativeMathResult{}
		switch {
		case err != nil:
			result.Error = err.Error()
		case results[i].Error != nil:
			result.Error = results[i].Error.Error()
		default:
			result.Number = results[i].Number
			var buf bytes.Buffer
			if encodeErr := png.Encode(&buf, results[i].Image); encodeErr != nil {
				result.Error = encodeErr.Error()
			} else {
				result.Image = base64.StdEncoding.EncodeToString(buf.Bytes())
			}
		}
		visible = append(visible, result)
	}
	data, _ := json.Marshal(visible)
	e.mathKey, e.mathResult = string(key), string(data)
	return e.mathResult
}
