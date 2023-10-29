package infix

import (
	"fmt"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

// After parsing, every infix operator treated as left-associative and has the same precedence.
// In infix.go, we will fix this.

type InfixResolver struct {
	decls []*ast.InfixDecl
}

func NewInfixResolver() *InfixResolver {
	return &InfixResolver{}
}

func (r *InfixResolver) Init(program []ast.Node) error {
	for _, node := range program {
		ast.Transform(node, func(n ast.Node) ast.Node {
			switch n := n.(type) {
			case *ast.InfixDecl:
				r.add(n)
			}
			return n
		})
	}
	return nil
}

func (r *InfixResolver) Run(program []ast.Node) ([]ast.Node, error) {
	for i, node := range program {
		program[i] = ast.Transform(node, func(n ast.Node) ast.Node {
			switch n := n.(type) {
			case *ast.Binary:
				return r.mkBinary(n.Op, n.Left, n.Right)
			case *ast.Paren:
				if len(n.Elems) == 1 {
					return n.Elems[0]
				}
			}
			return n
		})
	}
	return program, nil
}

func (r *InfixResolver) add(infix *ast.InfixDecl) {
	r.decls = append(r.decls, infix)
}

func (r InfixResolver) prec(op token.Token) int {
	for _, decl := range r.decls {
		if decl.Name.Lexeme == op.Lexeme {
			return decl.Prec.Literal.(int)
		}
	}
	return 0
}

func (r InfixResolver) assoc(op token.Token) token.TokenKind {
	for _, decl := range r.decls {
		if decl.Name.Lexeme == op.Lexeme {
			return decl.Assoc.Kind
		}
	}
	return token.INFIXL
}

func (r InfixResolver) mkBinary(op token.Token, left, right ast.Node) ast.Node {
	switch left := left.(type) {
	case *ast.Binary:
		// (left.Left left.Op left.Right) op right
		if r.assocRight(left.Op, op) {
			// left.Left left.Op (left.Right op right)
			newRight := r.mkBinary(op, left.Right, right)
			return &ast.Binary{Left: left.Left, Op: left.Op, Right: newRight}
		}
	}
	return &ast.Binary{Left: left, Op: op, Right: right}
}

func (r InfixResolver) assocRight(op1, op2 token.Token) bool {
	prec1 := r.prec(op1)
	prec2 := r.prec(op2)
	if prec1 > prec2 {
		return false
	} else if prec1 < prec2 {
		return true
	}
	// same precedence
	if r.assoc(op1) != r.assoc(op2) {
		panic(utils.ErrorAt(op2, fmt.Sprintf("cannot mix %v and %v. need parentheses", op1, op2)))
	}
	if r.assoc(op1) == token.INFIXL {
		return false
	} else if r.assoc(op1) == token.INFIXR {
		return true
	}
	panic(utils.ErrorAt(op1, fmt.Sprintf("cannot mix %v and %v. need parentheses", op1, op2)))
}
