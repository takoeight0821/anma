// Simple evaluator for testing.
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

func (ev *Evaluator) lookup(name token.Token) (Value, error) {
	id := id{name.Lexeme, name.Literal.(int)}
	value, ok := ev.env[id]
	if !ok {
		return nil, evalError(name, fmt.Sprintf("undefined variable: %v", name))
	}
	return value, nil
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
		return nil, evalError(node.Base(), fmt.Sprintf("unexpected node: %v", n))
	}
}

func evalLiteral(token token.Token) (Value, error) {
	switch t := token.Literal.(type) {
	case int:
		return Int(t), nil
	case float64:
		return Float(t), nil
	default:
		return nil, evalError(token, fmt.Sprintf("unexpected literal: %v", t))
	}
}

func evalPrim(name token.Token, args []Value) (Value, error) {
	switch name.Lexeme {
	case "add":
		if len(args) != 2 {
			return nil, evalError(name, fmt.Sprintf("expected 2 arguments, got %d", len(args)))
		}
		if lhs, ok := args[0].(Int); ok {
			if rhs, ok := args[1].(Int); ok {
				return Int(lhs + rhs), nil
			}
			return nil, evalError(name, fmt.Sprintf("expected Int, got %T", args[1]))
		}
		return nil, evalError(name, fmt.Sprintf("expected Int, got %T", args[0]))
	default:
		return nil, evalError(name, fmt.Sprintf("unexpected primitive: %v", name))
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

func evalError(node ast.Node, msg string) error {
	return utils.ErrorAt(node.Base(), "[eval] "+msg)
}
