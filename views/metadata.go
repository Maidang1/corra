package views

import (
	"strings"

	"github.com/Maidang1/corra/internal/mdext"
	"github.com/egoist/mygo/ui"
	"gopkg.in/yaml.v3"
)

type documentMetadata struct {
	prefix, body string
	tags         []string
}

func parseDocumentMetadata(source string) documentMetadata {
	result := documentMetadata{body: source}
	fm, body := mdext.SplitFrontMatter(source)
	if fm == "" {
		return result
	}
	start := strings.IndexByte(fm, '\n') + 1
	end := strings.LastIndex(strings.TrimRight(fm, "\r\n"), "\n") + 1
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(fm[start:end]), &doc); err != nil || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return result
	}
	labels := map[string]string{"title": "标题", "author": "作者", "source": "来源", "published": "发布", "license": "许可", "fetched": "获取", "date": "日期", "description": "描述"}
	mapping := doc.Content[0]
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i].Value, mapping.Content[i+1]
		if strings.EqualFold(key, "tags") {
			values := []*yaml.Node{value}
			if value.Kind == yaml.SequenceNode {
				values = value.Content
			}
			for _, tag := range values {
				if text := metadataValue(tag); text != "" {
					result.tags = append(result.tags, "#"+text)
				}
			}
			continue
		}
		label := labels[strings.ToLower(key)]
		if label == "" {
			label = key
		}
		if text := metadataValue(value); text != "" {
			result.tags = append(result.tags, label+"："+text)
		}
	}
	// Keep fence spelling, comments, CRLFs and separator lines byte for byte.
	trimmed := strings.TrimLeft(body, "\r\n")
	result.prefix = fm + body[:len(body)-len(trimmed)]
	result.body = trimmed
	return result
}

func metadataValue(node *yaml.Node) string {
	if node.Kind == yaml.ScalarNode {
		return strings.Join(strings.Fields(node.Value), " ")
	}
	data, err := yaml.Marshal(node)
	if err != nil {
		return ""
	}
	return strings.Join(strings.Fields(string(data)), " ")
}

func (v *View) documentMetadata() documentMetadata {
	if v.metadataSource != v.App.Source {
		v.metadataSource = v.App.Source
		v.metadata = parseDocumentMetadata(v.App.Source)
	}
	return v.metadata
}

func (v *View) metadataHeader(c *ui.Context) {
	meta := v.documentMetadata()
	if len(meta.tags) == 0 {
		return
	}
	t := c.Theme()
	ui.Row(c).FillWidth().MaxWidth(760*v.App.Zoom).Wrap().Gap(6).Padding(6, 12, 8, 12).Children(func() {
		for _, label := range meta.tags {
			ui.Box(c).MaxWidth(260).Padding(4, 9).Radius(10).Background(t.SurfaceHover).Border(1, t.Border).Tooltip(label).Children(func() {
				ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).SingleLine()
			})
		}
	})
}
