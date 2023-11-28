package eval

import (
	"errors"
	"fmt"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

// Eval evaluates the given node and returns the result.
func (ev *Evaluator) Eval(node ast.Node) (Value, error) {
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
		return Unit{}, ev.evalLet(node)
	case *ast.Codata:
		panic("unreachable: codata must be desugared")
	case *ast.Clause:
		panic("unreachable: clause cannot appear outside of case")
	case *ast.Lambda:
		return ev.evalLambda(node), nil
	case *ast.Case:
		return ev.evalCase(node)
	case *ast.Object:
		return ev.evalObject(node), nil
	case *ast.Field:
		panic("unreachable: field cannot appear outside of object")
	case *ast.TypeDecl:
		return Unit{}, ev.evalTypeDecl(node)
	case *ast.VarDecl:
		return Unit{}, ev.evalVarDecl(node)
	case *ast.InfixDecl:
		return Unit{}, nil
	case *ast.This:
		panic("unreachable: this cannot appear outside of pattern")
	}

	panic(fmt.Sprintf("unreachable: %v", node))
}

func (ev *Evaluator) evalVar(node *ast.Var) (Value, error) {
	name := tokenToName(node.Name)
	if v := ev.EvEnv.get(name); v != nil {
		return v, nil
	}
	return nil, utils.ErrorAt{Where: node.Base(), Err: UndefinedVariableError{Name: node.Name}}
}

func (ev *Evaluator) evalLiteral(node *ast.Literal) (Value, error) {
	//exhaustive:ignore
	switch node.Kind {
	case token.INTEGER:
		return Int(node.Literal.(int)), nil
	case token.STRING:
		return String(node.Literal.(string)), nil
	default:
		return nil, errorAt(node.Base(), InvalidLiteralError{Kind: node.Kind})
	}
}

func (ev *Evaluator) evalParen(node *ast.Paren) (Value, error) {
	return ev.Eval(node.Expr)
}

func (ev *Evaluator) evalAccess(node *ast.Access) (Value, error) {
	receiver, err := ev.Eval(node.Receiver)
	if err != nil {
		return nil, err
	}

	switch receiver := receiver.(type) {
	case Object:
		if v, ok := receiver.Fields[node.Name.Lexeme]; ok {
			v, err = runThunk(v)
			if err != nil {
				return nil, err
			}
			receiver.Fields[node.Name.Lexeme] = v
			return v, nil
		}
		return nil, errorAt(node.Base(), UndefinedFieldError{Receiver: receiver, Name: node.Name.Lexeme})
	default:
		return nil, errorAt(node.Base(), NotObjectError{Receiver: receiver})
	}
}

func (ev *Evaluator) evalCall(node *ast.Call) (Value, error) {
	fn, err := ev.Eval(node.Func)
	if err != nil {
		return nil, err
	}
	switch fn := fn.(type) {
	case Callable:
		args := make([]Value, len(node.Args))
		for i, arg := range node.Args {
			args[i], err = ev.Eval(arg)
			if err != nil {
				return nil, err
			}
		}
		v, err := fn.Apply(node.Base(), args...)
		if err != nil {
			return nil, errorAt(node.Base(), err)
		}
		return v, nil
	default:
		return nil, errorAt(node.Base(), NotCallableError{Func: fn})
	}
}

func (ev *Evaluator) evalPrim(node *ast.Prim) (Value, error) {
	prim := fetchPrim(node.Name)
	if prim == nil {
		return nil, errorAt(node.Base(), UndefinedPrimError{Name: node.Name})
	}

	args := make([]Value, len(node.Args))
	for i, arg := range node.Args {
		var err error
		args[i], err = ev.Eval(arg)
		if err != nil {
			return nil, err
		}
	}

	return prim(ev, args...)
}

func asInt(v Value) (Int, bool) {
	switch v := v.(type) {
	case Int:
		return v, true
	default:
		return 0, false
	}
}

func fetchPrim(name token.Token) func(*Evaluator, ...Value) (Value, error) {
	switch name.Lexeme {
	case "add":
		return func(ev *Evaluator, args ...Value) (Value, error) {
			if len(args) != 2 {
				return nil, errorAt(name, InvalidArgumentCountError{Expected: 2, Actual: len(args)})
			}
			v0, ok := asInt(args[0])
			if !ok {
				return nil, errorAt(name, InvalidArgumentTypeError{Expected: "Int", Actual: args[0]})
			}
			v1, ok := asInt(args[1])
			if !ok {
				return nil, errorAt(name, InvalidArgumentTypeError{Expected: "Int", Actual: args[1]})
			}
			return v0 + v1, nil
		}
	case "mul":
		return func(ev *Evaluator, args ...Value) (Value, error) {
			if len(args) != 2 {
				return nil, errorAt(name, InvalidArgumentCountError{Expected: 2, Actual: len(args)})
			}
			v0, ok := asInt(args[0])
			if !ok {
				return nil, errorAt(name, InvalidArgumentTypeError{Expected: "Int", Actual: args[0]})
			}
			v1, ok := asInt(args[1])
			if !ok {
				return nil, errorAt(name, InvalidArgumentTypeError{Expected: "Int", Actual: args[1]})
			}
			return v0 * v1, nil
		}
	default:
		return nil
	}
}

func (ev *Evaluator) evalBinary(node *ast.Binary) (Value, error) {
	name := tokenToName(node.Op)
	if op := ev.EvEnv.get(name); op != nil {
		switch op := op.(type) {
		case Callable:
			left, err := ev.Eval(node.Left)
			if err != nil {
				return nil, err
			}
			right, err := ev.Eval(node.Right)
			if err != nil {
				return nil, err
			}
			v, err := op.Apply(node.Base(), left, right)
			if err != nil {
				return nil, errorAt(node.Base(), err)
			}
			return v, nil
		default:
			return nil, errorAt(node.Base(), NotCallableError{Func: op})
		}
	}
	return nil, utils.ErrorAt{Where: node.Base(), Err: UndefinedVariableError{Name: node.Op}}
}

func (ev *Evaluator) evalAssert(node *ast.Assert) (Value, error) {
	return ev.Eval(node.Expr)
}

// evalLet evaluates the given let expression.
// let expression does not create a new scope.
// It just overrides the existing bindings or creates new bindings if not exists.
func (ev *Evaluator) evalLet(node *ast.Let) error {
	body, err := ev.Eval(node.Body)
	if err != nil {
		return err
	}
	if env, ok := body.match(node.Bind); ok {
		for name, v := range env {
			ev.EvEnv.set(name, v)
		}
		return nil
	}
	return errorAt(node.Base(), PatternMatchError{Patterns: []ast.Node{node.Bind}, Values: []Value{body}})
}

func (ev *Evaluator) evalLambda(node *ast.Lambda) Function {
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

// evalCase evaluates the given case expression.
// It first evaluates all scrutinees and then tries to match them with each clause.
// If a match is found, it evaluates the corresponding expressions and returns the result.
// If no match is found, it returns an error.
func (ev *Evaluator) evalCase(node *ast.Case) (Value, error) {
	scrs := make([]Value, len(node.Scrutinees))
	for i, scr := range node.Scrutinees {
		var err error
		scrs[i], err = ev.Eval(scr)
		if err != nil {
			return nil, err
		}
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
				var err error
				ret, err = ev.Eval(expr)
				if err != nil {
					return nil, err
				}
			}

			ev.EvEnv = ev.EvEnv.parent
			return ret, nil
		}
		err = errors.Join(err, PatternMatchError{Patterns: clause.Patterns, Values: scrs})
	}
	return nil, errorAt(node.Base(), err)
}

// matchClause matches the given clause's patterns with the given scrutinees.
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

func (ev *Evaluator) evalObject(node *ast.Object) Object {
	fields := make(map[string]Value)
	for _, field := range node.Fields {
		fields[field.Name] = Thunk{Evaluator: *ev, Body: field.Exprs}
	}
	return Object{Fields: fields}
}

func (ev *Evaluator) evalTypeDecl(node *ast.TypeDecl) error {
	for _, ctor := range node.Types {
		err := ev.defineConstructor(ctor)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ev *Evaluator) defineConstructor(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Var:
		ev.EvEnv.set(tokenToName(node.Name), Data{Tag: tokenToName(node.Name), Elems: nil})
		return nil
	case *ast.Call:
		switch fn := node.Func.(type) {
		case *ast.Var:
			ev.EvEnv.set(tokenToName(fn.Name), Constructor{Evaluator: *ev, Tag: tokenToName(fn.Name), Params: len(node.Args)})
			return nil
		case *ast.Prim:
			// For type checking
			// Ignore in evaluation
			return nil
		}
	case *ast.Prim:
		// For type checking
		// Ignore in evaluation
		return nil
	}
	return errorAt(node.Base(), NotConstructorError{Node: node})
}

func (ev *Evaluator) evalVarDecl(node *ast.VarDecl) error {
	if node.Expr != nil {
		v, err := ev.Eval(node.Expr)
		if err != nil {
			return err
		}
		ev.EvEnv.set(tokenToName(node.Name), v)
	}
	return nil
}
