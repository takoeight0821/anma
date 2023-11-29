package codata

import (
	"fmt"
	"sort"
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
		return n, fmt.Errorf("flat %v: %w", n, err)
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

const (
	notChecked = -2
	noArgs     = -1
	zeroArgs   = 0
)

type ArityError struct {
	Expected int // expected arity, or notChecked, or noArgs
}

func (e ArityError) Error() string {
	if e.Expected == notChecked {
		return "unreachable: arity is not checked"
	}
	return fmt.Sprintf("arity mismatch: expected %d arguments", e.Expected)
}

func checkArity(expected, actual int, where token.Token) error {
	if expected == notChecked {
		return nil
	}
	if expected != actual {
		return utils.ErrorAt{Where: where, Err: ArityError{Expected: expected}}
	}
	return nil
}

func flatCodata(c *ast.Codata) (ast.Node, error) {
	// Generate PatternList
	arity := notChecked
	clauses := make([]plistClause, len(c.Clauses))
	for i, cl := range c.Clauses {
		plist, err := newPatternList(cl)
		if err != nil {
			return nil, err
		}
		clauses[i] = plistClause{plist, cl.Exprs}
		if arity == notChecked {
			arity = arityOf(plist)
		}
		err = checkArity(arity, arityOf(plist), cl.Base())
		if err != nil {
			return nil, err
		}
	}

	return newBuilder().build(arity, clauses)
}

type builder struct {
	scrutinees []token.Token
}

func newBuilder() *builder {
	return &builder{}
}

// dispatch to Object or Lambda based on arity.
func (b *builder) build(arity int, clauses []plistClause) (ast.Node, error) {
	if arity == noArgs {
		return b.object(clauses)
	}
	return b.lambda(arity, clauses)
}

type UnsupportedPatternError struct {
	Clause plistClause
}

func (e UnsupportedPatternError) Error() string {
	return fmt.Sprintf("unsupported pattern %v", e.Clause)
}

type plistClause struct {
	plist patternList
	exprs []ast.Node
}

func (c plistClause) String() string {
	return fmt.Sprintf("%v -> %v", c.plist, c.exprs)
}

// Pop the first accessor of each clause and group remaining clauses by the popped accessor.
func (b builder) groupClausesByAccessor(clauses []plistClause) (map[string][]plistClause, error) {
	next := make(map[string][]plistClause)
	for _, c := range clauses {
		plist := c.plist
		if field, plist, ok := pop(plist); ok {
			next[field.String()] = append(
				next[field.String()],
				plistClause{plist, c.exprs})
		} else {
			return nil, utils.ErrorAt{Where: plist.Base(), Err: UnsupportedPatternError{Clause: c}}
		}
	}
	return next, nil
}

func (b builder) fieldBody(cs []plistClause) ([]*ast.Clause, error) {
	// if any of cs has no accessors and has guards, generate Case expression

	// new clauses for case expression in a field
	caseClauses := make([]*ast.Clause, len(cs))

	// restClauses are clauses which have unpopped accessors
	restPatterns := make([]struct {
		index int
		plist patternList
	}, 0)
	// for each rest clause, the body expression is sum of expressions of all rest clauses.
	restClauses := make([]plistClause, 0)

	for i, c := range cs {
		// if c has no accessors, generate pattern matching clause
		if len(c.plist.accessors) == 0 {
			caseClauses[i] = plistToClause(c.plist, c.exprs...)
		} else {
			// otherwise, add to restPatterns and restClauses
			restPatterns = append(restPatterns, struct {
				index int
				plist patternList
			}{i, c.plist})

			restClauses = append(restClauses, c)
		}
	}

	for _, p := range restPatterns {
		// for each rest clause, perform pattern matching ahead of time.
		obj, err := b.object(restClauses)
		if err != nil {
			return nil, err
		}
		caseClauses[p.index] = plistToClause(p.plist, obj)
	}

	return caseClauses, nil
}

func (b builder) object(clauses []plistClause) (ast.Node, error) {
	// Pop the first accessor of each clause and group remaining clauses by the popped accessor.
	next, err := b.groupClausesByAccessor(clauses)
	if err != nil {
		return nil, err
	}
	nextKeys := make([]string, 0, len(next))
	for k := range next {
		nextKeys = append(nextKeys, k)
	}
	sort.Strings(nextKeys)

	fields := make([]*ast.Field, 0)

	// Generate each field's body expression
	// Object fields are generated in the dictionary order of field names.
	for _, field := range nextKeys {
		cs := next[field]
		body, err := b.fieldBody(cs)
		if err != nil {
			return nil, err
		}
		fields = append(fields,
			&ast.Field{
				Name:  field,
				Exprs: newCase(b.scrutinees, body),
			})
	}
	return &ast.Object{Fields: fields}, nil
}

// Generate lambda and dispatch body expression to Object or Case based on existence of accessors.
func (b *builder) lambda(arity int, clauses []plistClause) (ast.Node, error) {
	baseToken := clauses[0].plist.Base()
	// Generate Scrutinees
	b.scrutinees = make([]token.Token, arity)
	for i := 0; i < arity; i++ {
		b.scrutinees[i] = newVar(fmt.Sprintf("x%d", i), baseToken)
	}

	// If any of clauses has accessors, body expression is Object.
	for _, c := range clauses {
		if len(c.plist.accessors) != 0 {
			obj, err := b.object(clauses)
			if err != nil {
				return nil, err
			}
			return newLambda(b.scrutinees, obj), nil
		}
	}

	// otherwise, body expression is Case.
	caseClauses := make([]*ast.Clause, 0)
	for _, c := range clauses {
		caseClauses = append(caseClauses, plistToClause(c.plist, c.exprs...))
	}
	return newLambda(b.scrutinees, newCase(b.scrutinees, caseClauses)...), nil
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
	return fmt.Sprintf("invalid call pattern %v", e.Pattern)
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

// Get Args of Call{This, ...}.
func params(p ast.Node) ([]ast.Node, error) {
	switch p := p.(type) {
	case *ast.Access:
		return params(p.Receiver)
	case *ast.Call:
		if _, ok := p.Func.(*ast.This); !ok {
			return nil, utils.ErrorAt{Where: p.Base(), Err: InvalidCallPatternError{Pattern: p}}
		}
		return p.Args, nil
	default:
		return nil, nil
	}
}

type patternList struct {
	accessors []token.Token
	params    []ast.Node
}

func newPatternList(clause *ast.Clause) (patternList, error) {
	if len(clause.Patterns) != 1 {
		panic("invalid pattern")
	}

	accessors := accessors(clause.Patterns[0])
	params, err := params(clause.Patterns[0])
	if err != nil {
		return patternList{}, err
	}
	return patternList{accessors: accessors, params: params}, err
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

func (p patternList) Plate(err error, f func(ast.Node, error) (ast.Node, error)) (ast.Node, error) {
	for i, param := range p.params {
		p.params[i], err = f(param, err)
	}
	return p, err
}

var _ ast.Node = patternList{}

func arityOf(p patternList) int {
	if p.params == nil {
		return noArgs
	}
	return len(p.params)
}

// Split PatternList into the first accessor and the rest.
//
//exhaustruct:ignore
func pop(p patternList) (token.Token, patternList, bool) {
	if len(p.accessors) == 0 {
		return token.Token{}, p, false
	}
	return p.accessors[0], patternList{accessors: p.accessors[1:], params: p.params}, true
}
