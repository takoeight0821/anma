package main

import (
	"errors"
	"fmt"
)

//go:generate go run ./tools/main.go -comment -in parser.go -out docs/syntax.ebnf

type Parser struct {
	tokens  []Token
	current int
	err     error
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens, 0, nil}
}

func (p *Parser) ParseExpr() (Node, error) {
	p.err = nil
	node := p.expr()
	return node, p.err
}

func (p *Parser) ParseDecl() ([]Node, error) {
	p.err = nil
	nodes := []Node{}
	for !p.IsAtEnd() {
		nodes = append(nodes, p.decl())
	}
	return nodes, p.err
}

// decl = typeDecl | varDecl | infixDecl ;
func (p *Parser) decl() Node {
	if p.match(TYPE) {
		return p.typeDecl()
	}
	if p.match(DEF) {
		return p.varDecl()
	}
	return p.infixDecl()
}

// typeDecl = "type" IDENTIFIER "=" type ;
func (p *Parser) typeDecl() TypeDecl {
	p.consume(TYPE, "expected `type`")
	name := p.consume(IDENT, "expected identifier")
	p.consume(EQUAL, "expected `=`")
	typ := p.typ()
	return TypeDecl{Name: name, Type: typ}
}

// varDecl = "def" identifier "=" expr | "def" identifier ":" type | "def" identifier ":" type "=" expr ;
func (p *Parser) varDecl() VarDecl {
	p.consume(DEF, "expected `def`")
	name := p.consume(IDENT, "expected identifier")
	var typ Node
	var expr Node
	if p.match(COLON) {
		p.advance()
		typ = p.typ()
	}
	if p.match(EQUAL) {
		p.advance()
		expr = p.expr()
	}
	return VarDecl{Name: name, Type: typ, Expr: expr}
}

// infixDecl = ("infix" | "infixl" | "infixr") INTEGER OPERATOR ;
func (p *Parser) infixDecl() InfixDecl {
	kind := p.advance()
	if kind.Kind != INFIX && kind.Kind != INFIXL && kind.Kind != INFIXR {
		p.recover(parseError(kind, "expected `infix`, `infixl`, or `infixr`"))
		return InfixDecl{}
	}
	precedence := p.consume(INTEGER, "expected integer")
	name := p.consume(OPERATOR, "expected operator")
	return InfixDecl{Assoc: kind, Prec: precedence, Name: name}
}

// expr = let | fn | assert ;
func (p *Parser) expr() Node {
	if p.IsAtEnd() {
		p.recover(parseError(p.peek(), "expected expression"))
		return nil
	}
	if p.match(LET) {
		return p.let()
	}
	if p.match(FN) {
		return p.fn()
	}
	return p.assert()
}

// let = "let" pattern "=" assert ;
func (p *Parser) let() Let {
	p.advance()
	pattern := p.pattern()
	p.consume(EQUAL, "expected `=`")
	expr := p.assert()
	return Let{Bind: pattern, Body: expr}
}

// fn = "fn" pattern "{" expr (";" expr)* ";"? "}" ;
func (p *Parser) fn() Lambda {
	p.advance()
	pattern := p.pattern()
	p.consume(LEFTBRACE, "expected `{`")
	exprs := []Node{p.expr()}
	for p.match(SEMICOLON) {
		p.advance()
		if p.match(RIGHTBRACE) {
			break
		}
		exprs = append(exprs, p.expr())
	}
	p.consume(RIGHTBRACE, "expected `}`")
	return Lambda{Pattern: pattern, Exprs: exprs}
}

// atom = var | literal | paren | codata ;
func (p *Parser) atom() Node {
	switch t := p.advance(); t.Kind {
	case IDENT:
		return Var{Name: t}
	case INTEGER, STRING:
		return Literal{Token: t}
	case LEFTPAREN:
		if p.match(RIGHTPAREN) {
			p.advance()
			return Paren{}
		}
		elems := []Node{p.expr()}
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHTPAREN) {
				break
			}
			elems = append(elems, p.expr())
		}
		p.consume(RIGHTPAREN, "expected `)`")
		return Paren{Elems: elems}
	case LEFTBRACE:
		return p.codata()
	default:
		p.recover(parseError(t, "expected variable, literal, or parenthesized expression"))
		return nil
	}
}

// assert = binop (":" type)* ;
func (p *Parser) assert() Node {
	expr := p.binary()
	for p.match(COLON) {
		p.advance()
		typ := p.typ()
		expr = Assert{Expr: expr, Type: typ}
	}
	return expr
}

// binary = access (operator access)* ;
func (p *Parser) binary() Node {
	expr := p.access()
	for p.match(OPERATOR) {
		op := p.advance()
		right := p.access()
		expr = Binary{Left: expr, Op: op, Right: right}
	}
	return expr
}

// access = call ("." IDENTIFIER)* ;
func (p *Parser) access() Node {
	expr := p.call()
	for p.match(DOT) {
		p.advance()
		name := p.consume(IDENT, "expected identifier")
		expr = Access{Receiver: expr, Name: name}
	}
	return expr
}

// call = atom ("(" ")" | "(" expr ("," expr)* ","? ")")* ;
func (p *Parser) call() Node {
	expr := p.atom()
	for p.match(LEFTPAREN) {
		expr = p.finishCall(expr)
	}
	return expr
}

func (p *Parser) finishCall(fun Node) Call {
	p.consume(LEFTPAREN, "expected `(`")
	args := []Node{}
	if !p.match(RIGHTPAREN) {
		args = append(args, p.expr())
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHTPAREN) {
				break
			}
			args = append(args, p.expr())
		}
	}
	p.consume(RIGHTPAREN, "expected `)`")
	return Call{Func: fun, Args: args}
}

// codata = "{" clause ("," clause)* ","? "}" ;
func (p *Parser) codata() Codata {
	clauses := []Clause{p.clause()}
	for p.match(COMMA) {
		p.advance()
		if p.match(RIGHTBRACE) {
			break
		}
		clauses = append(clauses, p.clause())
	}
	p.consume(RIGHTBRACE, "expected `}`")
	return Codata{Clauses: clauses}
}

// clause = pattern "->" expr (";" expr)* ";"? ;
func (p *Parser) clause() Clause {
	pattern := p.pattern()
	p.consume(ARROW, "expected `->`")
	exprs := []Node{p.expr()}
	for p.match(SEMICOLON) {
		p.advance()
		if p.match(RIGHTBRACE) {
			break
		}
		exprs = append(exprs, p.expr())
	}
	return Clause{Pattern: pattern, Exprs: exprs}
}

// pattern = accessPat ;
func (p *Parser) pattern() Node {
	if p.IsAtEnd() {
		p.recover(parseError(p.peek(), "expected pattern"))
		return nil
	}
	return p.accessPat()
}

// accessPat = callPat ("." IDENTIFIER)* ;
func (p *Parser) accessPat() Node {
	pat := p.callPat()
	for p.match(DOT) {
		p.advance()
		name := p.consume(IDENT, "expected identifier")
		pat = Access{Receiver: pat, Name: name}
	}
	return pat
}

// callPat = atomPat ("(" ")" | "(" pattern ("," pattern)* ","? ")")* ;
func (p *Parser) callPat() Node {
	pat := p.atomPat()
	for p.match(LEFTPAREN) {
		pat = p.finishCallPat(pat)
	}
	return pat
}

func (p *Parser) finishCallPat(fun Node) Call {
	p.consume(LEFTPAREN, "expected `(`")
	args := []Node{}
	if !p.match(RIGHTPAREN) {
		args = append(args, p.pattern())
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHTPAREN) {
				break
			}
			args = append(args, p.pattern())
		}
	}
	p.consume(RIGHTPAREN, "expected `)`")
	return Call{Func: fun, Args: args}
}

// atomPat = IDENT | INTEGER | STRING | "(" pattern ("," pattern)* ","? ")" ;
func (p *Parser) atomPat() Node {
	switch t := p.advance(); t.Kind {
	case SHARP:
		return This{Token: t}
	case IDENT:
		return Var{Name: t}
	case INTEGER, STRING:
		return Literal{Token: t}
	case LEFTPAREN:
		if p.match(RIGHTPAREN) {
			p.advance()
			return Paren{}
		}
		patterns := []Node{p.pattern()}
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHTPAREN) {
				break
			}
			patterns = append(patterns, p.pattern())
		}
		p.consume(RIGHTPAREN, "expected `)`")
		return Paren{Elems: patterns}
	default:
		p.recover(parseError(t, "expected variable, literal, or parenthesized pattern"))
		return nil
	}
}

// type = binopType ;
func (p *Parser) typ() Node {
	if p.IsAtEnd() {
		p.recover(parseError(p.peek(), "expected type"))
		return nil
	}
	return p.binopType()
}

// binopType = callType (operator callType)* ;
func (p *Parser) binopType() Node {
	typ := p.callType()
	for p.match(OPERATOR) || p.match(ARROW) {
		op := p.advance()
		right := p.callType()
		typ = Binary{Left: typ, Op: op, Right: right}
	}
	return typ
}

// callType = atomType ("(" ")" | "(" type ("," type)* ","? ")")* ;
func (p *Parser) callType() Node {
	typ := p.atomType()
	for p.match(LEFTPAREN) {
		typ = p.finishCallType(typ)
	}
	return typ
}

func (p *Parser) finishCallType(fun Node) Call {
	p.consume(LEFTPAREN, "expected `(`")
	args := []Node{}
	if !p.match(RIGHTPAREN) {
		args = append(args, p.typ())
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHTPAREN) {
				break
			}
			args = append(args, p.typ())
		}
	}
	p.consume(RIGHTPAREN, "expected `)`")
	return Call{Func: fun, Args: args}
}

// atomType = IDENT | "{" fieldType ("," fieldType)* ","? "}" | "(" type ("," type)* ","? ")" ;
func (p *Parser) atomType() Node {
	switch t := p.advance(); t.Kind {
	case IDENT:
		return Var{Name: t}
	case LEFTBRACE:
		fields := []Field{p.fieldType()}
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHTBRACE) {
				break
			}
			fields = append(fields, p.fieldType())
		}
		p.consume(RIGHTBRACE, "expected `}`")
		return Object{Fields: fields}
	case LEFTPAREN:
		if p.match(RIGHTPAREN) {
			p.advance()
			return Paren{}
		}
		types := []Node{p.typ()}
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHTPAREN) {
				break
			}
			types = append(types, p.typ())
		}
		p.consume(RIGHTPAREN, "expected `)`")
		return Paren{Elems: types}
	default:
		p.recover(parseError(t, "expected variable or parenthesized type"))
		return nil
	}
}

// fieldType = IDENTIFIER ":" type ;
func (p *Parser) fieldType() Field {
	name := p.consume(IDENT, "expected identifier")
	p.consume(COLON, "expected `:`")
	typ := p.typ()
	return Field{Name: name.Lexeme, Exprs: []Node{typ}}
}

func (p *Parser) recover(err error) {
	p.err = errors.Join(p.err, err)
}

func (p Parser) peek() Token {
	return p.tokens[p.current]
}

func (p *Parser) advance() Token {
	if !p.IsAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p Parser) previous() Token {
	return p.tokens[p.current-1]
}

func (p Parser) IsAtEnd() bool {
	return p.peek().Kind == EOF
}

func (p Parser) match(kind TokenKind) bool {
	if p.IsAtEnd() {
		return false
	}
	return p.peek().Kind == kind
}

func (p *Parser) consume(kind TokenKind, message string) Token {
	if p.match(kind) {
		return p.advance()
	}

	p.err = errors.Join(p.err, parseError(p.peek(), message))
	return p.peek()
}

func parseError(t Token, message string) error {
	if t.Kind == EOF {
		return fmt.Errorf("at end: %s", message)
	}
	return fmt.Errorf("at %d: `%s`, %s", t.Line, t.Lexeme, message)
}
