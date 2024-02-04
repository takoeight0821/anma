package codata

import (
	"fmt"
	"log"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
)

// ToObservation converts ast.Node to Observation.
func ToObservation(copattern ast.Node) Observation {
	switch c := copattern.(type) {
	case *ast.This:
		return nil
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

// Observation is abstruction of copattern.
// This is an extension of function arity.
type Observation interface {
	fmt.Stringer
}

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

// Sequence means sequence of observations.
// e.g. `#(x, y, z).foo` is `Sequence{Apply{3}, Field{"foo"}}`
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

func push(o Observation, x Observation) Observation {
	switch o := o.(type) {
	case nil:
		return Sequence{[]Observation{x}}
	case Sequence:
		return Sequence{append(o.Observations, x)}
	default:
		return Sequence{[]Observation{o, x}}
	}
}

var _ Observation = Sequence{}

// Union means union of observations.
// e.g. `#(x, y, z).foo | #(x, y, z).bar` is `Union{Sequence{Apply{3}, Field{"foo"}}, Sequence{Apply{3}, Field{"bar"}}}`
type Union struct {
	Observations map[string]Observation
}

func (u Union) String() string {
	var result string
	isFirst := true
	for s := range u.Observations {
		if !isFirst {
			result += " | "
		}
		result += s
		isFirst = false
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
			for k, v := range y.Observations {
				x.Observations[k] = v
			}
			return x
		default:
			x.Observations[y.String()] = y
			return x
		}
	default:
		switch y := y.(type) {
		case Union:
			y.Observations[x.String()] = x
			return y
		default:
			return Union{map[string]Observation{x.String(): x, y.String(): y}}
		}
	}
}

var _ Observation = Union{}
