package eval

import (
	"errors"
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/ast"
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

type Value interface {
	fmt.Stringer
	match(pattern ast.Node) (map[Name]Value, bool)
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

// Eval evaluates the given node and returns the result.
func (ev *Evaluator) Eval(node ast.Node) Value {
	switch node := node.(type) {
	case *ast.Var:
		return ev.evalVar(node)
	case *ast.Literal:
		return ev.evalLiteral(node)
	case *ast.Paren:
		return ev.evalParen(node)
	case *ast.Access:
		return ev.evalAccess(node)
	case *ast.Call:
		return ev.evalCall(node)
	case *ast.Prim:
		return ev.evalPrim(node)
	case *ast.Binary:
		return ev.evalBinary(node)
	case *ast.Assert:
		return ev.evalAssert(node)
	case *ast.Let:
		return ev.evalLet(node)
	case *ast.Codata:
		panic("unreachable: codata must be desugared")
	case *ast.Clause:
		panic("unreachable: clause cannot appear outside of case")
	case *ast.Lambda:
		return ev.evalLambda(node)
	case *ast.Case:
		return ev.evalCase(node)
	case *ast.Object:
		return ev.evalObject(node)
	case *ast.Field:
		panic("unreachable: field cannot appear outside of object")
	case *ast.TypeDecl:
		return ev.evalTypeDecl(node)
	case *ast.VarDecl:
		return ev.evalVarDecl(node)
	case *ast.InfixDecl:
		return Unit{}
	case *ast.This:
		panic("unreachable: this cannot appear outside of pattern")
	}

	panic(fmt.Sprintf("unreachable: %v", node))
}

func (ev *Evaluator) evalVar(node *ast.Var) Value {
	name := tokenToName(node.Name)
	if v := ev.EvEnv.get(name); v != nil {
		return v
	}
	ev.error(node.Base(), UndefinedVariableError{Name: name})
	return nil
}

type UndefinedVariableError struct {
	Name Name
}

func (e UndefinedVariableError) Error() string {
	return fmt.Sprintf("undefined variable `%v`", e.Name)
}

func (ev *Evaluator) evalLiteral(node *ast.Literal) Value {
	//exhaustive:ignore
	switch node.Kind {
	case token.INTEGER:
		return Int(node.Literal.(int))
	case token.STRING:
		return String(node.Literal.(string))
	default:
		ev.error(node.Base(), InvalidLiteralError{Kind: node.Kind})
		return nil
	}
}

type InvalidLiteralError struct {
	Kind token.TokenKind
}

func (e InvalidLiteralError) Error() string {
	return fmt.Sprintf("invalid literal `%v`", e.Kind)
}

func (ev *Evaluator) evalParen(node *ast.Paren) Value {
	return ev.Eval(node.Expr)
}

func (ev *Evaluator) evalAccess(node *ast.Access) Value {
	receiver := ev.Eval(node.Receiver)
	switch receiver := receiver.(type) {
	case Object:
		if v, ok := receiver.Fields[node.Name.Lexeme]; ok {
			// TODO: update receiver.Fields[name] to runThunk(v)
			return runThunk(v)
		}
		ev.error(node.Base(), UndefinedFieldError{Receiver: receiver, Name: node.Name.Lexeme})
	}
	return nil
}

type UndefinedFieldError struct {
	Receiver Object
	Name     string
}

func (e UndefinedFieldError) Error() string {
	return fmt.Sprintf("undefined field `%v` of %s", e.Name, e.Receiver)
}

func (ev *Evaluator) evalCall(node *ast.Call) Value {
	fn := ev.Eval(node.Func)
	switch fn := fn.(type) {
	case Callable:
		args := make([]Value, len(node.Args))
		for i, arg := range node.Args {
			args[i] = ev.Eval(arg)
		}
		return fn.Apply(node.Base(), args...)
	}

	ev.error(node.Base(), NotFunctionError{Func: fn})
	return nil
}

type Callable interface {
	Apply(token.Token, ...Value) Value
}

type InvalidArgumentCountError struct {
	Expected int
	Actual   int
}

func (e InvalidArgumentCountError) Error() string {
	return fmt.Sprintf("invalid argument count: expected %d, actual %d", e.Expected, e.Actual)
}

type NotFunctionError struct {
	Func Value
}

func (e NotFunctionError) Error() string {
	return fmt.Sprintf("not a function: %v", e.Func)
}

func (ev *Evaluator) evalPrim(node *ast.Prim) Value {
	prim := fetchPrim(node.Name)
	if prim == nil {
		ev.error(node.Base(), UndefinedPrimError{Name: node.Name})
		return nil
	}

	args := make([]Value, len(node.Args))
	for i, arg := range node.Args {
		args[i] = ev.Eval(arg)
	}

	return prim(ev, args...)
}

func fetchPrim(name token.Token) func(*Evaluator, ...Value) Value {
	switch name.Lexeme {
	case "add":
		return func(ev *Evaluator, args ...Value) Value {
			if len(args) != 2 {
				ev.error(name, InvalidArgumentCountError{Expected: 2, Actual: len(args)})
				return nil
			}
			switch args[0].(type) {
			case Int:
				switch args[1].(type) {
				case Int:
					return Int(args[0].(Int) + args[1].(Int))
				}
			}
			ev.error(name, InvalidArgumentTypeError{Expected: "Int", Actual: args[0]})
			return nil
		}
	case "mul":
		return func(ev *Evaluator, args ...Value) Value {
			if len(args) != 2 {
				ev.error(name, InvalidArgumentCountError{Expected: 2, Actual: len(args)})
				return nil
			}
			switch args[0].(type) {
			case Int:
				switch args[1].(type) {
				case Int:
					return Int(args[0].(Int) * args[1].(Int))
				}
			}
			ev.error(name, InvalidArgumentTypeError{Expected: "Int", Actual: args[0]})
			return nil
		}
	default:
		return nil
	}
}

type InvalidArgumentTypeError struct {
	Expected string
	Actual   Value
}

func (e InvalidArgumentTypeError) Error() string {
	return fmt.Sprintf("invalid argument type: expected %s, actual %v", e.Expected, e.Actual)
}

type UndefinedPrimError struct {
	Name token.Token
}

func (e UndefinedPrimError) Error() string {
	return fmt.Sprintf("undefined prim `%v`", e.Name)
}

func (ev *Evaluator) evalBinary(node *ast.Binary) Value {
	name := tokenToName(node.Op)
	if op := ev.EvEnv.get(name); op != nil {
		switch op := op.(type) {
		case Callable:
			return op.Apply(node.Base(), ev.Eval(node.Left), ev.Eval(node.Right))
		}
		ev.error(node.Base(), NotFunctionError{Func: op})
		return nil
	}
	ev.error(node.Base(), UndefinedVariableError{Name: name})
	return nil
}

func (ev *Evaluator) evalAssert(node *ast.Assert) Value {
	return ev.Eval(node.Expr)
}

func (ev *Evaluator) evalLet(node *ast.Let) Value {
	body := ev.Eval(node.Body)
	if env, ok := body.match(node.Bind); ok {
		for name, v := range env {
			ev.EvEnv.set(name, v)
		}
		return Unit{}
	}
	ev.error(node.Base(), PatternMatchError{Patterns: []ast.Node{node.Bind}, Values: []Value{body}})
	return nil
}

type PatternMatchError struct {
	Patterns []ast.Node
	Values   []Value
}

func (e PatternMatchError) Error() string {
	return fmt.Sprintf("pattern match failed: %v = %v", e.Patterns, e.Values)
}

func (ev *Evaluator) evalLambda(node *ast.Lambda) Value {
	params := make([]Name, len(node.Params))
	for i, param := range node.Params {
		params[i] = tokenToName(param)
	}
	return Function{
		Evaluator: *ev,
		Params:    params,
		Body:      node.Exprs,
	}
}

func (ev *Evaluator) evalCase(node *ast.Case) Value {
	scrs := make([]Value, len(node.Scrutinees))
	for i, scr := range node.Scrutinees {
		scrs[i] = ev.Eval(scr)
	}

	var err error
	for _, clause := range node.Clauses {
		if env, ok := matchClause(clause, scrs); ok {
			ev.EvEnv = newEvEnv(ev.EvEnv)

			for name, v := range env {
				ev.EvEnv.set(name, v)
			}
			var ret Value
			for _, expr := range clause.Exprs {
				ret = ev.Eval(expr)
			}

			ev.EvEnv = ev.EvEnv.parent
			return ret
		}
		err = errors.Join(err, PatternMatchError{Patterns: clause.Patterns, Values: scrs})
	}
	ev.error(node.Base(), err)
	return nil
}

func matchClause(clause *ast.Clause, scrs []Value) (map[Name]Value, bool) {
	if len(clause.Patterns) != len(scrs) {
		return nil, false
	}
	env := make(map[Name]Value)
	for i, pattern := range clause.Patterns {
		m, ok := scrs[i].match(pattern)
		if !ok {
			return nil, false
		}
		for k, v := range m {
			env[k] = v
		}
	}
	return env, true
}

func (ev *Evaluator) evalObject(node *ast.Object) Value {
	fields := make(map[string]Value)
	for _, field := range node.Fields {
		fields[field.Name] = Thunk{Evaluator: *ev, Body: field.Exprs}
	}
	return Object{Fields: fields}
}

func (ev *Evaluator) evalTypeDecl(node *ast.TypeDecl) Value {
	for _, ctor := range node.Types {
		ev.defineConstructor(ctor)
	}
	return Unit{}
}

func (ev *Evaluator) defineConstructor(node ast.Node) {
	switch node := node.(type) {
	case *ast.Var:
		ev.EvEnv.set(tokenToName(node.Name), Data{Tag: tokenToName(node.Name), Elems: nil})
		return
	case *ast.Call:
		switch fn := node.Func.(type) {
		case *ast.Var:
			ev.EvEnv.set(tokenToName(fn.Name), Constructor{Evaluator: *ev, Tag: tokenToName(fn.Name), Params: len(node.Args)})
			return
		case *ast.Prim:
			// For type checking
			// Ignore in evaluation
			return
		}
	case *ast.Prim:
		// For type checking
		// Ignore in evaluation
		return
	}
	ev.error(node.Base(), NotConstructorError{Node: node})
}

type NotConstructorError struct {
	Node ast.Node
}

func (e NotConstructorError) Error() string {
	return fmt.Sprintf("not a constructor: %v", e.Node)
}

func (ev *Evaluator) evalVarDecl(node *ast.VarDecl) Value {
	if node.Expr != nil {
		ev.EvEnv.set(tokenToName(node.Name), ev.Eval(node.Expr))
	}
	return Unit{}
}
