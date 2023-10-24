package main

import (
	"fmt"
)

// After parsing, every infix operator treated as left-associative and has the same precedence.
// In infix.go, we will fix this.

type InfixResolver struct {
	decls []*InfixDecl
}

func NewInfixResolver() *InfixResolver {
	return &InfixResolver{}
}

func (r *InfixResolver) Load(node Node) {
	Transform(node, func(n Node) Node {
		switch n := n.(type) {
		case *InfixDecl:
			r.add(n)
		}
		return n
	})
}

func (r *InfixResolver) Resolve(node Node) Node {
	return Transform(node, func(n Node) Node {
		switch n := n.(type) {
		case *Binary:
			return r.mkBinary(n.Op, n.Left, n.Right)
		case *Paren:
			if len(n.Elems) == 1 {
				return n.Elems[0]
			}
		}
		return n
	})
}

func (r *InfixResolver) add(infix *InfixDecl) {
	r.decls = append(r.decls, infix)
}

func (r InfixResolver) prec(op Token) int {
	for _, decl := range r.decls {
		if decl.Name.Lexeme == op.Lexeme {
			return decl.Prec.Literal.(int)
		}
	}
	return 0
}

func (r InfixResolver) assoc(op Token) TokenKind {
	for _, decl := range r.decls {
		if decl.Name.Lexeme == op.Lexeme {
			return decl.Assoc.Kind
		}
	}
	return INFIXL
}

func (r InfixResolver) mkBinary(op Token, left, right Node) Node {
	switch left := left.(type) {
	case *Binary:
		// (left.Left left.Op left.Right) op right
		if r.assocRight(left.Op, op) {
			// left.Left left.Op (left.Right op right)
			newRight := r.mkBinary(op, left.Right, right)
			return &Binary{Left: left.Left, Op: left.Op, Right: newRight}
		}
	}
	return &Binary{Left: left, Op: op, Right: right}
}

func (r InfixResolver) assocRight(op1, op2 Token) bool {
	prec1 := r.prec(op1)
	prec2 := r.prec(op2)
	if prec1 > prec2 {
		return false
	} else if prec1 < prec2 {
		return true
	}
	// same precedence
	if r.assoc(op1) != r.assoc(op2) {
		panic(errorAt(op2, fmt.Sprintf("cannot mix %v and %v. need parentheses", op1, op2)))
	}
	if r.assoc(op1) == INFIXL {
		return false
	} else if r.assoc(op1) == INFIXR {
		return true
	}
	panic(errorAt(op1, fmt.Sprintf("cannot mix %v and %v. need parentheses", op1, op2)))
}
