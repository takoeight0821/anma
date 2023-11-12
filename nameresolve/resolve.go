// Package nameresolve resolves variable names and allocates unique numbers to them.
// It also checks if a variable is already defined in the same scope.
// All errors are accumulated and returned at the end of the process.
package nameresolve

import (
	"errors"
	"fmt"
	"log"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

// Resolver resolves variable names and allocates unique numbers to them.
type Resolver struct {
	supply int   // Supply of unique numbers.
	env    *env  // Current environment.
	err    error // Error accumulator. nil if no error. Use addError() to add an error.
}

func NewResolver() *Resolver {
	return &Resolver{
		supply: 0,
		env:    newEnv(nil),
		err:    nil,
	}
}

type env struct {
	parent *env           // Parent environment.
	table  map[string]int // Variable name -> unique number.
}

func newEnv(parent *env) *env {
	return &env{
		parent: parent,
		table:  make(map[string]int),
	}
}

func (r *Resolver) Name() string {
	return "nameresolve.Resolver"
}

func (r *Resolver) Init(program []ast.Node) error {
	// Register top-level declarations.
	for _, node := range program {
		r.registerTopLevel(node)
	}
	return r.err
}

func (r *Resolver) Run(program []ast.Node) ([]ast.Node, error) {
	for i, node := range program {
		program[i] = r.solve(node)
	}
	return program, r.err
}

func (r *Resolver) addError(err error) {
	r.err = errors.Join(r.err, err)
}

func (r *Resolver) resetError() {
	r.err = nil
}

func (r *Resolver) define(name token.Token) {
	r.env.table[name.Lexeme] = r.supply
	r.supply++
}

type NotDefinedError struct {
	Name token.Token
}

func (e NotDefinedError) Error() string {
	return utils.MsgAt(e.Name, fmt.Sprintf("%s is not defined", e.Name.Pretty()))
}

func (e *env) lookup(name token.Token) (token.Token, error) {
	if uniq, ok := e.table[name.Lexeme]; ok {
		return token.Token{Kind: name.Kind, Lexeme: name.Lexeme, Line: name.Line, Literal: uniq}, nil
	}

	if e.parent != nil {
		return e.parent.lookup(name)
	}

	return name, NotDefinedError{Name: name}
}

// Define all top-level variables in the node.
func (r *Resolver) registerTopLevel(node ast.Node) {
	switch n := node.(type) {
	case *ast.TypeDecl:
		r.assign(n.Def, allVariables)
		r.assign(n.Type, ifNotDefined)
	case *ast.VarDecl:
		if _, ok := r.env.table[n.Name.Lexeme]; ok {
			r.addError(AlreadyDefinedError{Name: n.Name})
		}
		r.define(n.Name)
	}
}

// solve all variables in the node.
func (r *Resolver) solve(node ast.Node) ast.Node {
	switch n := node.(type) {
	case *ast.Var:
		var err error
		n.Name, err = r.env.lookup(n.Name)
		r.addError(err)
		return n
	case *ast.Literal:
		return n
	case *ast.Paren:
		for i, elem := range n.Elems {
			n.Elems[i] = r.solve(elem)
		}
		return n
	case *ast.Access:
		n.Receiver = r.solve(n.Receiver)
		return n
	case *ast.Call:
		n.Func = r.solve(n.Func)
		for i, arg := range n.Args {
			n.Args[i] = r.solve(arg)
		}
		return n
	case *ast.Prim:
		for i, arg := range n.Args {
			n.Args[i] = r.solve(arg)
		}
		return n
	case *ast.Binary:
		var err error
		n.Op, err = r.env.lookup(n.Op)
		if err != nil {
			r.addError(err)
		}
		n.Left = r.solve(n.Left)
		n.Right = r.solve(n.Right)
		return n
	case *ast.Assert:
		n.Expr = r.solve(n.Expr)
		n.Type = r.solve(n.Type)
		return n
	case *ast.Let:
		r.assign(n.Bind, overwrite)
		n.Bind = r.solve(n.Bind)
		n.Body = r.solve(n.Body)
		return n
	case *ast.Codata:
		log.Panicf("codata must be desugared before name resolution:\n%v", n)
		return n
	case *ast.Clause:
		r.env = newEnv(r.env)
		defer func() { r.env = r.env.parent }()
		r.assign(n.Pattern, asPattern)
		n.Pattern = r.solve(n.Pattern)
		for i, expr := range n.Exprs {
			n.Exprs[i] = r.solve(expr)
		}
		return n
	case *ast.Lambda:
		r.env = newEnv(r.env)
		defer func() { r.env = r.env.parent }()
		r.assign(n.Pattern, asPattern)
		n.Pattern = r.solve(n.Pattern)
		for i, expr := range n.Exprs {
			n.Exprs[i] = r.solve(expr)
		}
		return n
	case *ast.Case:
		n.Scrutinee = r.solve(n.Scrutinee)
		for i, clause := range n.Clauses {
			n.Clauses[i] = r.solve(clause).(*ast.Clause)
		}
		return n
	case *ast.Object:
		for i, field := range n.Fields {
			n.Fields[i] = r.solve(field).(*ast.Field)
		}
		return n
	case *ast.Field:
		r.env = newEnv(r.env)
		defer func() { r.env = r.env.parent }()
		for i, expr := range n.Exprs {
			n.Exprs[i] = r.solve(expr)
		}
		return n
	case *ast.TypeDecl:
		n.Def = r.solve(n.Def)
		n.Type = r.solve(n.Type)
		return n
	case *ast.VarDecl:
		var err error
		n.Name, err = r.env.lookup(n.Name)
		r.addError(err)
		if n.Type != nil {
			n.Type = r.solve(n.Type)
		}
		if n.Expr != nil {
			n.Expr = r.solve(n.Expr)
		}
		return n
	case *ast.InfixDecl:
		var err error
		n.Name, err = r.env.lookup(n.Name)
		r.addError(err)
		return n
	case *ast.This:
		return n
	default:
		log.Panicf("unexpected node: %v", n)
		return n
	}
}

type mode func(*Resolver, ast.Node) []string

type AlreadyDefinedError struct {
	Name token.Token
}

func (e AlreadyDefinedError) Error() string {
	return utils.MsgAt(e.Name, fmt.Sprintf("%s is already defined", e.Name.Pretty()))
}

// allVariables define all variables in the node.
// If a variable is already defined in current scope, it is an error.
func allVariables(r *Resolver, node ast.Node) []string {
	var defined []string
	ast.Transform(node, func(n ast.Node) ast.Node {
		switch n := n.(type) {
		case *ast.Var:
			if _, ok := r.env.table[n.Name.Lexeme]; ok {
				r.addError(AlreadyDefinedError{Name: n.Name})
			}
			r.define(n.Name)
			defined = append(defined, n.Name.Lexeme)
		}
		return n
	})
	return defined
}

// overwrite defines all variables in the node.
// If a variable is already defined in current scope, it is overwritten.
func overwrite(r *Resolver, node ast.Node) []string {
	var defined []string
	ast.Transform(node, func(n ast.Node) ast.Node {
		switch n := n.(type) {
		case *ast.Var:
			r.define(n.Name)
			defined = append(defined, n.Name.Lexeme)
		}
		return n
	})
	return defined
}

// ifNotDefined define variables in the node if they are not defined.
func ifNotDefined(r *Resolver, node ast.Node) []string {
	var defined []string
	ast.Transform(node, func(n ast.Node) ast.Node {
		switch n := n.(type) {
		case *ast.Var:
			if _, err := r.env.lookup(n.Name); err != nil {
				r.define(n.Name)
				defined = append(defined, n.Name.Lexeme)
			}
		}
		return n
	})
	return defined
}

type InvalidPatternError struct {
	Pattern ast.Node
}

func (e InvalidPatternError) Error() string {
	return utils.MsgAt(e.Pattern.Base(), fmt.Sprintf("invalid pattern %v", e.Pattern))
}

// Define variables in the node as pattern.
// If a variable appears as a function, it is ignored.
func asPattern(r *Resolver, node ast.Node) []string {
	switch n := node.(type) {
	case *ast.Var:
		if _, ok := r.env.table[n.Name.Lexeme]; ok {
			r.addError(AlreadyDefinedError{Name: n.Name})
		}
		r.define(n.Name)
		return []string{n.Name.Lexeme}
	case *ast.Literal:
		return nil
	case *ast.Paren:
		var defined []string
		for _, elem := range n.Elems {
			new := r.assign(elem, asPattern)
			defined = append(defined, new...)
		}
		return defined
	case *ast.Access:
		return r.assign(n.Receiver, asPattern)
	case *ast.Call:
		var defined []string
		for _, arg := range n.Args {
			new := r.assign(arg, asPattern)
			defined = append(defined, new...)
		}
		return defined
	default:
		r.addError(InvalidPatternError{Pattern: node})
		return nil
	}
}

// assign defines variables in the node.
// The mode function determines which variables are defined.
// Returns a list of defined variables.
func (r *Resolver) assign(node ast.Node, mode mode) []string {
	return mode(r, node)
}
