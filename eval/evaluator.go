package eval

import (
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/token"
)

type Evaluator struct {
	*EvEnv
	handler func(error)
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		EvEnv: newEvEnv(nil),
	}
}

func (ev *Evaluator) SetErrorHandler(handler func(error)) {
	ev.handler = handler
}

func (ev *Evaluator) error(where token.Token, err error) {
	if where.Kind == token.EOF {
		err = fmt.Errorf("at end: %w", err)
	} else {
		err = fmt.Errorf("at %d: `%s`, %w", where.Line, where.Lexeme, err)
	}

	if ev.handler != nil {
		ev.handler(err)
	} else {
		panic(err)
	}
}

type Name string

func tokenToName(t token.Token) Name {
	if t.Kind != token.IDENT && t.Kind != token.OPERATOR {
		panic(fmt.Sprintf("tokenToName: %s", t))
	}

	return Name(fmt.Sprintf("%s.%#v", t.Lexeme, t.Literal))
}

type EvEnv struct {
	parent *EvEnv
	values map[Name]Value
}

func newEvEnv(parent *EvEnv) *EvEnv {
	return &EvEnv{
		parent: parent,
		values: make(map[Name]Value),
	}
}

func (env *EvEnv) String() string {
	var b strings.Builder
	b.WriteString("{")
	for name, v := range env.values {
		b.WriteString(fmt.Sprintf(" %s:%v", name, v))
	}
	b.WriteString(" }")
	if env.parent != nil {
		b.WriteString("\n\t&")
		b.WriteString(env.parent.String())
	}
	return b.String()
}

func (env *EvEnv) get(name Name) Value {
	if v, ok := env.values[name]; ok {
		return v
	}
	if env.parent != nil {
		return env.parent.get(name)
	}
	return nil
}

func (env *EvEnv) set(name Name, v Value) {
	env.values[name] = v
}

func (env *EvEnv) SearchMain() (Value, bool) {
	if env == nil {
		return nil, false
	}

	for name, v := range env.values {
		if strings.HasPrefix(string(name), "main.") {
			return v, true
		}
	}

	return env.parent.SearchMain()
}
