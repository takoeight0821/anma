package eval

import (
	"errors"
	"fmt"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
)

// Eval evaluates the given node and returns the result.
func (ev *Evaluator) Eval(node ast.Node) Value {
	if ev.err != nil {
		return nil
	}
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

	ev.error(node.Base(), NotCallableError{Func: fn})
	return nil
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

func asInt(v Value) (Int, bool) {
	switch v := v.(type) {
	case Int:
		return v, true
	}
	return 0, false
}

func fetchPrim(name token.Token) func(*Evaluator, ...Value) Value {
	switch name.Lexeme {
	case "add":
		return func(ev *Evaluator, args ...Value) Value {
			if len(args) != 2 {
				ev.error(name, InvalidArgumentCountError{Expected: 2, Actual: len(args)})
				return nil
			}
			v0, ok := asInt(args[0])
			if !ok {
				ev.error(name, InvalidArgumentTypeError{Expected: "Int", Actual: args[0]})
				return nil
			}
			v1, ok := asInt(args[1])
			if !ok {
				ev.error(name, InvalidArgumentTypeError{Expected: "Int", Actual: args[1]})
				return nil
			}
			return Int(v0 + v1)
		}
	case "mul":
		return func(ev *Evaluator, args ...Value) Value {
			if len(args) != 2 {
				ev.error(name, InvalidArgumentCountError{Expected: 2, Actual: len(args)})
				return nil
			}
			v0, ok := asInt(args[0])
			if !ok {
				ev.error(name, InvalidArgumentTypeError{Expected: "Int", Actual: args[0]})
				return nil
			}
			v1, ok := asInt(args[1])
			if !ok {
				ev.error(name, InvalidArgumentTypeError{Expected: "Int", Actual: args[1]})
				return nil
			}
			return Int(v0 * v1)
		}
	default:
		return nil
	}
}

func (ev *Evaluator) evalBinary(node *ast.Binary) Value {
	name := tokenToName(node.Op)
	if op := ev.EvEnv.get(name); op != nil {
		switch op := op.(type) {
		case Callable:
			return op.Apply(node.Base(), ev.Eval(node.Left), ev.Eval(node.Right))
		}
		ev.error(node.Base(), NotCallableError{Func: op})
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

func (ev *Evaluator) evalVarDecl(node *ast.VarDecl) Value {
	if node.Expr != nil {
		ev.EvEnv.set(tokenToName(node.Name), ev.Eval(node.Expr))
	}
	return Unit{}
}
