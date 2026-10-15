package richtext

import (
	"strings"

	east "github.com/yuin/goldmark/extension/ast"
)

func convertTable(table *east.Table, source []byte, raw string) []Paragraph {
	paragraph := Paragraph{Kind: "table", Runs: []Run{}}
	for _, alignment := range table.Alignments {
		name := "none"
		switch alignment {
		case east.AlignLeft:
			name = "left"
		case east.AlignRight:
			name = "right"
		case east.AlignCenter:
			name = "center"
		}
		paragraph.Alignments = append(paragraph.Alignments, name)
	}
	for row := table.FirstChild(); row != nil; row = row.NextSibling() {
		cells := []Cell{}
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			content := Paragraph{Runs: []Run{}}
			if !collectRuns(cell, source, Run{}, &content) {
				return sourceLines(raw)
			}
			cells = append(cells, Cell{Runs: content.Runs})
		}
		paragraph.Rows = append(paragraph.Rows, cells)
	}
	return []Paragraph{paragraph}
}

func serializeTable(paragraph Paragraph) string {
	if len(paragraph.Rows) == 0 {
		return ""
	}
	columns := len(paragraph.Rows[0])
	var output strings.Builder
	for index, row := range paragraph.Rows {
		if index > 0 {
			output.WriteByte('\n')
		}
		output.WriteByte('|')
		for column := 0; column < columns; column++ {
			value := ""
			if column < len(row) {
				value = serializeCell(row[column].Runs)
			}
			output.WriteString(" " + strings.TrimSpace(value) + " |")
		}
		if index == 0 {
			output.WriteString("\n|")
			for column := 0; column < columns; column++ {
				delimiter := "---"
				if column < len(paragraph.Alignments) {
					switch paragraph.Alignments[column] {
					case "left":
						delimiter = ":---"
					case "right":
						delimiter = "---:"
					case "center":
						delimiter = ":---:"
					}
				}
				output.WriteString(" " + delimiter + " |")
			}
		}
	}
	return output.String()
}
