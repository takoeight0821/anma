package main

import "fmt"

// EvalCtx stores variable-to-value table.
// In naive assumption, rename.go resolves all scope problems. So EvalCtx can be a single big map.
// But thinking about memory usage, we have to delete unused entries of the table.
// This is why EvalCtx is a chain of maps.
// Scope rule for EvalCtx may different from rename.go.
type EvalCtx struct {
	env    map[id]value
	parent *EvalCtx
}

func NewEvalCtx() *EvalCtx {
	return &EvalCtx{env: make(map[id]value), parent: nil}
}

func (e *EvalCtx) bind(t Token, v value) {
	x := newId(t)
	if _, ok := e.env[x]; ok {
		panic(errorAt(t, fmt.Sprintf("%v is already defined in this scope", t)))
	}
	e.env[x] = v
}

func (e *EvalCtx) lookup(t Token) value {
	if e == nil {
		panic(errorAt(t, fmt.Sprintf("%v is not defined", t)))
	}

	x := newId(t)
	if v, ok := e.env[x]; ok {
		return v
	}

	return e.parent.lookup(t)
}

type id struct {
	name string
	uniq int
}

func newId(t Token) id {
	if v, ok := t.Literal.(int); ok {
		return id{name: t.Lexeme, uniq: v}
	}
	panic(errorAt(t, fmt.Sprintf("%#v is invalid", t)))
}

type value any

var (
	_ value = int(0)
	_ value = float64(0.0)
	_ value = []value{}
	_ value = make(map[string]value)
	_ value = closure{}
)

type closure struct {
	env    *Env
	params []id
	body   []Node
}

func (closure) String() string {
	return "<function>"
}

func Eval(ctx *EvalCtx, node Node) value {
	switch n := node.(type) {
	case *Var:
		return ctx.lookup(n.Name)
	case *Literal:
		return n.Literal
	case *Paren:
		tuple := make([]value, len(n.Elems))
		for i, elem := range n.Elems {
			tuple[i] = Eval(ctx, elem)
		}
		return tuple
	case *Access:
		v := Eval(ctx, n.Receiver)
		return evalAccess(v, n.Name)
	case *Call:
		fun := Eval(ctx, n.Func)
		args := make([]value, len(n.Args))
		for i, arg := range n.Args {
			args[i] = Eval(ctx, arg)
		}
		return evalCall(fun, args)
	default:
		panic(errorAt(n.Base(), fmt.Sprintf("not implemented %v", n)))
	}
}

func evalAccess(v value, n Token) value {
	panic("TODO")
}

func evalCall(fun value, args []value) value {
	panic("TODO")
}
