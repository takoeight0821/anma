package internal

import (
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

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

func NewPatternList(clause *ast.Clause) (PatternList, error) {
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
func (p patternList) Pop() (token.Token, PatternList, bool) {
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

type PatternList interface {
	ast.Node
	ArityOf() int
	Pop() (token.Token, PatternList, bool)
	HasAccess() bool
	Params() []ast.Node
}
