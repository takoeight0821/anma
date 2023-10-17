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

func (p *Parser) Parse() (Node, error) {
	p.err = nil
	node := p.expr()
	return node, p.err
}

// expr = let | fn | assert ;
func (p *Parser) expr() Node {
	return p.let()
}

// let = "let" pattern "=" assert ;
// fn = "fn" pattern "{" expr (";" expr)* ";"? "}" ;
func (p *Parser) let() Node {
	if p.match(LET) {
		p.advance()
		pattern := p.pattern()
		p.consume(EQUAL, "expected `=`")
		expr := p.assert()
		return Let{Bind: pattern, Body: expr}
	}
	if p.match(FN) {
		p.advance()
		pattern := p.pattern()
		p.consume(LEFT_BRACE, "expected `{`")
		exprs := []Node{p.expr()}
		for p.match(SEMICOLON) {
			p.advance()
			if p.match(RIGHT_BRACE) {
				break
			}
			exprs = append(exprs, p.expr())
		}
		p.consume(RIGHT_BRACE, "expected `}`")
		return Lambda{Pattern: pattern, Exprs: exprs}
	}
	return p.assert()
}

// atom = var | literal | paren | codata ;
func (p *Parser) atom() Node {
	switch t := p.advance(); t.Kind {
	case IDENT:
		return Var{Name: t}
	case INTEGER, STRING:
		return Literal{Token: t}
	case LEFT_PAREN:
		if p.match(RIGHT_PAREN) {
			p.advance()
			return Paren{}
		}
		elems := []Node{p.expr()}
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHT_PAREN) {
				break
			}
			elems = append(elems, p.expr())
		}
		p.consume(RIGHT_PAREN, "expected `)`")
		return Paren{Elems: elems}
	case LEFT_BRACE:
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
	for p.match(LEFT_PAREN) {
		expr = p.finishCall(expr)
	}
	return expr
}

func (p *Parser) finishCall(fun Node) Call {
	p.consume(LEFT_PAREN, "expected `(`")
	args := []Node{}
	if !p.match(RIGHT_PAREN) {
		args = append(args, p.expr())
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHT_PAREN) {
				break
			}
			args = append(args, p.expr())
		}
	}
	p.consume(RIGHT_PAREN, "expected `)`")
	return Call{Func: fun, Args: args}
}

// codata = "{" clause ("," clause)* ","? "}" ;
func (p *Parser) codata() Codata {
	clauses := []Clause{p.clause()}
	for p.match(COMMA) {
		p.advance()
		if p.match(RIGHT_BRACE) {
			break
		}
		clauses = append(clauses, p.clause())
	}
	p.consume(RIGHT_BRACE, "expected `}`")
	return Codata{Clauses: clauses}
}

// clause = pattern "->" expr (";" expr)* ";"? ;
func (p *Parser) clause() Clause {
	pattern := p.pattern()
	p.consume(ARROW, "expected `->`")
	exprs := []Node{p.expr()}
	for p.match(SEMICOLON) {
		p.advance()
		if p.match(RIGHT_BRACE) {
			break
		}
		exprs = append(exprs, p.expr())
	}
	return Clause{Pattern: pattern, Exprs: exprs}
}

// pattern = accessPat ;
func (p *Parser) pattern() Node {
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
	for p.match(LEFT_PAREN) {
		pat = p.finishCallPat(pat)
	}
	return pat
}

func (p *Parser) finishCallPat(fun Node) Call {
	p.consume(LEFT_PAREN, "expected `(`")
	args := []Node{}
	if !p.match(RIGHT_PAREN) {
		args = append(args, p.pattern())
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHT_PAREN) {
				break
			}
			args = append(args, p.pattern())
		}
	}
	p.consume(RIGHT_PAREN, "expected `)`")
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
	case LEFT_PAREN:
		if p.match(RIGHT_PAREN) {
			p.advance()
			return Paren{}
		}
		patterns := []Node{p.pattern()}
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHT_PAREN) {
				break
			}
			patterns = append(patterns, p.pattern())
		}
		p.consume(RIGHT_PAREN, "expected `)`")
		return Paren{Elems: patterns}
	default:
		p.recover(parseError(t, "expected variable, literal, or parenthesized pattern"))
		return nil
	}
}

// type = binopType ;
func (p *Parser) typ() Node {
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
	for p.match(LEFT_PAREN) {
		typ = p.finishCallType(typ)
	}
	return typ
}

func (p *Parser) finishCallType(fun Node) Call {
	p.consume(LEFT_PAREN, "expected `(`")
	args := []Node{}
	if !p.match(RIGHT_PAREN) {
		args = append(args, p.typ())
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHT_PAREN) {
				break
			}
			args = append(args, p.typ())
		}
	}
	p.consume(RIGHT_PAREN, "expected `)`")
	return Call{Func: fun, Args: args}
}

// atomType = IDENT | "(" type ("," type)* ","? ")" ;
func (p *Parser) atomType() Node {
	switch t := p.advance(); t.Kind {
	case IDENT:
		return Var{Name: t}
	case LEFT_PAREN:
		if p.match(RIGHT_PAREN) {
			p.advance()
			return Paren{}
		}
		types := []Node{p.typ()}
		for p.match(COMMA) {
			p.advance()
			if p.match(RIGHT_PAREN) {
				break
			}
			types = append(types, p.typ())
		}
		p.consume(RIGHT_PAREN, "expected `)`")
		return Paren{Elems: types}
	default:
		p.recover(parseError(t, "expected variable or parenthesized type"))
		return nil
	}
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
