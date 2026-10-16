package render

import (
	"regexp"
	"strings"
)

// sequence.js (bramp/js-sequence-diagrams) is the engine behind Typora's
// ```sequence fence. Its statements map one-to-one onto mermaid's
// sequenceDiagram, so it is translated and rendered by the shared engine.
// Unrecognised lines are an error, keeping the source-card fallback.

var (
	// A->B: message, B-->C: message, C->>D, D-->>E. The longer arrows come
	// first: leftmost-first alternation would otherwise stop at "->" inside "->>".
	sequenceMessageRe = regexp.MustCompile(`^\s*([^\-:>\s][^\-:>]*?)\s*(-->>|->>|-->|->)\s*([^:\s][^:]*?)\s*(?::\s*(.*))?$`)
	// Note right of Bob: text / Note over A,B: text
	// sequence.js spells it "Note"; match it case-insensitively.
	sequenceNoteRe = regexp.MustCompile(`(?i)^\s*note\s+(over|left\s+of|right\s+of)\s+([^:]+?)\s*:\s*(.*)$`)
	// participant A / participant A as B
	sequenceParticipantRe = regexp.MustCompile(`^\s*participant\s+(\S+)(?:\s+as\s+(.+))?$`)
)

// sequenceToMermaid converts sequence.js source to mermaid sequenceDiagram.
func sequenceToMermaid(src string) (string, error) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	var out strings.Builder
	out.WriteString("sequenceDiagram\n")
	participants := map[string]bool{}
	statements := 0

	emitParticipant := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || participants[name] {
			return
		}
		participants[name] = true
		out.WriteString("    participant ")
		out.WriteString(name)
		out.WriteString("\n")
	}

	for _, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		// A title is presentation only; mermaid has no sequence title.
		if strings.HasPrefix(strings.ToLower(line), "title:") {
			continue
		}
		if m := sequenceParticipantRe.FindStringSubmatch(line); m != nil {
			emitParticipant(m[1])
			if alias := strings.TrimSpace(m[2]); alias != "" {
				// "participant A as B" already rendered A; re-emit with alias.
				out.WriteString("    participant ")
				out.WriteString(strings.TrimSpace(m[1]))
				out.WriteString(" as ")
				out.WriteString(alias)
				out.WriteString("\n")
			}
			statements++
			continue
		}
		if m := sequenceNoteRe.FindStringSubmatch(line); m != nil {
			out.WriteString("    Note ")
			out.WriteString(normalizeSpace(m[1]))
			out.WriteString(" ")
			out.WriteString(normalizeSpace(m[2]))
			out.WriteString(": ")
			out.WriteString(sequenceMessageText(m[3]))
			out.WriteString("\n")
			statements++
			continue
		}
		if m := sequenceMessageRe.FindStringSubmatch(line); m != nil {
			from, arrow, to, text := strings.TrimSpace(m[1]), m[2], strings.TrimSpace(m[3]), m[4]
			emitParticipant(from)
			emitParticipant(to)
			out.WriteString("    ")
			out.WriteString(from)
			out.WriteString(" ")
			// sequence.js "->" is a solid arrow and "-->" a dashed one; both
			// carry an arrowhead, so mermaid's ->> forms are the match.
			out.WriteString(sequenceArrow(arrow))
			out.WriteString(" ")
			out.WriteString(to)
			if strings.TrimSpace(text) != "" {
				out.WriteString(": ")
				out.WriteString(sequenceMessageText(text))
			}
			out.WriteString("\n")
			statements++
			continue
		}
		return "", errUnsupportedDiagram
	}
	if statements == 0 {
		return "", errUnsupportedDiagram
	}
	return out.String(), nil
}

func sequenceArrow(arrow string) string {
	if strings.Contains(arrow, "--") {
		return "-->>"
	}
	return "->>"
}

// sequenceMessageText keeps a message on one mermaid statement: a semicolon
// would terminate it early, and a quote would end the quoted label.
func sequenceMessageText(text string) string {
	return strings.NewReplacer(";", ",", "\n", " ").Replace(strings.TrimSpace(text))
}

func normalizeSpace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
