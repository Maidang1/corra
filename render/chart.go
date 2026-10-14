package render

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ChartSpec is the parsed model of a ```chart fenced block. It is UI-agnostic;
// views paint it natively.
type ChartSpec struct {
	Type   string   `yaml:"type"` // line, bar, hbar, pie
	Title  string   `yaml:"title"`
	Labels []string `yaml:"labels"`
	Series []Series `yaml:"series"`
	Width  int      `yaml:"width"` // optional viewbox hints
	Height int      `yaml:"height"`
}

// Series is one named data series.
type Series struct {
	Name string    `yaml:"name"`
	Data []float64 `yaml:"data"`
}

// ParseChart parses the YAML body of a ```chart block.
func ParseChart(src string) (*ChartSpec, error) {
	var spec ChartSpec
	if err := yaml.Unmarshal([]byte(src), &spec); err != nil {
		return nil, fmt.Errorf("chart yaml: %w", err)
	}
	switch spec.Type {
	case "line", "bar", "hbar", "pie":
	default:
		return nil, fmt.Errorf("chart: unknown type %q (want line, bar, hbar or pie)", spec.Type)
	}
	if len(spec.Series) == 0 {
		return nil, fmt.Errorf("chart: at least one series is required")
	}
	if spec.Type != "pie" && len(spec.Labels) == 0 {
		return nil, fmt.Errorf("chart: labels are required")
	}
	for _, s := range spec.Series {
		if len(s.Data) != len(spec.Labels) {
			return nil, fmt.Errorf("chart: series %q has %d points, want %d", s.Name, len(s.Data), len(spec.Labels))
		}
	}
	return &spec, nil
}
