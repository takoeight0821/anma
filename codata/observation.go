package codata

import (
	"fmt"
	"log"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
)

// Observation is abstruction of copattern.
// It is like Space in the swift compiler.

type Observation interface{}

// Field means field access.
// e.g. `.foo`
type Field struct {
	Name token.Token
}

func (f Field) String() string {
	return "." + f.Name.String()
}

var _ Observation = Field{}

// Apply means arguments.
// e.g. `(x, y, z)`
type Apply struct {
	Count int
}

func (a Apply) String() string {
	return fmt.Sprintf("(%d)", a.Count)
}

var _ Observation = Apply{}

// This means `#` keyword.
type This struct{}

func (t This) String() string {
	return "#"
}

var _ Observation = This{}

// Sequence means sequence of observations.
// e.g. `#(x, y, z).foo` is `Sequence{This{}, Apply{3}, Field{"foo"}}`
type Sequence struct {
	Observations []Observation
}

func (s Sequence) String() string {
	var result string
	for _, o := range s.Observations {
		result += o.(fmt.Stringer).String()
	}
	return result
}

var _ Observation = Sequence{}

// Union means union of observations.
// e.g. `#(x, y, z).foo | #(x, y, z).bar` is `Union{Sequence{This{}, Apply{3}, Field{"foo"}}, Sequence{This{}, Apply{3}, Field{"bar"}}}`
type Union struct {
	Observations []Observation
}

func (u Union) String() string {
	var result string
	for i, o := range u.Observations {
		if i != 0 {
			result += " | "
		}
		result += o.(fmt.Stringer).String()
	}
	return result
}

func merge(x, y Observation) Observation {
	switch x := x.(type) {
	case nil:
		return y
	case Union:
		switch y := y.(type) {
		case Union:
			return Union{append(x.Observations, y.Observations...)}
		default:
			return Union{append(x.Observations, y)}
		}
	default:
		switch y := y.(type) {
		case Union:
			return Union{append(y.Observations, x)}
		default:
			return Union{[]Observation{x, y}}
		}
	}
}

var _ Observation = Union{}

// ToObservation converts ast.Node to Observation.
func ToObservation(copattern ast.Node) Observation {
	switch c := copattern.(type) {
	case *ast.This:
		return This{}
	case *ast.Call:
		f := ToObservation(c.Func)
		return push(f, Apply{Count: len(c.Args)})
	case *ast.Access:
		r := ToObservation(c.Receiver)
		return push(r, Field{Name: c.Name})
	default:
		log.Panicf("unexpected node %v", c)
		return nil
	}
}

func push(o Observation, x Observation) Observation {
	switch o := o.(type) {
	case Sequence:
		return Sequence{append(o.Observations, x)}
	default:
		return Sequence{[]Observation{o, x}}
	}
}
