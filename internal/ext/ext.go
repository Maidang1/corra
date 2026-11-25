// Package ext assembles editor extensions. An extension brings one piece
// of syntax whole: its node and mark types, Markdown rules and commands
// (and, from the editor, its views and keys). Built-in syntax uses the
// same interface as any other.
package ext

import (
	"fmt"
	"sort"

	"github.com/Maidang1/corra/internal/doc"
	"github.com/Maidang1/corra/internal/editor"
	"github.com/Maidang1/corra/internal/mdcodec"
)

// Extension is one piece of syntax.
type Extension struct {
	Name string
	// Priority orders extensions: higher first, so their rules and
	// command chains win.
	Priority int

	Nodes []doc.NodeSpec
	Marks []doc.MarkSpec

	Markdown mdcodec.Rules

	// Commands returns named commands for the built schema, which menus,
	// the command palette and keys call by name.
	Commands func(reg *doc.Registry) map[string]doc.Command
	// Chains adds commands to the dispatch chains of keys like "Enter",
	// "Backspace", "Delete", "Tab" and "Shift-Tab"; the earliest that
	// applies runs.
	Chains func(reg *doc.Registry) map[string][]doc.Command

	// Views say how the extension's node and mark types look, by name.
	Views map[string]editor.ViewSpec
	// Keys bind keys to named commands.
	Keys []editor.KeyBinding
	// InputRules turn typed syntax into structure.
	InputRules func(reg *doc.Registry) []editor.InputRule
	// Slash adds entries to the "/" menu.
	Slash []editor.SlashItem
	// Completions add popups typing opens, as ":" for emoji.
	Completions func(reg *doc.Registry) []editor.Completion
}

// Kit is a built set of extensions: one schema, one codec, all commands.
type Kit struct {
	Registry    *doc.Registry
	Codec       *mdcodec.Codec
	Commands    map[string]doc.Command
	Chains      map[string]doc.Command
	Views       map[string]editor.ViewSpec
	Keys        []editor.KeyBinding
	InputRules  []editor.InputRule
	Slash       []editor.SlashItem
	Completions []editor.Completion
	Extensions  []Extension
}

// EditorConfig returns an editor configuration with the kit's schema,
// codec, commands, views, keys and rules.
func (k *Kit) EditorConfig() editor.Config {
	return editor.Config{
		Registry: k.Registry, Codec: k.Codec, Commands: k.Commands, Chains: k.Chains,
		Views: k.Views, Keys: k.Keys, InputRules: k.InputRules, Slash: k.Slash, Completions: k.Completions,
	}
}

// Build registers exts into a new schema and codec.
func Build(exts ...Extension) (*Kit, error) {
	exts = append([]Extension(nil), exts...)
	sort.SliceStable(exts, func(i, j int) bool { return exts[i].Priority > exts[j].Priority })
	reg := doc.NewRegistry()
	var rules mdcodec.Rules
	// Lower priorities first, so higher ones override shared handlers.
	for i := len(exts) - 1; i >= 0; i-- {
		rules.Add(exts[i].Markdown)
	}
	// Register the lowest priority first: the core's paragraph is then
	// the first text block, which empty containers and new blocks default
	// to.
	for i := len(exts) - 1; i >= 0; i-- {
		e := exts[i]
		for _, n := range e.Nodes {
			if err := reg.AddNode(n); err != nil {
				return nil, fmt.Errorf("extension %s: %w", e.Name, err)
			}
		}
		for _, m := range e.Marks {
			if err := reg.AddMark(m); err != nil {
				return nil, fmt.Errorf("extension %s: %w", e.Name, err)
			}
		}
	}
	if err := reg.Build(); err != nil {
		return nil, err
	}
	k := &Kit{Registry: reg, Codec: mdcodec.New(reg, rules), Commands: map[string]doc.Command{}, Chains: map[string]doc.Command{}, Extensions: exts}
	chains := map[string][]doc.Command{}
	k.Views = editor.DefaultViews()
	for i := len(exts) - 1; i >= 0; i-- {
		for name, v := range exts[i].Views {
			k.Views[name] = v
		}
	}
	// The "/" menu lists core entries first, in registration order.
	for i := len(exts) - 1; i >= 0; i-- {
		k.Slash = append(k.Slash, exts[i].Slash...)
	}
	for _, e := range exts {
		k.Keys = append(k.Keys, e.Keys...)
		if e.Completions != nil {
			k.Completions = append(k.Completions, e.Completions(reg)...)
		}
		if e.InputRules != nil {
			k.InputRules = append(k.InputRules, e.InputRules(reg)...)
		}
		if e.Commands != nil {
			for name, c := range e.Commands(reg) {
				if _, dup := k.Commands[name]; dup {
					return nil, fmt.Errorf("extension %s: command %q defined twice", e.Name, name)
				}
				k.Commands[name] = c
			}
		}
		if e.Chains != nil {
			for key, cs := range e.Chains(reg) {
				chains[key] = append(chains[key], cs...)
			}
		}
	}
	for key, cs := range chains {
		k.Chains[key] = doc.FirstOf(cs...)
	}
	return k, nil
}

// MustBuild is Build, panicking on an error.
func MustBuild(exts ...Extension) *Kit {
	k, err := Build(exts...)
	if err != nil {
		panic(err)
	}
	return k
}

// Node returns the registered node type name, or panics: for extensions
// looking up their own types.
func Node(reg *doc.Registry, name string) *doc.NodeType {
	t := reg.Node(name)
	if t == nil {
		panic("ext: no node type " + name)
	}
	return t
}

// Mark returns the registered mark type name, or panics.
func Mark(reg *doc.Registry, name string) *doc.MarkType {
	t := reg.Mark(name)
	if t == nil {
		panic("ext: no mark type " + name)
	}
	return t
}
