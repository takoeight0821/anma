// Package nameresolve resolves variable names and allocates unique numbers to them.
// It also checks if a variable is already defined in the same scope.
// All errors are accumulated and returned at the end of the process.
package nameresolve

import (
	"fmt"
	"log"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

// Resolver resolves variable names and allocates unique numbers to them.
type Resolver struct {
	supply int  // Supply of unique numbers.
	env    *env // Current environment.
}

func NewResolver() *Resolver {
	return &Resolver{
		supply: 0,
		env:    newEnv(nil),
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
			return program, err
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
	return fmt.Sprintf("%s is not defined", e.Name.String())
}

func (e *env) lookup(name token.Token) (token.Token, error) {
	if uniq, ok := e.table[name.Lexeme]; ok {
		return token.Token{Kind: name.Kind, Lexeme: name.Lexeme, Line: name.Line, Literal: uniq}, nil
	}

	if e.parent != nil {
		return e.parent.lookup(name)
	}

	return name, utils.ErrorAt{Where: name, Err: NotDefinedError{Name: name}}
}

// Define all top-level variables in the node.
func (r *Resolver) registerTopLevel(node ast.Node) error {
	switch n := node.(type) {
	case *ast.TypeDecl:
		_, err := r.assign(n.Def, allVariables)
		if err != nil {
			return err
		}
		for _, typ := range n.Types {
			_, err := r.assign(typ, ifNotDefined)
			if err != nil {
				return err
			}
		}
	case *ast.VarDecl:
		if _, ok := r.env.table[n.Name.Lexeme]; ok {
			return utils.ErrorAt{Where: n.Base(), Err: AlreadyDefinedError{Name: n.Name}}
		}
		r.define(n.Name)
	}
	return nil
}

func (r *Resolver) solveToken(t token.Token) (token.Token, error) {
	return r.env.lookup(t)
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
		var err error
		n.Expr, err = r.solve(n.Expr)
		return n, err
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
		var err error
		for i, arg := range n.Args {
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
		if err != nil {
			return n, err
		}
		return n, nil
	case *ast.Assert:
		var err error
		n.Expr, err = r.solve(n.Expr)
		if err != nil {
			return n, err
		}
		n.Type, err = r.solve(n.Type)
		if err != nil {
			return n, err
		}
		return n, nil
	case *ast.Let:
		_, err := r.assign(n.Bind, overwrite)
		if err != nil {
			return n, err
		}
		n.Bind, err = r.solve(n.Bind)
		if err != nil {
			return n, err
		}
		n.Body, err = r.solve(n.Body)
		if err != nil {
			return n, err
		}
		return n, nil
	case *ast.Codata:
		log.Panicf("codata must be desugared before name resolution:\n%v", n)
		return n, nil
	case *ast.Clause:
		r.env = newEnv(r.env)
		defer func() { r.env = r.env.parent }()
		for i, pattern := range n.Patterns {
			_, err := r.assign(pattern, asPattern)
			if err != nil {
				return n, err
			}
			n.Patterns[i], err = r.solve(pattern)
			if err != nil {
				return n, err
			}
		}
		for i, expr := range n.Exprs {
			var err error
			n.Exprs[i], err = r.solve(expr)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.Lambda:
		r.env = newEnv(r.env)
		defer func() { r.env = r.env.parent }()
		for i, param := range n.Params {
			_, err := r.assignToken(param, asPattern)
			if err != nil {
				return n, err
			}
			n.Params[i], err = r.solveToken(param)
			if err != nil {
				return n, err
			}
		}
		for i, expr := range n.Exprs {
			var err error
			n.Exprs[i], err = r.solve(expr)
			if err != nil {
				return n, err
			}
		}
		return n, nil
	case *ast.Case:
		for i, scr := range n.Scrutinees {
			var err error
			n.Scrutinees[i], err = r.solve(scr)
			if err != nil {
				return n, err
			}
		}
		for i, clause := range n.Clauses {
			clause, err := r.solve(clause)
			if err != nil {
				return n, err
			}
			n.Clauses[i] = clause.(*ast.Clause)
		}
		return n, nil
	case *ast.Object:
		for i, field := range n.Fields {
			field, err := r.solve(field)
			if err != nil {
				return n, err
			}
			n.Fields[i] = field.(*ast.Field)
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
		for i, typ := range n.Types {
			n.Types[i], err = r.solve(typ)
			if err != nil {
				return n, err
			}
		}
		return n, nil
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
		if err != nil {
			return n, err
		}
		return n, nil
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
	return fmt.Sprintf("%s is already defined", e.Name.String())
}

// allVariables define all variables in the node.
// If a variable is already defined in current scope, it is an error.
func allVariables(r *Resolver, node ast.Node) ([]string, error) {
	var defined []string
	var err error
	ast.Traverse(node, func(n ast.Node) ast.Node {
		if err != nil {
			return n
		}
		switch n := n.(type) {
		case *ast.Var:
			if _, ok := r.env.table[n.Name.Lexeme]; ok {
				err = utils.ErrorAt{Where: n.Base(), Err: AlreadyDefinedError{Name: n.Name}}
				return n
			}
			r.define(n.Name)
			defined = append(defined, n.Name.Lexeme)
			return n
		default:
			return n
		}
	})
	return defined, err
}

// overwrite defines all variables in the node.
// If a variable is already defined in current scope, it is overwritten.
func overwrite(r *Resolver, node ast.Node) ([]string, error) {
	var defined []string
	ast.Traverse(node, func(n ast.Node) ast.Node {
		switch n := n.(type) {
		case *ast.Var:
			r.define(n.Name)
			defined = append(defined, n.Name.Lexeme)
			return n
		default:
			return n
		}
	})
	return defined, nil
}

// ifNotDefined define variables in the node if they are not defined.
func ifNotDefined(r *Resolver, node ast.Node) ([]string, error) {
	var defined []string
	ast.Traverse(node, func(n ast.Node) ast.Node {
		switch n := n.(type) {
		case *ast.Var:
			if _, err := r.env.lookup(n.Name); err != nil {
				r.define(n.Name)
				defined = append(defined, n.Name.Lexeme)
			}
			return n
		default:
			return n
		}
	})
	return defined, nil
}

type InvalidPatternError struct {
	Pattern ast.Node
}

func (e InvalidPatternError) Error() string {
	return fmt.Sprintf("invalid pattern %v", e.Pattern)
}

// Define variables in the node as pattern.
// If a variable appears as a function, it is ignored.
func asPattern(r *Resolver, node ast.Node) ([]string, error) {
	switch n := node.(type) {
	case *ast.Var:
		if _, ok := r.env.table[n.Name.Lexeme]; ok {
			return nil, utils.ErrorAt{Where: n.Base(), Err: AlreadyDefinedError{Name: n.Name}}
		}
		r.define(n.Name)
		return []string{n.Name.Lexeme}, nil
	case *ast.Literal:
		return nil, nil
	case *ast.Paren:
		return r.assign(n.Expr, asPattern)
	case *ast.Access:
		return r.assign(n.Receiver, asPattern)
	case *ast.Call:
		var defined []string
		for _, arg := range n.Args {
			newDefs, err := r.assign(arg, asPattern)
			if err != nil {
				return nil, err
			}
			defined = append(defined, newDefs...)
		}
		return defined, nil
	default:
		return nil, utils.ErrorAt{Where: node.Base(), Err: InvalidPatternError{Pattern: node}}
	}
}

// assign defines variables in the node.
// The mode function determines which variables are defined.
// Returns a list of defined variables.
func (r *Resolver) assign(node ast.Node, mode mode) ([]string, error) {
	return mode(r, node)
}

func (r *Resolver) assignToken(t token.Token, mode mode) ([]string, error) {
	return r.assign(&ast.Var{Name: t}, mode)
}
