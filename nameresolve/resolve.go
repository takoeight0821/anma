package nameresolve

import (
	"fmt"
	"log"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

type Resolver struct {
	supply int
	env    *env
}

func NewResolver() *Resolver {
	return &Resolver{
		supply: 0,
		env:    newEnv(nil),
	}
}

type env struct {
	parent *env
	table  map[string]int
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
		err := r.registerTopLevel(node)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Resolver) Run(program []ast.Node) ([]ast.Node, error) {
	for i, node := range program {
		var err error
		program[i], err = r.solve(node)
		if err != nil {
			return nil, err
		}
	}
	return program, nil
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
func (r *Resolver) registerTopLevel(node ast.Node) error {
	switch n := node.(type) {
	case *ast.TypeDecl:
		_, err := r.assign(n.Def, allVariables)
		if err != nil {
			return err
		}
		_, err = r.assign(n.Type, ifNotDefined)
		return err
	case *ast.VarDecl:
		if _, ok := r.env.table[n.Name.Lexeme]; ok {
			return AlreadyDefinedError{Name: n.Name}
		}
		r.define(n.Name)
		return nil
	default:
		return nil
	}
}

// solve all variables in the node.
func (r *Resolver) solve(node ast.Node) (ast.Node, error) {
	switch n := node.(type) {
	case *ast.Var:
		var err error
		n.Name, err = r.env.lookup(n.Name)
		return n, err
	case *ast.Literal:
		return n, nil
	case *ast.Paren:
		for i, elem := range n.Elems {
			var err error
			n.Elems[i], err = r.solve(elem)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.Access:
		var err error
		n.Receiver, err = r.solve(n.Receiver)
		return n, err
	case *ast.Call:
		var err error
		n.Func, err = r.solve(n.Func)
		if err != nil {
			return n, err
		}
		for i, arg := range n.Args {
			n.Args[i], err = r.solve(arg)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.Prim:
		for i, arg := range n.Args {
			var err error
			n.Args[i], err = r.solve(arg)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.Binary:
		var err error
		n.Op, err = r.env.lookup(n.Op)
		if err != nil {
			return n, err
		}
		n.Left, err = r.solve(n.Left)
		if err != nil {
			return n, err
		}
		n.Right, err = r.solve(n.Right)
		return n, err
	case *ast.Assert:
		var err error
		n.Expr, err = r.solve(n.Expr)
		if err != nil {
			return n, err
		}
		n.Type, err = r.solve(n.Type)
		return n, err
	case *ast.Let:
		_, err := r.assign(n.Bind, allVariables)
		if err != nil {
			return n, err
		}
		n.Bind, err = r.solve(n.Bind)
		if err != nil {
			return n, err
		}
		n.Body, err = r.solve(n.Body)
		return n, err
	case *ast.Codata:
		log.Panicf("codata must be desugared before name resolution:\n%v", n)
		return n, nil
	case *ast.Clause:
		r.env = newEnv(r.env)
		defer func() { r.env = r.env.parent }()
		_, err := r.assign(n.Pattern, asPattern)
		if err != nil {
			return n, err
		}
		n.Pattern, err = r.solve(n.Pattern)
		if err != nil {
			return n, err
		}
		for i, expr := range n.Exprs {
			n.Exprs[i], err = r.solve(expr)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.Lambda:
		r.env = newEnv(r.env)
		defer func() { r.env = r.env.parent }()
		_, err := r.assign(n.Pattern, asPattern)
		if err != nil {
			return n, err
		}
		n.Pattern, err = r.solve(n.Pattern)
		if err != nil {
			return n, err
		}
		for i, expr := range n.Exprs {
			n.Exprs[i], err = r.solve(expr)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.Case:
		var err error
		n.Scrutinee, err = r.solve(n.Scrutinee)
		if err != nil {
			return n, err
		}
		for i, clause := range n.Clauses {
			newClause, err := r.solve(clause)
			n.Clauses[i] = newClause.(*ast.Clause)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.Object:
		for i, field := range n.Fields {
			newField, err := r.solve(field)
			n.Fields[i] = newField.(*ast.Field)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.Field:
		r.env = newEnv(r.env)
		defer func() { r.env = r.env.parent }()
		for i, expr := range n.Exprs {
			var err error
			n.Exprs[i], err = r.solve(expr)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.TypeDecl:
		var err error
		n.Def, err = r.solve(n.Def)
		if err != nil {
			return n, err
		}
		n.Type, err = r.solve(n.Type)
		return n, err
	case *ast.VarDecl:
		var err error
		n.Name, err = r.env.lookup(n.Name)
		if err != nil {
			return n, err
		}
		if n.Type != nil {
			n.Type, err = r.solve(n.Type)
			if err != nil {
				return n, err
			}
		}
		if n.Expr != nil {
			n.Expr, err = r.solve(n.Expr)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.InfixDecl:
		var err error
		n.Name, err = r.env.lookup(n.Name)
		return n, err
	case *ast.This:
		return n, nil
	default:
		log.Panicf("unexpected node: %v", n)
		return n, nil
	}
}

type mode func(*Resolver, ast.Node) ([]string, error)

type AlreadyDefinedError struct {
	Name token.Token
}

func (e AlreadyDefinedError) Error() string {
	return utils.MsgAt(e.Name, fmt.Sprintf("%s is already defined", e.Name.Pretty()))
}

// Define all variables in the node.
// If a variable is already defined in current scope, it returns an error.
func allVariables(r *Resolver, node ast.Node) ([]string, error) {
	var err error
	var defined []string
	ast.Transform(node, func(n ast.Node) ast.Node {
		if err != nil {
			return n
		}
		switch n := n.(type) {
		case *ast.Var:
			if _, ok := r.env.table[n.Name.Lexeme]; ok {
				err = AlreadyDefinedError{Name: n.Name}
			}
			r.define(n.Name)
			defined = append(defined, n.Name.Lexeme)
		}
		return n
	})
	return defined, err
}

// Define variables in the node if they are not defined.
func ifNotDefined(r *Resolver, node ast.Node) ([]string, error) {
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
	return defined, nil
}

type InvalidPatternError struct {
	Pattern ast.Node
}

func (e InvalidPatternError) Error() string {
	return utils.MsgAt(e.Pattern.Base(), fmt.Sprintf("invalid pattern %v", e.Pattern))
}

// Define variables in the node as pattern.
// If a variable appears as a function, it is ignored.
// If the node is not a pattern, it returns an error.
func asPattern(r *Resolver, node ast.Node) ([]string, error) {
	switch n := node.(type) {
	case *ast.Var:
		if _, ok := r.env.table[n.Name.Lexeme]; ok {
			return nil, AlreadyDefinedError{Name: n.Name}
		}
		r.define(n.Name)
		return []string{n.Name.Lexeme}, nil
	case *ast.Literal:
		return nil, nil
	case *ast.Paren:
		var defined []string
		for _, elem := range n.Elems {
			new, err := r.assign(elem, asPattern)
			if err != nil {
				return defined, err
			}
			defined = append(defined, new...)
		}
		return defined, nil
	case *ast.Access:
		return r.assign(n.Receiver, asPattern)
	case *ast.Call:
		var defined []string
		for _, arg := range n.Args {
			new, err := r.assign(arg, asPattern)
			if err != nil {
				return defined, err
			}
			defined = append(defined, new...)
		}
		return defined, nil
	default:
		return nil, InvalidPatternError{Pattern: node}
	}
}

// assign defines variables in the node.
// The mode function determines which variables are defined.
// Returns a list of defined variables.
func (r *Resolver) assign(node ast.Node, mode mode) ([]string, error) {
	return mode(r, node)
}
