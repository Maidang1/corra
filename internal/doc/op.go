package doc

import (
	"fmt"
	"slices"
)

// OpKind is which primitive an Op records.
type OpKind uint8

const (
	OpReplaceText OpKind = iota
	OpSetAttrs
	OpSetType
	OpReplaceChildren
)

func (k OpKind) String() string {
	switch k {
	case OpReplaceText:
		return "ReplaceText"
	case OpSetAttrs:
		return "SetAttrs"
	case OpSetType:
		return "SetType"
	case OpReplaceChildren:
		return "ReplaceChildren"
	}
	return fmt.Sprintf("OpKind(%d)", int(k))
}

// OpOrigin says what made a change.
type OpOrigin uint8

const (
	OriginCommand OpOrigin = iota
	OriginNormalize
	OriginSource
	OriginExternal
)

// Op is one primitive change with its old and new values, so it can be
// inverted without the document and replayed on the document before it.
// Values in an Op are immutable once recorded.
type Op struct {
	Kind   OpKind
	Origin OpOrigin
	// Target is the text block (ReplaceText), the node (SetAttrs,
	// SetType) or the parent (ReplaceChildren).
	Target NodeID
	// From and To are UTF-16 offsets (ReplaceText) or child indexes
	// (ReplaceChildren) of the replaced range before the change.
	From, To int

	OldText, NewText   Inline
	OldAttrs, NewAttrs Attrs
	OldType, NewType   *NodeType
	OldKids, NewKids   []*Node
}

// Invert returns the op that undoes o.
func (o Op) Invert() Op {
	r := o
	switch o.Kind {
	case OpReplaceText:
		r.To = o.From + o.NewText.Len()
		r.OldText, r.NewText = o.NewText, o.OldText
	case OpSetAttrs, OpSetType:
		r.OldAttrs, r.NewAttrs = o.NewAttrs, o.OldAttrs
		r.OldType, r.NewType = o.NewType, o.OldType
	case OpReplaceChildren:
		r.To = o.From + len(o.NewKids)
		r.OldKids, r.NewKids = slices.Clone(o.NewKids), slices.Clone(o.OldKids)
	}
	return r
}

func (o Op) String() string {
	switch o.Kind {
	case OpReplaceText:
		return fmt.Sprintf("ReplaceText(%d, %d..%d, %q→%q)", o.Target, o.From, o.To, o.OldText.String(), o.NewText.String())
	case OpSetAttrs:
		return fmt.Sprintf("SetAttrs(%d, %v→%v)", o.Target, o.OldAttrs.Map(), o.NewAttrs.Map())
	case OpSetType:
		return fmt.Sprintf("SetType(%d, %s→%s)", o.Target, o.OldType.Name(), o.NewType.Name())
	case OpReplaceChildren:
		return fmt.Sprintf("ReplaceChildren(%d, %d..%d, %d→%d nodes)", o.Target, o.From, o.To, len(o.OldKids), len(o.NewKids))
	}
	return o.Kind.String()
}

// Apply replays o in tx.
func (tx *Tx) Apply(o Op) {
	switch o.Kind {
	case OpReplaceText:
		tx.ReplaceText(o.Target, o.From, o.To, o.NewText)
	case OpSetAttrs:
		tx.SetAttrs(o.Target, o.NewAttrs.Map())
	case OpSetType:
		tx.SetType(o.Target, o.NewType, o.NewAttrs.Map())
	case OpReplaceChildren:
		tx.ReplaceChildren(o.Target, o.From, o.To, o.NewKids...)
	}
}
