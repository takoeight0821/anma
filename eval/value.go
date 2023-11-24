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
	Apply(token.Token, ...Value) Value
}

type Unit struct{}

func (u Unit) String() string {
	return "<unit>"
}

func (u Unit) match(pattern ast.Node) (map[Name]Value, bool) {
	switch pattern := pattern.(type) {
	case *ast.Var:
		return map[Name]Value{tokenToName(pattern.Name): u}, true
	}
	return nil, false
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
	var b strings.Builder
	b.WriteString("<function")
	for _, param := range f.Params {
		b.WriteString(" ")
		b.WriteString(string(param))
	}
	b.WriteString(">")
	return b.String()
}

func (f Function) match(pattern ast.Node) (map[Name]Value, bool) {
	switch pattern := pattern.(type) {
	case *ast.Var:
		return map[Name]Value{tokenToName(pattern.Name): f}, true
	}
	return nil, false
}

func (f Function) Apply(where token.Token, args ...Value) Value {
	if len(f.Params) != len(args) {
		f.error(where, InvalidArgumentCountError{Expected: len(f.Params), Actual: len(args)})
	}
	f.EvEnv = newEvEnv(f.EvEnv)
	for i, param := range f.Params {
		f.EvEnv.set(param, args[i])
	}

	var ret Value
	for _, node := range f.Body {
		ret = f.Eval(node)
	}
	f.EvEnv = f.EvEnv.parent
	return ret
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
	}
	return nil, false
}

func runThunk(v Value) Value {
	switch v := v.(type) {
	case Thunk:
		var ret Value
		for _, node := range v.Body {
			ret = v.Eval(node)
		}
		if _, ok := ret.(Thunk); ok {
			panic("unreachable: thunk cannot return thunk")
		}
		return ret
	default:
		return v
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
	}
	return nil, false
}

var _ Value = Object{}

// Data represents a algebraic data type value.
type Data struct {
	Tag   Name
	Elems []Value
}

func (d Data) String() string {
	var b strings.Builder
	b.WriteString(string(d.Tag))
	b.WriteString("(")
	for i, elem := range d.Elems {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(elem.String())
	}
	b.WriteString(")")
	return b.String()
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
	}
	return nil, false
}

func (c Constructor) Apply(where token.Token, args ...Value) Value {
	if len(args) != c.Params {
		c.error(where, InvalidArgumentCountError{Expected: c.Params, Actual: len(args)})
	}
	return Data{Tag: c.Tag, Elems: args}
}

var (
	_ Value    = Constructor{}
	_ Callable = Constructor{}
)
