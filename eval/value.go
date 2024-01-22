package eval

import (
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
)

type Value interface {
	fmt.Stringer
	match(pattern ast.Node) (map[Name]Value, bool)
}

type Callable interface {
	Apply(where token.Token, args ...Value) (Value, error)
}

type Unit struct{}

func (u Unit) String() string {
	return "<unit>"
}

func (u Unit) match(pattern ast.Node) (map[Name]Value, bool) {
	switch pattern := pattern.(type) {
	case *ast.Var:
		return map[Name]Value{tokenToName(pattern.Name): u}, true
	default:
		return nil, false
	}
}

var _ Value = Unit{}

type Int int

func (i Int) String() string {
	return fmt.Sprintf("%d", i)
}

func (i Int) match(pattern ast.Node) (map[Name]Value, bool) {
	switch pattern := pattern.(type) {
	case *ast.Var:
		return map[Name]Value{tokenToName(pattern.Name): i}, true
	case *ast.Literal:
		if pattern.Kind == token.INTEGER && pattern.Literal == i {
			return map[Name]Value{}, true
		}
	}

	return nil, false
}

var _ Value = Int(0)

type String string

func (s String) String() string {
	return fmt.Sprintf("%q", string(s))
}

func (s String) match(pattern ast.Node) (map[Name]Value, bool) {
	switch pattern := pattern.(type) {
	case *ast.Var:
		return map[Name]Value{tokenToName(pattern.Name): s}, true
	case *ast.Literal:
		if pattern.Kind == token.STRING && pattern.Literal == s {
			return map[Name]Value{}, true
		}
	}

	return nil, false
}

var _ Value = String("")

// Function represents a closure value.
type Function struct {
	Evaluator
	Params []Name
	Body   []ast.Node
}

func (f Function) String() string {
	var builder strings.Builder
	builder.WriteString("<function")
	for _, param := range f.Params {
		builder.WriteString(" ")
		builder.WriteString(string(param))
	}
	builder.WriteString(">")

	return builder.String()
}

func (f Function) match(pattern ast.Node) (map[Name]Value, bool) {
	switch pattern := pattern.(type) {
	case *ast.Var:
		return map[Name]Value{tokenToName(pattern.Name): f}, true
	default:
		return nil, false
	}
}

func (f Function) Apply(where token.Token, args ...Value) (Value, error) {
	if len(f.Params) != len(args) {
		return nil, errorAt(where, InvalidArgumentCountError{Expected: len(f.Params), Actual: len(args)})
	}
	f.evEnv = newEvEnv(f.evEnv)
	for i, param := range f.Params {
		f.evEnv.set(param, args[i])
	}

	var ret Value
	for _, node := range f.Body {
		var err error
		ret, err = f.Eval(node)
		if err != nil {
			return nil, err
		}
	}
	f.evEnv = f.evEnv.parent

	return ret, nil
}

var (
	_ Value    = Function{}
	_ Callable = Function{}
)

// Thunk represents a thunk value.
// It is used to delay the evaluation of object fields.
type Thunk struct {
	Evaluator
	Body []ast.Node
}

func (t Thunk) String() string {
	return "<thunk>"
}

func (t Thunk) match(pattern ast.Node) (map[Name]Value, bool) {
	switch pattern := pattern.(type) {
	case *ast.Var:
		return map[Name]Value{tokenToName(pattern.Name): t}, true
	default:
		return nil, false
	}
}

func runThunk(value Value) (Value, error) {
	switch value := value.(type) {
	case Thunk:
		var ret Value
		for _, node := range value.Body {
			var err error
			ret, err = value.Eval(node)
			if err != nil {
				return nil, err
			}
		}
		if _, ok := ret.(Thunk); ok {
			panic("unreachable: thunk cannot return thunk")
		}

		return ret, nil
	default:
		return value, nil
	}
}

var _ Value = Thunk{}

// Object represents an object value.
type Object struct {
	Fields map[string]Value
}

func (o Object) String() string {
	return "<object>"
}

func (o Object) match(pattern ast.Node) (map[Name]Value, bool) {
	switch pattern := pattern.(type) {
	case *ast.Var:
		return map[Name]Value{tokenToName(pattern.Name): o}, true
	default:
		return nil, false
	}
}

var _ Value = Object{}

// Data represents a algebraic data type value.
type Data struct {
	Tag   Name
	Elems []Value
}

func (d Data) String() string {
	var builder strings.Builder
	builder.WriteString(string(d.Tag))
	builder.WriteString("(")
	for i, elem := range d.Elems {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(elem.String())
	}
	builder.WriteString(")")

	return builder.String()
}

func (d Data) match(pattern ast.Node) (map[Name]Value, bool) {
	switch pattern := pattern.(type) {
	case *ast.Var:
		return map[Name]Value{tokenToName(pattern.Name): d}, true
	case *ast.Call:
		switch fn := pattern.Func.(type) {
		case *ast.Var:
			if tokenToName(fn.Name) != d.Tag {
				return nil, false
			}
			matches := make(map[Name]Value)
			for i, elem := range d.Elems {
				if i >= len(pattern.Args) {
					return nil, false
				}
				m, ok := elem.match(pattern.Args[i])
				if !ok {
					return nil, false
				}
				for k, v := range m {
					matches[k] = v
				}
			}

			return matches, true
		default:
			panic(fmt.Sprintf("unreachable: %s", pattern.Func))
		}
	}

	return nil, false
}

var _ Value = Data{}

type Constructor struct {
	Evaluator
	Tag    Name
	Params int
}

func (c Constructor) String() string {
	return fmt.Sprintf("%s/%d", c.Tag, c.Params)
}

func (c Constructor) match(pattern ast.Node) (map[Name]Value, bool) {
	switch pattern := pattern.(type) {
	case *ast.Var:
		return map[Name]Value{tokenToName(pattern.Name): c}, true
	default:
		return nil, false
	}
}

func (c Constructor) Apply(where token.Token, args ...Value) (Value, error) {
	if len(args) != c.Params {
		return nil, errorAt(where, InvalidArgumentCountError{Expected: c.Params, Actual: len(args)})
	}

	return Data{Tag: c.Tag, Elems: args}, nil
}

var (
	_ Value    = Constructor{}
	_ Callable = Constructor{}
)
