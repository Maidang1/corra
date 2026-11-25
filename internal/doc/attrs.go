package doc

import (
	"fmt"
	"reflect"
	"slices"
)

// A is shorthand for the attribute maps constructors take.
type A = map[string]any

// Attrs are a node's or a mark's attributes. They are immutable: With
// returns a new value. Keys and defaults come from the type's spec.
type Attrs struct{ m map[string]any }

// NewAttrs copies kv into attributes.
func NewAttrs(kv map[string]any) Attrs {
	if len(kv) == 0 {
		return Attrs{}
	}
	m := make(map[string]any, len(kv))
	for k, v := range kv {
		if v != nil {
			m[k] = v
		}
	}
	return Attrs{m}
}

func (a Attrs) Get(k string) any { return a.m[k] }

func (a Attrs) Has(k string) bool {
	_, ok := a.m[k]
	return ok
}

func (a Attrs) String(k string) string {
	s, _ := a.m[k].(string)
	return s
}

func (a Attrs) Bool(k string) bool {
	b, _ := a.m[k].(bool)
	return b
}

func (a Attrs) Int(k string) int {
	switch v := a.m[k].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	}
	return 0
}

func (a Attrs) Float(k string) float64 {
	switch v := a.m[k].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	}
	return 0
}

func (a Attrs) Len() int { return len(a.m) }

// Keys returns the attribute names, sorted.
func (a Attrs) Keys() []string {
	keys := make([]string, 0, len(a.m))
	for k := range a.m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// With returns a copy with k set to v; a nil v removes k.
func (a Attrs) With(k string, v any) Attrs {
	m := make(map[string]any, len(a.m)+1)
	for kk, vv := range a.m {
		m[kk] = vv
	}
	if v == nil {
		delete(m, k)
	} else {
		m[k] = v
	}
	return Attrs{m}
}

// Map returns a copy of the attributes as a map.
func (a Attrs) Map() map[string]any {
	m := make(map[string]any, len(a.m))
	for k, v := range a.m {
		m[k] = v
	}
	return m
}

func (a Attrs) Equal(b Attrs) bool {
	if len(a.m) != len(b.m) {
		return false
	}
	for k, v := range a.m {
		w, ok := b.m[k]
		if !ok || !reflect.DeepEqual(v, w) {
			return false
		}
	}
	return true
}

// AttrSpec declares one attribute. A nil Default makes it required.
type AttrSpec struct {
	Default  any
	Validate func(any) error
}

// resolveAttrs checks given against specs and fills in defaults.
func resolveAttrs(specs map[string]AttrSpec, given Attrs, owner string) (Attrs, error) {
	for k := range given.m {
		if _, ok := specs[k]; !ok {
			return Attrs{}, fmt.Errorf("doc: %s has no attribute %q", owner, k)
		}
	}
	if len(specs) == 0 {
		return Attrs{}, nil
	}
	m := make(map[string]any, len(specs))
	for k, s := range specs {
		v, ok := given.m[k]
		if !ok {
			if s.Default == nil {
				return Attrs{}, fmt.Errorf("doc: %s requires attribute %q", owner, k)
			}
			v = s.Default
		}
		if s.Validate != nil {
			if err := s.Validate(v); err != nil {
				return Attrs{}, fmt.Errorf("doc: %s attribute %q: %w", owner, k, err)
			}
		}
		m[k] = v
	}
	return Attrs{m}, nil
}
