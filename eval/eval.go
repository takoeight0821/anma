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
	env map[id]Value
}

// NewEvaluator creates a new Evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{env: make(map[id]Value)}
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

func (ev *Evaluator) lookup(name token.Token) (Value, error) {
	id := id{name.Lexeme, name.Literal.(int)}
	if value, ok := ev.env[id]; ok {
		return value, nil
	}
	return nil, NotDefinedError{Name: name}
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

func (ev *Evaluator) Eval(node ast.Node) (Value, error) {
	switch n := node.(type) {
	case *ast.Var:
		return ev.lookup(n.Name)
	case *ast.Literal:
		return evalLiteral(n.Token)
	case *ast.Prim:
		values := make([]Value, len(n.Args))
		for i, arg := range n.Args {
			var err error
			values[i], err = ev.Eval(arg)
			if err != nil {
				return nil, err
			}
		}
		return evalPrim(n.Name, values)
	case *ast.Binary:
		op, err := ev.lookup(n.Op)
		if err != nil {
			return nil, err
		}

		lhs, err := ev.Eval(n.Left)
		if err != nil {
			return nil, err
		}

		rhs, err := ev.Eval(n.Right)
		if err != nil {
			return nil, err
		}

		return apply(n.Base(), op, lhs, rhs)
	case *ast.Lambda:
		return newFunction(ev, n.Pattern, n.Exprs)
	case *ast.VarDecl:
		value, err := ev.Eval(n.Expr)
		if err != nil {
			return nil, err
		}
		ev.define(n.Name, value)
		return nil, nil
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

func evalLiteral(token token.Token) (Value, error) {
	switch t := token.Literal.(type) {
	case int:
		return Int(t), nil
	case float64:
		return Float(t), nil
	default:
		return nil, UnexpectedLiteralError{Literal: token}
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

func evalPrim(name token.Token, args []Value) (Value, error) {
	switch name.Lexeme {
	case "add":
		if len(args) != 2 {
			return nil, ArityError{Base: name, Expected: 2, Args: args}
		}
		if lhs, ok := args[0].(Int); ok {
			if rhs, ok := args[1].(Int); ok {
				return Int(lhs + rhs), nil
			}
			return nil, TypeError{Base: name, expected: "Int", Value: args[1]}
		}
		return nil, TypeError{Base: name, expected: "Int", Value: args[0]}
	default:
		return nil, UnexpectedPrimError{Name: name}
	}
}

func newFunction(ev *Evaluator, pattern ast.Node, exprs []ast.Node) (Value, error) {
	// copy evaluator
	newEv := &Evaluator{env: make(map[id]Value)}
	for k, v := range ev.env {
		newEv.env[k] = v
	}

	return &Function{ev: newEv, pattern: pattern, exprs: exprs}, nil
}

type NotFunctionError struct {
	Base  token.Token
	Value Value
}

func (e NotFunctionError) Error() string {
	return utils.MsgAt(e.Base, fmt.Sprintf("%v is not a function", e.Value))
}

func apply(base token.Token, fun Value, args ...Value) (Value, error) {
	switch f := fun.(type) {
	case *Function:
		return f.ev.match(f.pattern, args, f.exprs)
	default:
		return nil, NotFunctionError{Base: base, Value: fun}
	}
}

func (ev *Evaluator) match(pattern ast.Node, args []Value, body []ast.Node) (Value, error) {
	panic("TODO")
}
