package eval

import (
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/token"
)

type Evaluator struct {
	*evEnv
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		evEnv: newEvEnv(nil),
	}
}

type Name string

func tokenToName(t token.Token) Name {
	if t.Kind != token.IDENT && t.Kind != token.OPERATOR {
		panic(fmt.Sprintf("tokenToName: %s", t))
	}

	return Name(fmt.Sprintf("%s.%#v", t.Lexeme, t.Literal))
}

type evEnv struct {
	parent *evEnv
	values map[Name]Value
}

func newEvEnv(parent *evEnv) *evEnv {
	return &evEnv{
		parent: parent,
		values: make(map[Name]Value),
	}
}

func (env *evEnv) String() string {
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

func (env *evEnv) get(name Name) Value {
	if v, ok := env.values[name]; ok {
		return v
	}
	if env.parent != nil {
		return env.parent.get(name)
	}
	return nil
}

func (env *evEnv) set(name Name, v Value) {
	env.values[name] = v
}

func (env *evEnv) SearchMain() (Function, bool) {
	if env == nil {
		return Function{}, false
	}

	for name, v := range env.values {
		if strings.HasPrefix(string(name), "main.") {
			f, ok := v.(Function)
			if !ok {
				return Function{}, false
			}
			return f, true
		}
	}

	return env.parent.SearchMain()
}
