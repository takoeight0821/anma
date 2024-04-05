package codata

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
)

// Flat converts [Codata] to [Object], [Case], and [Lambda].
type Flat struct {
	scrutinees []token.Token
	guards     map[int][]ast.Node
}

func (*Flat) Name() string {
	return "newcodata.Flat"
}

func (*Flat) Init([]ast.Node) error {
	return nil
}

func (f *Flat) Run(program []ast.Node) ([]ast.Node, error) {
	for i, n := range program {
		var err error
		program[i], err = f.flat(n)
		if err != nil {
			return program, err
		}
	}

	return program, nil
}

func (f *Flat) flat(node ast.Node) (ast.Node, error) {
	node, err := ast.Traverse(node, f.flatEach)
	if err != nil {
		return node, fmt.Errorf("flat %v: %w", node, err)
	}

	return node, nil
}

// flatEach flattens [Codata] nodes.
// If error occured, return the original node and the error. Because ast.Traverse needs it.
func (f *Flat) flatEach(node ast.Node, err error) (ast.Node, error) {
	// early return if error occured
	if err != nil {
		return node, err
	}

	if c, ok := node.(*ast.Codata); ok {
		newNode, err := f.flatCodata(c)
		if err != nil {
			return node, err
		}

		return newNode, nil
	}

	return node, nil
}

func (f *Flat) flatCodata(c *ast.Codata) (ast.Node, error) {
	f.scrutinees = make([]token.Token, 0)
	f.guards = make(map[int][]ast.Node)

	plists := make(map[int][]ast.Node)
	for i, clause := range c.Clauses {
		plists[i] = makePatternList(clause.Pattern)
	}

	bodys := make(map[int]ast.Node)
	for i, clause := range c.Clauses {
		bodys[i] = clause.Expr
	}

	return f.build(plists, bodys)
}

// makePatternList makes a sequence of patterns from a pattern.
// For example, if the pattern is `#.f(x, y)`, the sequence is `[#.f, #(x, y)]`.
func makePatternList(p ast.Node) []ast.Node {
	switch p := p.(type) {
	case *ast.This:
		return []ast.Node{}
	case *ast.Access:
		pl := makePatternList(p.Receiver)
		return append(pl, &ast.Access{Receiver: &ast.This{Token: p.Base()}, Name: p.Name})
	case *ast.Call:
		pl := makePatternList(p.Func)
		return append(pl, &ast.Call{Func: &ast.This{Token: p.Base()}, Args: p.Args})
	case *ast.Paren:
		return makePatternList(p.Expr)
	default:
		panic(fmt.Sprintf("unexpected pattern: %v", p))
	}
}

func (f *Flat) build(plists map[int][]ast.Node, bodys map[int]ast.Node) (ast.Node, error) {
	if allEmpty(plists) {
		return f.buildCase(plists, bodys), nil
	}
	kind := kindOf(plists)
	switch kind {
	case Field:
		return f.buildObject(plists, bodys)
	case Function:
		return f.buildLambda(plists, bodys)
	default:
		return nil, mismatchError(plists)
	}
}

func allEmpty(plists map[int][]ast.Node) bool {
	for _, ps := range plists {
		if len(ps) != 0 {
			return false
		}
	}
	return true
}

type Kind int

const (
	Field Kind = iota
	Function
	Mismatch
)

func kindOf(plists map[int][]ast.Node) Kind {
	kind := Mismatch
	for _, ps := range plists {
		if len(ps) == 0 {
			return kind
		}

		switch ps[0].(type) {
		case *ast.Access:
			if kind == Mismatch {
				kind = Field
			}
			if kind != Field {
				return Mismatch
			}
		case *ast.Call:
			if kind == Mismatch {
				kind = Function
			}
			if kind != Function {
				return Mismatch
			}
		default:
			return Mismatch
		}
	}
	return kind
}

func mismatchError(plists map[int][]ast.Node) error {
	var builder strings.Builder
	fmt.Fprintf(&builder, "mismatched patterns:\n")
	for _, ps := range plists {
		fmt.Fprintf(&builder, "\t%v\n", ps[0])
	}
	return fmt.Errorf(builder.String())
}

func (f *Flat) buildCase(plists map[int][]ast.Node, bodys map[int]ast.Node) ast.Node {
	if len(f.scrutinees) == 0 {
		// If there is no scrutinee, generate a body.
		// Use the topmost body.
		topmost := searchTopmost(plists)
		return bodys[topmost]
	}

	plistsKeys := make([]int, 0, len(plists))
	for k := range plists {
		plistsKeys = append(plistsKeys, k)
	}
	slices.Sort(plistsKeys)

	var clauses [](*ast.CaseClause)
	for _, i := range plistsKeys {
		clauses = append(clauses, &ast.CaseClause{
			Patterns: f.guards[i],
			Expr:     bodys[i],
		})
	}

	scrutinees := make([]ast.Node, len(f.scrutinees))
	for i, s := range f.scrutinees {
		scrutinees[i] = &ast.Var{Name: s}
	}

	return &ast.Case{
		Scrutinees: scrutinees,
		Clauses:    clauses,
	}
}

func searchTopmost(plists map[int][]ast.Node) int {
	topmost := math.MaxInt
	for i := range plists {
		if i < topmost {
			topmost = i
		}
	}

	return topmost
}

func (f *Flat) buildObject(plists map[int][]ast.Node, bodys map[int]ast.Node) (ast.Node, error) {
	fields, rest, err := popField(plists)
	if err != nil {
		return nil, err
	}

	fieldsKeys := make([]string, 0, len(fields))
	for k := range fields {
		fieldsKeys = append(fieldsKeys, k)
	}
	slices.Sort(fieldsKeys)

	objectFields := make([]*ast.Field, 0)
	for _, name := range fieldsKeys {
		innerF := &Flat{scrutinees: f.scrutinees, guards: selectIndicies(fields[name], f.guards)}
		expr, err := innerF.build(selectIndicies(fields[name], rest), bodys)
		if err != nil {
			return nil, err
		}

		objectFields = append(objectFields, &ast.Field{
			Name: name,
			Expr: expr,
		})
	}

	return &ast.Object{
		Fields: objectFields,
	}, nil
}

func selectIndicies(indices []int, original map[int][]ast.Node) map[int][]ast.Node {
	selected := make(map[int][]ast.Node)
	for _, i := range indices {
		selected[i] = original[i]
	}
	return selected
}

func popField(plists map[int][]ast.Node) (map[string][]int, map[int][]ast.Node, error) {
	fields := make(map[string][]int)
	rest := make(map[int][]ast.Node)
	for i, ps := range plists {
		switch p := ps[0].(type) {
		case *ast.Access:
			fields[p.Name.Lexeme] = append(fields[p.Name.Lexeme], i)
		default:
			return nil, nil, fmt.Errorf("unexpected pattern: %v", p)
		}
		rest[i] = ps[1:]
	}
	return fields, rest, nil
}

func (f *Flat) buildLambda(plists map[int][]ast.Node, bodys map[int]ast.Node) (ast.Node, error) {
	guards, rest, err := popGuard(plists)
	if err != nil {
		return nil, err
	}

	for i, ps := range guards {
		guards[i] = append(f.guards[i], ps...)
	}

	arity := -1
	for _, ps := range guards {
		if arity == -1 {
			arity = len(ps)
		}
		if len(ps) != arity {
			return nil, fmt.Errorf("mismatched arity: %v", guards)
		}
	}
	if arity == -1 {
		return nil, fmt.Errorf("arity is not defined: %v", guards)
	}

	scrutinees := make([]token.Token, arity)
	for i := range scrutinees {
		// TODO: add line number from the original pattern
		scrutinees[i] = token.Token{Kind: token.IDENT, Lexeme: fmt.Sprintf("x%d", i), Line: 0, Literal: nil}
	}

	innerF := &Flat{scrutinees: append(f.scrutinees, scrutinees...), guards: guards}
	body, err := innerF.build(rest, bodys)
	if err != nil {
		return nil, err
	}

	return &ast.Lambda{
		Params: scrutinees,
		Expr:   body}, nil
}

func popGuard(plists map[int][]ast.Node) (map[int][]ast.Node, map[int][]ast.Node, error) {
	guards := make(map[int][]ast.Node)
	rest := make(map[int][]ast.Node)
	for i, ps := range plists {
		switch p := ps[0].(type) {
		case *ast.Call:
			guards[i] = p.Args
		default:
			return nil, nil, fmt.Errorf("unexpected pattern: %v", p)
		}
		rest[i] = ps[1:]
	}
	return guards, rest, nil
}
