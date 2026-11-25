package mdcodec

// Style is how the writer spells syntax that has several spellings.
type Style struct {
	Bullet       string // "-", "*" or "+"
	Emphasis     string // "*" or "_"
	Strong       string // "**" or "__"
	Fence        string // "`" or "~"
	OrderedDelim string // "." or ")"
	Rule         string // "---", "***" or "___"
}

// DefaultStyle is the most common GFM spelling.
func DefaultStyle() Style {
	return Style{Bullet: "-", Emphasis: "*", Strong: "**", Fence: "`", OrderedDelim: ".", Rule: "---"}
}
