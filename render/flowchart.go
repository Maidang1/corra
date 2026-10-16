package render

import (
	"regexp"
	"strings"
)

// flowchart.js (adrai/flowchart.js) is the engine behind Typora's ```flow
// fence. Its grammar is small and stable, so it is translated into an
// equivalent mermaid flowchart and rendered by the shared engine instead of
// carrying a second layout engine. Anything the translator does not recognise
// is an error, so the caller keeps the existing "render failure shows source"
// behaviour instead of drawing a wrong diagram.
//
//	nodeName=>nodeType: nodeText[|flowstate][:>urlLink]
//	node1->node2->node3
//	cond(yes)->node4

var (
	flowchartNodeRe    = regexp.MustCompile(`^\s*([A-Za-z0-9_.\-]+)\s*=>\s*([A-Za-z]+)\s*:\s*(.*)$`)
	flowchartSegmentRe = regexp.MustCompile(`^\s*([A-Za-z0-9_.\-]+)\s*(?:\(([^)]*)\))?\s*$`)
	flowchartArrowRe   = regexp.MustCompile(`\s*-+>\s*`)
	flowchartIDRe      = regexp.MustCompile(`[^A-Za-z0-9_]`)
	mermaidLabelRe     = regexp.MustCompile("[\"#|<>\n]")
)

// flowchartShapes maps flowchart.js node types onto mermaid shape delimiters.
// An unknown type falls back to a plain rectangle rather than failing.
var flowchartShapes = map[string][2]string{
	"start":       {"([", "])"}, // stadium
	"end":         {"([", "])"},
	"operation":   {"[", "]"},   // process box
	"inputoutput": {"[/", "/]"}, // parallelogram
	"input":       {"[/", "/]"},
	"output":      {"[\\", "\\]"}, // leaning the other way
	"subroutine":  {"[[", "]]"},   // predefined process
	"condition":   {"{", "}"},     // decision diamond
	"parallel":    {"{{", "}}"},   // parallel bar
}

type flowNode struct {
	label string
	shape [2]string
	link  string
}

// flowchartToMermaid converts flowchart.js source to mermaid flowchart source.
func flowchartToMermaid(src string) (string, error) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	nodes := map[string]*flowNode{}
	var order []string
	var edges []string

	node := func(id string) *flowNode {
		if n, ok := nodes[id]; ok {
			return n
		}
		n := &flowNode{label: id, shape: [2]string{"[", "]"}}
		nodes[id], order = n, append(order, id)
		return n
	}

	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if m := flowchartNodeRe.FindStringSubmatch(line); m != nil {
			label, link := splitFlowLabel(m[3])
			n := node(m[1])
			n.label = label
			if shape, ok := flowchartShapes[strings.ToLower(m[2])]; ok {
				n.shape = shape
			}
			n.link = link
			continue
		}
		if !strings.Contains(line, "->") {
			return "", errUnsupportedDiagram
		}
		parts := flowchartArrowRe.Split(line, -1)
		if len(parts) < 2 {
			return "", errUnsupportedDiagram
		}
		// "cond(yes)->e" labels the edge leaving cond, so the path names travel
		// with the source segment until the next arrow is consumed.
		previous, previousLabels := "", []string(nil)
		for _, part := range parts {
			m := flowchartSegmentRe.FindStringSubmatch(part)
			if m == nil {
				return "", errUnsupportedDiagram
			}
			node(m[1])
			if previous != "" {
				edges = append(edges, mermaidEdge(mermaidID(previous), previousLabels, mermaidID(m[1])))
			}
			previous = m[1]
			previousLabels = pathLabels(m[2])
		}
	}
	if len(order) == 0 {
		return "", errUnsupportedDiagram
	}

	var out strings.Builder
	out.WriteString("flowchart TD\n")
	for _, id := range order {
		n := nodes[id]
		out.WriteString("    ")
		out.WriteString(mermaidID(id))
		out.WriteString(n.shape[0])
		out.WriteString(quoteMermaidLabel(n.label))
		out.WriteString(n.shape[1])
		out.WriteString("\n")
		// flowchart.js's ":>url" makes the node clickable. Mermaid's "click"
		// emits that URL as a visible text element, so the link is dropped
		// rather than rendered as junk.
	}
	for _, edge := range edges {
		out.WriteString("    ")
		out.WriteString(edge)
		out.WriteString("\n")
	}
	return out.String(), nil
}

// splitFlowLabel separates "text|flowstate:>url[target]" into label and URL.
// flowstate is a CSS class and the trailing [target] is a link window name;
// neither belongs in the rendered label.
func splitFlowLabel(text string) (label, link string) {
	if at := strings.Index(text, ":>"); at >= 0 {
		label, link = text[:at], text[at+2:]
		if open := strings.IndexByte(link, '['); open >= 0 {
			link = link[:open]
		}
	} else {
		label = text
	}
	if pipe := strings.IndexByte(label, '|'); pipe >= 0 {
		label = label[:pipe]
	}
	return strings.TrimSpace(label), strings.TrimSpace(link)
}

// pathLabels keeps the names in "cond(yes)" / "para(path1, bottom)" that are
// not layout directions.
func pathLabels(inside string) []string {
	var labels []string
	for _, part := range strings.Split(inside, ",") {
		part = strings.TrimSpace(part)
		if part == "" || isFlowDirection(part) {
			continue
		}
		labels = append(labels, part)
	}
	return labels
}

func isFlowDirection(part string) bool {
	switch strings.ToLower(part) {
	case "left", "right", "top", "bottom":
		return true
	}
	return false
}

func mermaidID(id string) string {
	// Mermaid node ids must be plain identifiers, and a few bare words are
	// keywords in its grammar.
	return "n" + flowchartIDRe.ReplaceAllString(id, "_")
}

// mermaidEdge renders one flowchart.js connection, labelling it with the
// source node's path names.
func mermaidEdge(fromID string, labels []string, toID string) string {
	if len(labels) == 0 {
		return fromID + " --> " + toID
	}
	// Unquoted, because |"yes"| renders the quotes as part of the label.
	return fromID + " -->|" + mermaidEdgeLabel(labels) + "| " + toID
}

// mermaidEdgeLabel makes one path name safe between "|" delimiters.
func mermaidEdgeLabel(labels []string) string {
	text := strings.Join(labels, "/")
	return mermaidLabelRe.ReplaceAllString(text, "")
}

// quoteMermaidLabel wraps a label in double quotes, which is the only form that
// survives labels containing ":", "#" or brackets.
func quoteMermaidLabel(label string) string {
	// An empty label ("e=>end:>url") draws an empty shape; the id reads better.
	text := mermaidLabelRe.ReplaceAllString(label, "")
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return label
	}
	return "\"" + text + "\""
}
