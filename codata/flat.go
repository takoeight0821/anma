package codata

import (
	"fmt"
	"sort"
	"strings"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/codata/internal"
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

type ArityError struct {
	Expected int // expected arity, or notChecked, or noArgs
}

func (e ArityError) Error() string {
	if e.Expected == internal.NotChecked {
		return "unreachable: arity is not checked"
	}
	return fmt.Sprintf("arity mismatch: expected %d arguments", e.Expected)
}

func checkArity(expected, actual int, where token.Token) error {
	if expected == internal.NotChecked {
		return nil
	}
	if expected != actual {
		return utils.ErrorAt{Where: where, Err: ArityError{Expected: expected}}
	}
	return nil
}

func flatCodata(c *ast.Codata) (ast.Node, error) {
	// Generate PatternList
	arity := internal.NotChecked
	clauses := make([]plistClause, len(c.Clauses))
	for i, cl := range c.Clauses {
		plist, err := internal.NewPatternList(cl)
		if err != nil {
			return nil, err
		}
		clauses[i] = plistClause{plist, cl.Exprs}
		if arity == internal.NotChecked {
			arity = plist.ArityOf()
		}
		err = checkArity(arity, plist.ArityOf(), cl.Base())
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
	if arity == internal.NoArgs {
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
	plist internal.PatternList
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
		if field, plist, ok := plist.Pop(); ok {
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
		plist internal.PatternList
	}, 0)
	// for each rest clause, the body expression is sum of expressions of all rest clauses.
	restClauses := make([]plistClause, 0)

	for i, c := range cs {
		// if c has no accessors, generate pattern matching clause
		if !c.plist.HasAccess() {
			caseClauses[i] = plistToClause(c.plist, c.exprs...)
		} else {
			// otherwise, add to restPatterns and restClauses
			restPatterns = append(restPatterns, struct {
				index int
				plist internal.PatternList
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
		if c.plist.HasAccess() {
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
func newLambda(params []token.Token, exprs ...ast.Node) *ast.Lambda {
	return &ast.Lambda{Params: params, Exprs: exprs}
}

// newVar creates a new Var node with the given name and a token.
func newVar(name string, base token.Token) token.Token {
	return token.Token{Kind: token.IDENT, Lexeme: name, Line: base.Line, Literal: nil}
}

// plistToClause creates a new Clause node with the given pattern and expressions.
// pattern must be a patternList.
func plistToClause(plist internal.PatternList, exprs ...ast.Node) *ast.Clause {
	return &ast.Clause{Patterns: plist.Params(), Exprs: exprs}
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

// observation is a linked list of patterns including #(this).
type observation struct {
	guard    []ast.Node // guard is patterns for branching.
	sequence []ast.Node // sequence is a sequence of patterns (destructors)
}

func (o observation) String() string {
	var b strings.Builder
	b.WriteString("[ ")
	for _, s := range o.sequence {
		b.WriteString(s.String())
		b.WriteString(" ")
	}
	b.WriteString("| ")
	for _, g := range o.guard {
		b.WriteString(g.String())
		b.WriteString(" ")
	}
	b.WriteString("]")
	return b.String()
}

// newObservation creates a new observation node with the given pattern.
func newObservation(p ast.Node) *observation {
	return &observation{
		guard:    extractGuard(p),
		sequence: extractSequence(p),
	}
}

// extractGuard extracts guard from the given pattern.
func extractGuard(p ast.Node) []ast.Node {
	switch p := p.(type) {
	case *ast.Access:
		return extractGuard(p.Receiver)
	case *ast.Call:
		if _, ok := p.Func.(*ast.This); ok {
			return p.Args
		}
	case *ast.This:
		return []ast.Node{}
	}
	panic(fmt.Sprintf("invalid pattern %v", p))
}

// extractSequence extracts sequence from the given pattern.
func extractSequence(p ast.Node) []ast.Node {
	switch p := p.(type) {
	case *ast.Access:
		current := &ast.Access{Receiver: &ast.This{Token: p.Receiver.Base()}, Name: p.Name}
		return append(extractSequence(p.Receiver), current)
	case *ast.Call:
		if _, ok := p.Func.(*ast.This); !ok {
			panic(fmt.Sprintf("invalid pattern %v", p))
		}
		return []ast.Node{p}
	case *ast.This:
		return []ast.Node{}
	default:
		panic(fmt.Sprintf("invalid pattern %v", p))
	}
}
