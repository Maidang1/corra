package render

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/yuin/goldmark/ast"
)

// mathPlan is the ordered math context for one immutable document snapshot.
// Nodes use AST identity so identical equations keep distinct numbers.
type mathPlan struct {
	Expressions []MathExpression
	Nodes       map[ast.Node]int
}

func collectMath(root ast.Node) mathPlan {
	plan := mathPlan{Nodes: make(map[ast.Node]int)}
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var expression MathExpression
		switch n := node.(type) {
		case *ast.Image:
			return ast.WalkSkipChildren, nil
		case *mathInlineNode:
			expression.TeX, expression.Display = n.TeX, n.Display
		case *mathBlockNode:
			if n.open {
				return ast.WalkSkipChildren, nil
			}
			expression.TeX, expression.Display = n.TeX, true
		default:
			return ast.WalkContinue, nil
		}
		plan.Nodes[node] = len(plan.Expressions)
		plan.Expressions = append(plan.Expressions, expression)
		return ast.WalkSkipChildren, nil
	})
	return plan
}

// MathKey changes when any expression or option can affect another equation.
func MathKey(expressions []MathExpression, options MathOptions) string {
	data, _ := json.Marshal(struct {
		Expressions []MathExpression
		Options     MathOptions
	}{expressions, options})
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// MathIndex returns the batch slot of a top-level display equation.
func (d *Document) MathIndex(block int) (int, bool) {
	index, ok := d.mathBlocks[block]
	return index, ok
}
