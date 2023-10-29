package main

import "fmt"

// Evaluator stores variable-to-value table.
// In naive assumption, rename.go resolves all scope problems. So Evaluator can be a single big map.
// But thinking about memory usage, we have to delete unused entries of the table.
// This is why Evaluator is a chain of maps.
// Scope rule for Evaluator may different from rename.go.
type Evaluator struct {
	env    map[id]value
	parent *Evaluator
}

func NewEvaluator() *Evaluator {
	return &Evaluator{env: make(map[id]value), parent: nil}
}

func (e *Evaluator) Init(program []Node) error {
	return nil
}

func (e *Evaluator) Run(program []Node) ([]Node, error) {
	for _, node := range program {
		v, err := eval(e, node)
		if err != nil {
			return program, err
		}
		fmt.Println(v)
	}
	return program, nil
}

func (e *Evaluator) bind(t Token, v value) error {
	x := newId(t)
	if _, ok := e.env[x]; ok {
		return errorAt(t, fmt.Sprintf("%v is already defined in this scope", t))
	}
	e.env[x] = v
	return nil
}

func (e *Evaluator) lookup(t Token) (value, error) {
	if e == nil {
		return nil, errorAt(t, fmt.Sprintf("%v is not defined", t))
	}

	x := newId(t)
	if v, ok := e.env[x]; ok {
		return v, nil
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
	env    *rnEnv
	params []id
	body   []Node
}

func (closure) String() string {
	return "<function>"
}

func eval(ctx *Evaluator, node Node) (value, error) {
	switch n := node.(type) {
	case *Var:
		return ctx.lookup(n.Name)
	case *Literal:
		return n.Literal, nil
	case *Paren:
		tuple := make([]value, len(n.Elems))
		for i, elem := range n.Elems {
			var err error
			tuple[i], err = eval(ctx, elem)
			if err != nil {
				return nil, err
			}
		}
		return tuple, nil
	case *Access:
		v, err := eval(ctx, n.Receiver)
		if err != nil {
			return nil, err
		}
		return evalAccess(v, n.Name)
	case *Call:
		fun, err := eval(ctx, n.Func)
		if err != nil {
			return nil, err
		}
		args := make([]value, len(n.Args))
		for i, arg := range n.Args {
			args[i], err = eval(ctx, arg)
			if err != nil {
				return nil, err
			}
		}
		return evalCall(fun, args)
	default:
		return nil, errorAt(n.Base(), fmt.Sprintf("not implemented %v", n))
	}
}

func evalAccess(v value, n Token) (value, error) {
	panic("TODO")
}

func evalCall(fun value, args []value) (value, error) {
	panic("TODO")
}