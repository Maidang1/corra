package views

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// documentStats counts the structural elements a status card reports. It is a
// light scan, not a parse: the counts only drive a glanceable strip.
type documentStats struct {
	Headings int
	Links    int
	Images   int
	Tasks    int
	Code     int
}

// inspectDocument counts headings, links, images, tasks and fenced code
// blocks outside fences, so a code sample's "#" is not a heading.
func inspectDocument(src string) documentStats {
	var stats documentStats
	inFence := false
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if !inFence {
				stats.Code++
			}
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if headingLine(trimmed) {
			stats.Headings++
		}
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "- [ ] ") || strings.HasPrefix(lower, "- [x] ") || strings.HasPrefix(lower, "* [ ] ") {
			stats.Tasks++
		}
		images := strings.Count(line, "![")
		stats.Images += images
		if links := strings.Count(line, "](") - images; links > 0 {
			stats.Links += links
		}
	}
	return stats
}

// headingLine reports a Markdown ATX heading: one to six # then a space.
func headingLine(line string) bool {
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	return level >= 1 && level <= 6 && level < len(line) && (line[level] == ' ' || line[level] == '\t')
}

// statsStrip is the expanded row above the count: one chip per measure, with
// empty structural counts left out.
func (v *View) statsStrip(c *ui.Context, wc wordCount) {
	t := c.Theme()
	stats := inspectDocument(v.App.CurrentSource())
	chip := func(label, value string) {
		ui.Row(c).Gap(5).AlignItems(ui.Center).Padding(3, 8).Radius(7).Background(fieldFill(t)).Children(func() {
			ui.Text(c, label).FontSize(10.5).TextColor(t.TextMuted).NoWrap()
			ui.Text(c, value).FontSize(11.5).Bold().TextColor(t.Text).NoWrap()
		})
	}
	ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Padding(2, 22, 0, 22).Children(func() {
		ui.Spacer(c)
		chip("字数", fmt.Sprintf("%d", wc.Words))
		chip("字符", fmt.Sprintf("%d", wc.Chars))
		chip("行", fmt.Sprintf("%d", wc.Lines))
		if stats.Headings > 0 {
			chip("标题", fmt.Sprintf("%d", stats.Headings))
		}
		if stats.Links > 0 {
			chip("链接", fmt.Sprintf("%d", stats.Links))
		}
		if stats.Images > 0 {
			chip("图片", fmt.Sprintf("%d", stats.Images))
		}
		if stats.Tasks > 0 {
			chip("任务", fmt.Sprintf("%d", stats.Tasks))
		}
		if stats.Code > 0 {
			chip("代码块", fmt.Sprintf("%d", stats.Code))
		}
		chip("阅读", fmt.Sprintf("%d 分钟", wc.Minutes))
	})
}
