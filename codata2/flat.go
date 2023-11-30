package codata2

import "github.com/takoeight0821/anma/ast"

// Flat converts Copatterns ([Access] and [This] in [Pattern]) into [Object] and [Lambda].
type Flat struct{}

func (Flat) Name() string {
	return "codata.Flat"
}

func (Flat) Init([]ast.Node) error {
	return nil
}

func (Flat) Run(program []ast.Node) ([]ast.Node, error) {
	for i, n := range program {
		var err error
		program[i], err = flat(n)
		if err != nil {
			return program, err
		}
	}
	return program, nil
}

func flat(n ast.Node) (ast.Node, error) {
	n, err := ast.Traverse(n, flatEach)
	if err != nil {
		return n, err
	}
	return n, nil
}

// flatEach converts Copatterns ([Access] and [This] in [Pattern]) into [Object] and [Lambda].
// If error occurred, return the original node and the error. Because ast.Traverse needs it.
func flatEach(n ast.Node, err error) (ast.Node, error) {
	// early return if error occurred
	if err != nil {
		return n, err
	}
	if c, ok := n.(*ast.Codata); ok {
		newNode, err := flatCodata(c)
		if err != nil {
			return n, err
		}
		return newNode, nil
	}
	return n, nil
}

func flatCodata(c *ast.Codata) (ast.Node, error) {
	clauses := make([]plistClause, len(c.Clauses))
	for i, clause := range c.Clauses {
		plist, err := newPlist(clause.Patterns)
		if err != nil {
			return nil, err
		}
		clauses[i] = plistClause{
			plist: plist,
			exprs: clause.Exprs,
		}
	}
	return build(clauses), nil
}

type plistClause struct {
	plist plist
	exprs []ast.Node
}

type plist struct {
	sequence []ast.Node
}
