// Package eval is the simple evaluator for testing.
package eval

import (
	"fmt"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

// Evaluator evaluates the program.
type Evaluator struct {
	env     map[id]Value
	handler func(error)
}

// NewEvaluator creates a new Evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{env: make(map[id]Value), handler: func(err error) {
		panic(err)
	}}
}

func (ev *Evaluator) SetErrorHandler(handler func(error)) {
	ev.handler = handler
}

func (ev *Evaluator) Throw(err error) {
	ev.handler(err)
}

type id struct {
	name string
	uniq int
}

func (ev *Evaluator) define(name token.Token, value Value) {
	id := id{name.Lexeme, name.Literal.(int)}
	ev.env[id] = value
}

type NotDefinedError struct {
	Name token.Token
}

func (e NotDefinedError) Error() string {
	return utils.MsgAt(e.Name, fmt.Sprintf("%v is not defined", e.Name))
}

func (ev *Evaluator) lookup(name token.Token) Value {
	id := id{name.Lexeme, name.Literal.(int)}
	if value, ok := ev.env[id]; ok {
		return value
	}
	ev.Throw(NotDefinedError{Name: name})
	return nil
}

type Value interface {
	fmt.Stringer
}

type Int int

func (i Int) String() string {
	return fmt.Sprintf("%d", i)
}

type Float float64

func (f Float) String() string {
	return fmt.Sprintf("%f", f)
}

type Function struct {
	ev      *Evaluator
	pattern ast.Node
	exprs   []ast.Node
}

func (f *Function) String() string {
	return fmt.Sprintf("func(%v) {%v}", f.pattern, f.exprs)
}

type Tuple struct {
	Values []Value
}

func (t *Tuple) String() string {
	return fmt.Sprintf("(%v)", t.Values)
}

func (ev *Evaluator) Eval(node ast.Node) Value {
	switch n := node.(type) {
	case *ast.Var:
		return ev.lookup(n.Name)
	case *ast.Literal:
		return ev.evalLiteral(n.Token)
	case *ast.Prim:
		values := make([]Value, len(n.Args))
		for i, arg := range n.Args {
			values[i] = ev.Eval(arg)
		}
		return ev.evalPrim(n.Name, values)
	case *ast.Binary:
		op := ev.lookup(n.Op)
		lhs := ev.Eval(n.Left)
		rhs := ev.Eval(n.Right)
		return ev.apply(n.Base(), op, lhs, rhs)
	case *ast.Lambda:
		return newFunction(ev, n.Pattern, n.Exprs)
	case *ast.VarDecl:
		value := ev.Eval(n.Expr)
		ev.define(n.Name, value)
		return nil
	default:
		panic(fmt.Sprintf("not implemented %v", n))
	}
}

type UnexpectedLiteralError struct {
	Literal token.Token
}

func (e UnexpectedLiteralError) Error() string {
	return utils.MsgAt(e.Literal, fmt.Sprintf("unexpected literal: %v", e.Literal))
}

func (ev *Evaluator) evalLiteral(token token.Token) Value {
	switch t := token.Literal.(type) {
	case int:
		return Int(t)
	case float64:
		return Float(t)
	default:
		ev.Throw(UnexpectedLiteralError{Literal: token})
		return nil
	}
}

type ArityError struct {
	Base     token.Token
	Expected int
	Args     []Value
}

func (e ArityError) Error() string {
	return utils.MsgAt(e.Base, fmt.Sprintf("expected %d arguments, got %d", e.Expected, len(e.Args)))
}

type TypeError struct {
	Base     token.Token
	expected string
	Value    Value
}

func (e TypeError) Error() string {
	return utils.MsgAt(e.Base, fmt.Sprintf("expected %s, got %T", e.expected, e.Value))
}

type UnexpectedPrimError struct {
	Name token.Token
}

func (e UnexpectedPrimError) Error() string {
	return utils.MsgAt(e.Name, fmt.Sprintf("unexpected primitive: %v", e.Name))
}

func (ev *Evaluator) evalPrim(name token.Token, args []Value) Value {
	switch name.Lexeme {
	case "add":
		if len(args) != 2 {
			ev.Throw(ArityError{Base: name, Expected: 2, Args: args})
			return nil
		}
		if lhs, ok := args[0].(Int); ok {
			if rhs, ok := args[1].(Int); ok {
				return Int(lhs + rhs)
			}
			ev.Throw(TypeError{Base: name, expected: "Int", Value: args[1]})
			return nil
		}
		ev.Throw(TypeError{Base: name, expected: "Int", Value: args[0]})
		return nil
	default:
		ev.Throw(UnexpectedPrimError{Name: name})
		return nil
	}
}

func newFunction(ev *Evaluator, pattern ast.Node, exprs []ast.Node) Value {
	// copy evaluator
	newEv := &Evaluator{env: make(map[id]Value)}
	for k, v := range ev.env {
		newEv.env[k] = v
	}

	return &Function{ev: newEv, pattern: pattern, exprs: exprs}
}

type NotFunctionError struct {
	Base  token.Token
	Value Value
}

func (e NotFunctionError) Error() string {
	return utils.MsgAt(e.Base, fmt.Sprintf("%v is not a function", e.Value))
}

func (ev *Evaluator) apply(base token.Token, fun Value, args ...Value) Value {
	switch f := fun.(type) {
	case *Function:
		env, err := f.ev.match(f.pattern, args)
		if err != nil {
			ev.Throw(err)
			return nil
		}
		for k, v := range env {
			f.ev.define(k, v)
		}
		var value Value
		for _, expr := range f.exprs {
			value = f.ev.Eval(expr)
		}
		return value
	default:
		ev.Throw(NotFunctionError{Base: base, Value: fun})
		return nil
	}
}

func (ev *Evaluator) match(pattern ast.Node, args []Value) (map[token.Token]Value, error) {
	println("match", pattern.String(), args)
	switch p := pattern.(type) {
	case *ast.Tuple:
		if len(p.Elems) != len(args) {
			return nil, ArityError{Base: p.Base(), Expected: len(p.Elems), Args: args}
		}
		result := make(map[token.Token]Value)
		for i, elem := range p.Elems {
			env, err := ev.match(elem, []Value{args[i]})
			if err != nil {
				return nil, err
			}
			for k, v := range env {
				result[k] = v
			}
		}
		return result, nil
	case *ast.Var:
		if len(args) != 1 {
			return nil, ArityError{Base: p.Base(), Expected: 1, Args: args}
		}
		result := make(map[token.Token]Value)
		result[p.Name] = args[0]
		return result, nil
	}
	panic("not implemented")
}
