package infix

import (
	"fmt"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

// After parsing, every infix operator treated as left-associative and has the same precedence.
// In infix.go, we will fix this.

type Resolver struct {
	decls []*ast.InfixDecl
}

func NewInfixResolver() *Resolver {
	return &Resolver{decls: make([]*ast.InfixDecl, 0)}
}

func (r *Resolver) Name() string {
	return "infix.InfixResolver"
}

func (r *Resolver) Init(program []ast.Node) error {
	for _, node := range program {
		ast.Traverse(node, func(n ast.Node) ast.Node {
			switch n := n.(type) {
			case *ast.InfixDecl:
				r.add(n)
				return n
			default:
				return n
			}
		})
	}
	return nil
}

func (r *Resolver) Run(program []ast.Node) ([]ast.Node, error) {
	for i, node := range program {
		program[i] = ast.Traverse(node, func(n ast.Node) ast.Node {
			switch n := n.(type) {
			case *ast.Binary:
				return r.mkBinary(n.Op, n.Left, n.Right)
			case *ast.Paren:
				return n.Expr
			}
			return n
		})
	}
	return program, nil
}

func (r *Resolver) add(infix *ast.InfixDecl) {
	r.decls = append(r.decls, infix)
}

func (r Resolver) prec(op token.Token) int {
	for _, decl := range r.decls {
		if decl.Name.Lexeme == op.Lexeme {
			return decl.Prec.Literal.(int)
		}
	}
	return 0
}

func (r Resolver) assoc(op token.Token) token.Kind {
	for _, decl := range r.decls {
		if decl.Name.Lexeme == op.Lexeme {
			return decl.Assoc.Kind
		}
	}
	return token.INFIXL
}

func (r Resolver) mkBinary(op token.Token, left, right ast.Node) ast.Node {
	switch left := left.(type) {
	case *ast.Binary:
		// (left.Left left.Op left.Right) op right
		if r.assocRight(left.Op, op) {
			// left.Left left.Op (left.Right op right)
			newRight := r.mkBinary(op, left.Right, right)
			return &ast.Binary{Left: left.Left, Op: left.Op, Right: newRight}
		}
		return &ast.Binary{Left: left, Op: op, Right: right}
	default:
		return &ast.Binary{Left: left, Op: op, Right: right}
	}
}

func (r Resolver) assocRight(op1, op2 token.Token) bool {
	prec1 := r.prec(op1)
	prec2 := r.prec(op2)
	if prec1 > prec2 {
		return false
	} else if prec1 < prec2 {
		return true
	}
	// same precedence
	if r.assoc(op1) != r.assoc(op2) {
		panic(utils.ErrorAt{Where: op1, Err: NeedParenError{LeftOp: op1, RightOp: op2}})
	}
	if r.assoc(op1) == token.INFIXL {
		return false
	} else if r.assoc(op1) == token.INFIXR {
		return true
	}

	panic(utils.ErrorAt{Where: op1, Err: NeedParenError{LeftOp: op1, RightOp: op2}})
}

type NeedParenError struct {
	LeftOp, RightOp token.Token
}

func (e NeedParenError) Error() string {
	return fmt.Sprintf("need parentheses around %v and %v", e.LeftOp, e.RightOp)
}
