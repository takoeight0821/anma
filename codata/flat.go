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

func flat(node ast.Node) (ast.Node, error) {
	node, err := ast.Traverse(node, flatEach)
	if err != nil {
		return node, fmt.Errorf("flat %v: %w", node, err)
	}

	return node, nil
}

// flatEach converts Copatterns ([Access] and [This] in [Pattern]) into [Object] and [Lambda].
// If error occurred, return the original node and the error. Because ast.Traverse needs it.
func flatEach(node ast.Node, err error) (ast.Node, error) {
	// early return if error occurred
	if err != nil {
		return node, err
	}
	if c, ok := node.(*ast.Codata); ok {
		newNode, err := flatCodata(c)
		if err != nil {
			return node, err
		}

		return newNode, nil
	}

	return node, nil
}

type ArityError struct {
	Expected int // expected arity, or notChecked, or noArgs
}

func (e ArityError) Error() string {
	if e.Expected == NotChecked {
		return "unreachable: arity is not checked"
	}

	return fmt.Sprintf("arity mismatch: expected %d arguments", e.Expected)
}

func checkArity(expected, actual int, where token.Token) error {
	if expected == NotChecked {
		return nil
	}
	if expected != actual {
		return utils.PosError{Where: where, Err: ArityError{Expected: expected}}
	}

	return nil
}

func flatCodata(c *ast.Codata) (ast.Node, error) {
	// Generate PatternList
	arity := NotChecked
	clauses := make([]plistClause, len(c.Clauses))
	var observation Observation
	for i, clause := range c.Clauses {
		observation = merge(observation, ToObservation(clause.Pattern))
		plist, err := newPatternList(clause)
		if err != nil {
			return nil, err
		}
		clauses[i] = plistClause{plist, clause.Exprs}
		if arity == NotChecked {
			arity = plist.ArityOf()
		}
		err = checkArity(arity, plist.ArityOf(), clause.Base())
		if err != nil {
			return nil, err
		}
	}

	return build(arity, clauses)
}

// dispatch to Object or Lambda based on arity.
func build(arity int, clauses []plistClause) (ast.Node, error) {
	if arity == NoArgs {
		return object(nil, clauses)
	}

	return lambda(arity, clauses)
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
func groupClausesByAccessor(clauses []plistClause) (map[string][]plistClause, error) {
	next := make(map[string][]plistClause)
	for _, clause := range clauses {
		plist := clause.plist
		if field, plist, ok := plist.Pop(); ok {
			next[field.String()] = append(
				next[field.String()],
				plistClause{plist, clause.exprs})
		} else {
			return nil, utils.PosError{Where: plist.Base(), Err: UnsupportedPatternError{Clause: clause}}
		}
	}

	return next, nil
}

func fieldBody(scrutinees []token.Token, clauses []plistClause) ([]*ast.CaseClause, error) {
	// if any of cs has no accessors and has guards, generate Case expression

	// new clauses for case expression in a field
	caseClauses := make([]*ast.CaseClause, len(clauses))

	// restClauses are clauses which have unpopped accessors
	restPatterns := make([]struct {
		index int
		plist patternList
	}, 0)
	// for each rest clause, the body expression is sum of expressions of all rest clauses.
	restClauses := make([]plistClause, 0)

	for i, clause := range clauses {
		// if c has no accessors, generate pattern matching clause
		if !clause.plist.HasAccess() {
			caseClauses[i] = plistToClause(clause.plist, clause.exprs...)
		} else {
			// otherwise, add to restPatterns and restClauses
			restPatterns = append(restPatterns, struct {
				index int
				plist patternList
			}{i, clause.plist})

			restClauses = append(restClauses, clause)
		}
	}

	for _, pattern := range restPatterns {
		// for each rest clause, perform pattern matching ahead of time.
		obj, err := object(scrutinees, restClauses)
		if err != nil {
			return nil, err
		}
		caseClauses[pattern.index] = plistToClause(pattern.plist, obj)
	}

	return caseClauses, nil
}

func object(scrutinees []token.Token, clauses []plistClause) (ast.Node, error) {
	// Pop the first accessor of each clause and group remaining clauses by the popped accessor.
	next, err := groupClausesByAccessor(clauses)
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

		hasAccess := false
		for _, c := range cs {
			if c.plist.HasAccess() {
				hasAccess = true
				break
			}
		}

		if !hasAccess {
			body, err := fieldBody(scrutinees, cs)
			if err != nil {
				return nil, err
			}
			fields = append(fields,
				&ast.Field{
					Name:  field,
					Exprs: newCase(scrutinees, body),
				})
		} else {
			obj, err := object(scrutinees, cs)
			if err != nil {
				return nil, err
			}
			fields = append(fields,
				&ast.Field{
					Name:  field,
					Exprs: []ast.Node{obj},
				})
		}
	}

	return &ast.Object{Fields: fields}, nil
}

// Generate lambda and dispatch body expression to Object or Case based on existence of accessors.
func lambda(arity int, clauses []plistClause) (ast.Node, error) {
	baseToken := clauses[0].plist.Base()
	// Generate Scrutinees
	scrutinees := make([]token.Token, arity)
	for i := 0; i < arity; i++ {
		scrutinees[i] = newVar(fmt.Sprintf("x%d", i), baseToken)
	}

	// If any of clauses has accessors, body expression is Object.
	for _, c := range clauses {
		if c.plist.HasAccess() {
			obj, err := object(scrutinees, clauses)
			if err != nil {
				return nil, err
			}

			return newLambda(scrutinees, obj), nil
		}
	}

	// otherwise, body expression is Case.
	caseClauses := make([]*ast.CaseClause, 0)
	for _, c := range clauses {
		caseClauses = append(caseClauses, plistToClause(c.plist, c.exprs...))
	}

	return newLambda(scrutinees, newCase(scrutinees, caseClauses)...), nil
}

// newLambda creates a new Lambda node with the given parameters and expressions.
func newLambda(params []token.Token, exprs ...ast.Node) *ast.Lambda {
	return &ast.Lambda{Params: params, Exprs: exprs}
}

// newVar creates a new Var node with the given name and a token.
func newVar(name string, base token.Token) token.Token {
	return token.Token{Kind: token.IDENT, Lexeme: name, Line: base.Line, Literal: nil}
}

// plistToClause creates a new Clause node with the given pattern and expressions.
// pattern must be a patternList.
func plistToClause(plist patternList, exprs ...ast.Node) *ast.CaseClause {
	return &ast.CaseClause{Patterns: plist.Params(), Exprs: exprs}
}

// newCase creates a new Case node with the given scrutinees and clauses.
// If there is no scrutinee, return Exprs of the first clause.
func newCase(scrs []token.Token, clauses []*ast.CaseClause) []ast.Node {
	// if there is no scrutinee, return Exprs of the first clause
	// because case expression always matches the first clause.
	if len(scrs) == 0 {
		return clauses[0].Exprs
	}
	vars := make([]ast.Node, len(scrs))
	for i, s := range scrs {
		vars[i] = &ast.Var{Name: s}
	}

	return []ast.Node{&ast.Case{Scrutinees: vars, Clauses: clauses}}
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
func params(pattern ast.Node) ([]ast.Node, error) {
	switch pattern := pattern.(type) {
	case *ast.Access:
		return params(pattern.Receiver)
	case *ast.Call:
		if _, ok := pattern.Func.(*ast.This); !ok {
			return nil, utils.PosError{Where: pattern.Base(), Err: InvalidCallPatternError{Pattern: pattern}}
		}

		return pattern.Args, nil
	default:
		return nil, nil
	}
}

type patternList struct {
	accessors []token.Token
	params    []ast.Node
}

func newPatternList(clause *ast.CodataClause) (patternList, error) {
	accessors := accessors(clause.Pattern)
	params, err := params(clause.Pattern)
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

const (
	NotChecked = -2
	NoArgs     = -1
	ZeroArgs   = 0
)

func (p patternList) ArityOf() int {
	if p.params == nil {
		return NoArgs
	}

	return len(p.params)
}

// Split PatternList into the first accessor and the rest.
func (p patternList) Pop() (token.Token, patternList, bool) {
	if len(p.accessors) == 0 {
		return token.Token{}, p, false
	}

	return p.accessors[0], patternList{accessors: p.accessors[1:], params: p.params}, true
}

func (p patternList) HasAccess() bool {
	return len(p.accessors) != 0
}

func (p patternList) Params() []ast.Node {
	return p.params
}
