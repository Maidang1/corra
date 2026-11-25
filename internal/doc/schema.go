package doc

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ContentKind is what a node holds.
type ContentKind uint8

const (
	// ContentNone is an atom: everything it holds is in its attributes.
	ContentNone ContentKind = iota
	// ContentInline is a text block of marked text and inline atoms.
	ContentInline
	// ContentText is a text block of plain text: no marks, no atoms.
	ContentText
	// ContentBlocks is a container of child blocks.
	ContentBlocks
)

func (k ContentKind) String() string {
	switch k {
	case ContentNone:
		return "none"
	case ContentInline:
		return "inline"
	case ContentText:
		return "text"
	case ContentBlocks:
		return "blocks"
	}
	return fmt.Sprintf("ContentKind(%d)", int(k))
}

// NodeSpec declares a node type.
type NodeSpec struct {
	Name string
	// Group is one or more space-separated group names, as "block".
	Group   string
	Content ContentKind
	// Allow lists the type and group names a container may hold, or the
	// marks and inline atom types (and groups) a text block may hold.
	// Empty allows every non-inline type in a container and everything
	// in a text block.
	Allow []string
	Attrs map[string]AttrSpec
	// Inline makes an atom sit inside a text block's content.
	Inline bool
	// Isolating keeps editing operations from crossing the node's edges,
	// as a table cell or a callout.
	Isolating bool
	// Defining keeps the node when its content is replaced or pasted
	// over, as a code block or a heading.
	Defining bool
	// Normalize repairs a node of this type after an edit, through fix.
	Normalize func(n *Node, fix *Fixer)
}

// NodeType is a registered node type.
type NodeType struct {
	spec          NodeSpec
	groups        []string
	allow         map[*NodeType]bool
	allowMarks    map[*MarkType]bool
	allowAllMarks bool
	order         int
	reg           *Registry
}

func (t *NodeType) Name() string         { return t.spec.Name }
func (t *NodeType) Spec() NodeSpec       { return t.spec }
func (t *NodeType) Content() ContentKind { return t.spec.Content }
func (t *NodeType) Registry() *Registry  { return t.reg }

// IsTextBlock reports whether the type holds text (marked or plain).
func (t *NodeType) IsTextBlock() bool {
	return t.spec.Content == ContentInline || t.spec.Content == ContentText
}

func (t *NodeType) IsContainer() bool { return t.spec.Content == ContentBlocks }
func (t *NodeType) IsAtom() bool      { return t.spec.Content == ContentNone }
func (t *NodeType) IsInline() bool    { return t.spec.Inline }

func (t *NodeType) InGroup(g string) bool { return slices.Contains(t.groups, g) }

// Allows reports whether a node of type t may hold a child of type c:
// a block in a container, an inline atom in a text block.
func (t *NodeType) Allows(c *NodeType) bool {
	switch t.spec.Content {
	case ContentBlocks:
		return !c.spec.Inline && t.allow[c]
	case ContentInline:
		return c.spec.Inline && t.allow[c]
	}
	return false
}

// AllowsMark reports whether text in a block of type t may carry m.
func (t *NodeType) AllowsMark(m *MarkType) bool {
	if t.spec.Content != ContentInline {
		return false
	}
	return t.allowAllMarks || t.allowMarks[m]
}

// Registry holds the node and mark types of a schema. Register every
// type, then Build once before creating nodes.
type Registry struct {
	nodes    map[string]*NodeType
	nodeList []*NodeType
	marks    map[string]*MarkType
	markList []*MarkType
	built    bool
}

func NewRegistry() *Registry {
	return &Registry{nodes: map[string]*NodeType{}, marks: map[string]*MarkType{}}
}

func (r *Registry) AddNode(s NodeSpec) error {
	if r.built {
		return errors.New("doc: registry already built")
	}
	if s.Name == "" {
		return errors.New("doc: node spec without a name")
	}
	if _, dup := r.nodes[s.Name]; dup {
		return fmt.Errorf("doc: node %q registered twice", s.Name)
	}
	if s.Inline && s.Content != ContentNone {
		return fmt.Errorf("doc: inline node %q must be an atom", s.Name)
	}
	t := &NodeType{spec: s, groups: strings.Fields(s.Group), order: len(r.nodeList), reg: r}
	r.nodes[s.Name] = t
	r.nodeList = append(r.nodeList, t)
	return nil
}

func (r *Registry) AddMark(s MarkSpec) error {
	if r.built {
		return errors.New("doc: registry already built")
	}
	if s.Name == "" {
		return errors.New("doc: mark spec without a name")
	}
	if _, dup := r.marks[s.Name]; dup {
		return fmt.Errorf("doc: mark %q registered twice", s.Name)
	}
	t := &MarkType{spec: s, rank: s.Rank}
	r.marks[s.Name] = t
	r.markList = append(r.markList, t)
	return nil
}

// Build resolves names in Allow and Excludes. The schema must have a
// container named "doc", the root.
func (r *Registry) Build() error {
	if r.built {
		return nil
	}
	root, ok := r.nodes["doc"]
	if !ok || root.spec.Content != ContentBlocks {
		return errors.New(`doc: schema needs a container named "doc"`)
	}
	for _, t := range r.nodeList {
		switch t.spec.Content {
		case ContentBlocks:
			t.allow = map[*NodeType]bool{}
			if len(t.spec.Allow) == 0 {
				for _, c := range r.nodeList {
					if !c.spec.Inline {
						t.allow[c] = true
					}
				}
			}
			for _, name := range t.spec.Allow {
				found := false
				for _, c := range r.nodeList {
					if c.spec.Name == name || c.InGroup(name) {
						t.allow[c] = true
						found = true
					}
				}
				if !found {
					return fmt.Errorf("doc: %s allows unknown %q", t.spec.Name, name)
				}
			}
		case ContentInline:
			t.allow = map[*NodeType]bool{}
			t.allowMarks = map[*MarkType]bool{}
			if len(t.spec.Allow) == 0 {
				t.allowAllMarks = true
				for _, c := range r.nodeList {
					if c.spec.Inline {
						t.allow[c] = true
					}
				}
			}
			for _, name := range t.spec.Allow {
				found := false
				if m, ok := r.marks[name]; ok {
					t.allowMarks[m] = true
					found = true
				}
				for _, c := range r.nodeList {
					if c.spec.Inline && (c.spec.Name == name || c.InGroup(name)) {
						t.allow[c] = true
						found = true
					}
				}
				if !found {
					return fmt.Errorf("doc: %s allows unknown %q", t.spec.Name, name)
				}
			}
		}
	}
	for _, m := range r.markList {
		m.excl = map[*MarkType]bool{}
		for _, name := range m.spec.Excludes {
			if name == "_" {
				m.exclAll = true
				continue
			}
			o, ok := r.marks[name]
			if !ok {
				return fmt.Errorf("doc: mark %s excludes unknown %q", m.spec.Name, name)
			}
			m.excl[o] = true
		}
	}
	// Exclusion is directional, as in ProseMirror: a new mark drops those
	// it excludes, and is not added where an existing mark excludes it.
	r.built = true
	return nil
}

// MustBuild is Build, panicking on an error.
func (r *Registry) MustBuild() *Registry {
	if err := r.Build(); err != nil {
		panic(err)
	}
	return r
}

func (r *Registry) Built() bool { return r.built }

// Node returns the node type named name, or nil.
func (r *Registry) Node(name string) *NodeType { return r.nodes[name] }

// Mark returns the mark type named name, or nil.
func (r *Registry) Mark(name string) *MarkType { return r.marks[name] }

// Root returns the "doc" type.
func (r *Registry) Root() *NodeType { return r.nodes["doc"] }

func (r *Registry) Nodes() []*NodeType { return slices.Clone(r.nodeList) }
func (r *Registry) Marks() []*MarkType { return slices.Clone(r.markList) }

// DefaultTextBlock returns the first registered text block a container of
// type parent may hold, preferring marked text; nil if there is none.
func (r *Registry) DefaultTextBlock(parent *NodeType) *NodeType {
	var plain *NodeType
	for _, t := range r.nodeList {
		if !parent.Allows(t) {
			continue
		}
		if t.spec.Content == ContentInline {
			return t
		}
		if t.spec.Content == ContentText && plain == nil {
			plain = t
		}
	}
	return plain
}

// DefaultChild returns the type to fill an empty container of type
// parent with: a text block if allowed, else the first allowed container
// that can in turn be filled.
func (r *Registry) DefaultChild(parent *NodeType) *NodeType {
	if t := r.DefaultTextBlock(parent); t != nil {
		return t
	}
	for _, t := range r.nodeList {
		if parent.Allows(t) && t.IsContainer() && t != parent && r.DefaultTextBlock(t) != nil {
			return t
		}
	}
	return nil
}

// Wrapper returns a container type parent may hold that may in turn hold
// child, or nil.
func (r *Registry) Wrapper(parent, child *NodeType) *NodeType {
	for _, t := range r.nodeList {
		if t.IsContainer() && parent.Allows(t) && t.Allows(child) {
			return t
		}
	}
	return nil
}
