package codata

import (
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

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
		program[i] = flat(n)
	}
	return program, nil
}

func flat(n ast.Node) ast.Node {
	return ast.Traverse(n, flatEach)
}

func flatEach(n ast.Node) ast.Node {
	if n, ok := n.(*ast.Codata); ok {
		return flatCodata(n)
	}
	return n
}

const (
	notChecked = -2
	noArgs     = -1
	zeroArgs   = 0
)

type ArityError struct {
	Expected int // expected arity, or notChecked, or noArgs
	Where    token.Token
}

func (e ArityError) Error() string {
	if e.Expected == notChecked {
		return utils.MsgAt(e.Where, fmt.Sprintf("unreachable: arity is not checked"))
	}
	return utils.MsgAt(e.Where, fmt.Sprintf("arity mismatch: expected %d arguments", e.Expected))
}

func checkArity(expected, actual int, where token.Token) {
	if expected == notChecked {
		return
	}
	if expected != actual {
		panic(ArityError{Expected: expected, Where: where})
	}
}

func flatCodata(c *ast.Codata) ast.Node {
	// Generate PatternList
	arity := notChecked
	clauses := make([]*ast.Clause, len(c.Clauses))
	for i, cl := range c.Clauses {
		plist := NewPatternList(cl)
		clauses[i] = &ast.Clause{Patterns: []ast.Node{plist}, Exprs: cl.Exprs}
		if arity == notChecked {
			arity = arityOf(plist)
		}
		checkArity(arity, arityOf(plist), cl.Base())
	}

	return newBuilder().build(arity, clauses)
}

type builder struct {
	scrutinees []token.Token
}

func newBuilder() *builder {
	return &builder{}
}

// dispatch to Object or Lambda based on arity
func (b *builder) build(arity int, clauses []*ast.Clause) ast.Node {
	if arity == noArgs {
		return b.object(clauses)
	}
	return b.lambda(arity, clauses)
}

type UnsupportedPatternError struct {
	Clause *ast.Clause
}

func (e UnsupportedPatternError) Error() string {
	return utils.MsgAt(e.Clause.Base(), fmt.Sprintf("unsupported pattern %v", e.Clause))
}

func (b builder) object(clauses []*ast.Clause) ast.Node {
	// Pop the first accessor of each clause and group remaining clauses by the popped accessor.
	next := make(map[string][]*ast.Clause)
	for _, c := range clauses {
		plist := getPlist(c)
		if field, plist, ok := pop(plist); ok {
			next[field.String()] = append(
				next[field.String()],
				&ast.Clause{Patterns: []ast.Node{plist}, Exprs: c.Exprs})
		} else {
			panic(UnsupportedPatternError{Clause: c})
		}
	}

	fields := make([]*ast.Field, 0)

	// Generate each field's body expression
	// Object fields are generated in the dictionary order of field names.
	utils.OrderedFor(next, func(field string, cs []*ast.Clause) {
		// if any of cs has no accessors and has guards, generate Case expression

		/*
			case b.Scrutinee {
				caseClauses[0] (p0 -> e0)
				caseClauses[1] (p1 -> {restClauses})
				caseClauses[2] (p2 -> {restClauses})
			}
			(restClauses = caseClauses[1, 2])
		*/

		// new clauses for case expression in a field
		caseClauses := make([]*ast.Clause, len(cs))

		// for keeping order of clauses, use map[int]Clause instead of []Clause
		restClauses := make(map[int]*ast.Clause)
		for i, c := range cs {
			// if c has no accessors, generate pattern matching clause
			plist := getPlist(c)
			if len(plist.accessors) == 0 {
				caseClauses[i] = plistToClause(plist, c.Exprs...)
			} else {
				// otherwise, add to restClauses
				restClauses[i] = c
			}
		}

		// construct a clause for rest of (co) patterns
		restClausesList := make([]*ast.Clause, 0)
		utils.OrderedFor(restClauses, func(_ int, c *ast.Clause) {
			restClausesList = append(restClausesList, c)
		})

		for i, c := range restClauses {
			caseClauses[i] = plistToClause(getPlist(c), b.object(restClausesList))
		}

		fields = append(fields,
			&ast.Field{
				Name:  field,
				Exprs: newCase(b.scrutinees, caseClauses),
			})
		return
	})
	return &ast.Object{Fields: fields}
}

// Generate lambda and dispatch body expression to Object or Case based on existence of accessors.
func (b *builder) lambda(arity int, clauses []*ast.Clause) ast.Node {
	baseToken := clauses[0].Base()
	// Generate Scrutinees
	b.scrutinees = make([]token.Token, arity)
	for i := 0; i < arity; i++ {
		b.scrutinees[i] = newVar(fmt.Sprintf("x%d", i), baseToken)
	}

	// If any of clauses has accessors, body expression is Object.
	for _, c := range clauses {
		if len(getPlist(c).accessors) != 0 {
			return newLambda(b.scrutinees, b.object(clauses))
		}
	}

	// otherwise, body expression is Case.
	caseClauses := make([]*ast.Clause, 0)
	for _, c := range clauses {
		caseClauses = append(caseClauses, plistToClause(getPlist(c), c.Exprs...))
	}
	return newLambda(b.scrutinees, newCase(b.scrutinees, caseClauses)...)
}

// newLambda creates a new Lambda node with the given parameters and expressions.
// If there is only one parameter, return it without Paren pattern.
// Otherwise, parameters are wrapped by Paren pattern.
func newLambda(params []token.Token, exprs ...ast.Node) *ast.Lambda {
	if len(params) == 1 {
		return &ast.Lambda{Params: params, Exprs: exprs}
	}
	return &ast.Lambda{Params: params, Exprs: exprs}
}

// newVar creates a new Var node with the given name and a token.
func newVar(name string, base token.Token) token.Token {
	return token.Token{Kind: token.IDENT, Lexeme: name, Line: base.Line, Literal: nil}
}

// plistToClause creates a new Clause node with the given pattern and expressions.
// pattern must be a patternList.
func plistToClause(plist patternList, exprs ...ast.Node) *ast.Clause {
	return &ast.Clause{Patterns: plist.params, Exprs: exprs}
}

// newCase creates a new Case node with the given scrutinees and clauses.
// If there is no scrutinee, return Exprs of the first clause.
func newCase(scrs []token.Token, cs []*ast.Clause) []ast.Node {
	// if there is no scrutinee, return Exprs of the first clause
	// because case expression always matches the first clause.
	if len(scrs) == 0 {
		return cs[0].Exprs
	}
	vars := make([]ast.Node, len(scrs))
	for i, s := range scrs {
		vars[i] = &ast.Var{Name: s}
	}
	return []ast.Node{&ast.Case{Scrutinees: vars, Clauses: cs}}
}

type InvalidCallPatternError struct {
	Pattern ast.Node
}

func (e InvalidCallPatternError) Error() string {
	return utils.MsgAt(e.Pattern.Base(), fmt.Sprintf("invalid call pattern %v", e.Pattern))
}

// Collect all Access patterns recursively.
func accessors(p ast.Node) []token.Token {
	switch p := p.(type) {
	case *ast.Access:
		return append(accessors(p.Receiver), p.Name)
	default:
		return []token.Token{}
	}
}

// Get Args of Call{This, ...}
func params(p ast.Node) []ast.Node {
	switch p := p.(type) {
	case *ast.Access:
		return params(p.Receiver)
	case *ast.Call:
		if _, ok := p.Func.(*ast.This); !ok {
			panic(InvalidCallPatternError{Pattern: p})
		}
		return p.Args
	default:
		return nil
	}
}

type patternList struct {
	accessors []token.Token
	params    []ast.Node
}

func NewPatternList(clause *ast.Clause) patternList {
	if len(clause.Patterns) != 1 {
		panic("invalid pattern")
	}

	accessors := accessors(clause.Patterns[0])
	params := params(clause.Patterns[0])
	return patternList{accessors: accessors, params: params}
}

func NewPlistClause(clause *ast.Clause) *ast.Clause {
	return &ast.Clause{Patterns: []ast.Node{NewPatternList(clause)}, Exprs: clause.Exprs}
}

func getPlist(cl *ast.Clause) patternList {
	if len(cl.Patterns) != 1 {
		panic("invalid pattern")
	}
	return cl.Patterns[0].(patternList)
}

func (p patternList) Base() token.Token {
	if len(p.accessors) != 0 {
		return p.accessors[0]
	}
	if len(p.params) != 0 {
		return p.params[0].Base()
	}
	return token.Token{}
}

func (p patternList) String() string {
	accessors := make([]string, len(p.accessors))
	for i, a := range p.accessors {
		accessors[i] = a.String()
	}

	params := make([]string, len(p.params))
	for i, p := range p.params {
		params[i] = p.String()
	}

	return "[" + strings.Join(accessors, " ") + " | " + strings.Join(params, " ") + "]"
}

func (pl patternList) Plate(f func(ast.Node) ast.Node) ast.Node {
	for i, p := range pl.params {
		pl.params[i] = f(p)
	}
	return pl
}

var _ ast.Node = patternList{}

func arityOf(p patternList) int {
	if p.params == nil {
		return noArgs
	}
	return len(p.params)
}

// Split PatternList into the first accessor and the rest.
func pop(p patternList) (token.Token, patternList, bool) {
	if len(p.accessors) == 0 {
		return token.Token{}, p, false
	}
	a := p.accessors[0]
	p.accessors = p.accessors[1:]
	return a, p, true
}
