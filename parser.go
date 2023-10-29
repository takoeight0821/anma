package main

import (
	"errors"

	"github.com/takoeight0821/anma/internal/token"
)

//go:generate go run ./tools/main.go -comment -in parser.go -out docs/syntax.ebnf

type Parser struct {
	tokens  []token.Token
	current int
	err     error
}

func NewParser(tokens []token.Token) *Parser {
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
	if p.match(token.TYPE) {
		return p.typeDecl()
	}
	if p.match(token.DEF) {
		return p.varDecl()
	}
	return p.infixDecl()
}

// typeDecl = "type" token.IDENTIFIER "=" type ;
func (p *Parser) typeDecl() *TypeDecl {
	p.consume(token.TYPE, "expected `type`")
	name := p.consume(token.IDENT, "expected identifier")
	p.consume(token.EQUAL, "expected `=`")
	typ := p.typ()
	return &TypeDecl{Name: name, Type: typ}
}

// varDecl = "def" identifier "=" expr | "def" identifier ":" type | "def" identifier ":" type "=" expr ;
func (p *Parser) varDecl() *VarDecl {
	p.consume(token.DEF, "expected `def`")
	name := p.consume(token.IDENT, "expected identifier")
	var typ Node
	var expr Node
	if p.match(token.COLON) {
		p.advance()
		typ = p.typ()
	}
	if p.match(token.EQUAL) {
		p.advance()
		expr = p.expr()
	}
	return &VarDecl{Name: name, Type: typ, Expr: expr}
}

// infixDecl = ("infix" | "infixl" | "infixr") INTEGER token.OPERATOR ;
func (p *Parser) infixDecl() *InfixDecl {
	kind := p.advance()
	if kind.Kind != token.INFIX && kind.Kind != token.INFIXL && kind.Kind != token.INFIXR {
		p.recover(errorAt(kind, "expected `infix`, `infixl`, or `infixr`"))
		return &InfixDecl{}
	}
	precedence := p.consume(token.INTEGER, "expected integer")
	name := p.consume(token.OPERATOR, "expected operator")
	return &InfixDecl{Assoc: kind, Prec: precedence, Name: name}
}

// expr = let | fn | assert ;
func (p *Parser) expr() Node {
	if p.IsAtEnd() {
		p.recover(errorAt(p.peek(), "expected expression"))
		return nil
	}
	if p.match(token.LET) {
		return p.let()
	}
	if p.match(token.FN) {
		return p.fn()
	}
	return p.assert()
}

// let = "let" pattern "=" assert ;
func (p *Parser) let() *Let {
	p.advance()
	pattern := p.pattern()
	p.consume(token.EQUAL, "expected `=`")
	expr := p.assert()
	return &Let{Bind: pattern, Body: expr}
}

// fn = "fn" pattern "{" expr (";" expr)* ";"? "}" ;
func (p *Parser) fn() *Lambda {
	p.advance()
	pattern := p.pattern()
	p.consume(token.LEFTBRACE, "expected `{`")
	exprs := []Node{p.expr()}
	for p.match(token.SEMICOLON) {
		p.advance()
		if p.match(token.RIGHTBRACE) {
			break
		}
		exprs = append(exprs, p.expr())
	}
	p.consume(token.RIGHTBRACE, "expected `}`")
	return &Lambda{Pattern: pattern, Exprs: exprs}
}

// atom = var | literal | paren | codata ;
func (p *Parser) atom() Node {
	switch t := p.advance(); t.Kind {
	case token.IDENT:
		return &Var{Name: t}
	case token.INTEGER, token.STRING:
		return &Literal{Token: t}
	case token.LEFTPAREN:
		if p.match(token.RIGHTPAREN) {
			p.advance()
			return &Paren{}
		}
		elems := []Node{p.expr()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTPAREN) {
				break
			}
			elems = append(elems, p.expr())
		}
		p.consume(token.RIGHTPAREN, "expected `)`")
		return &Paren{Elems: elems}
	case token.LEFTBRACE:
		return p.codata()
	default:
		p.recover(errorAt(t, "expected variable, literal, or parenthesized expression"))
		return nil
	}
}

// assert = binop (":" type)* ;
func (p *Parser) assert() Node {
	expr := p.binary()
	for p.match(token.COLON) {
		p.advance()
		typ := p.typ()
		expr = &Assert{Expr: expr, Type: typ}
	}
	return expr
}

// binary = access (operator access)* ;
func (p *Parser) binary() Node {
	expr := p.access()
	for p.match(token.OPERATOR) {
		op := p.advance()
		right := p.access()
		expr = &Binary{Left: expr, Op: op, Right: right}
	}
	return expr
}

// access = call ("." token.IDENTIFIER)* ;
func (p *Parser) access() Node {
	expr := p.call()
	for p.match(token.DOT) {
		p.advance()
		name := p.consume(token.IDENT, "expected identifier")
		expr = &Access{Receiver: expr, Name: name}
	}
	return expr
}

// call = atom ("(" ")" | "(" expr ("," expr)* ","? ")")* ;
func (p *Parser) call() Node {
	expr := p.atom()
	for p.match(token.LEFTPAREN) {
		expr = p.finishCall(expr)
	}
	return expr
}

func (p *Parser) finishCall(fun Node) *Call {
	p.consume(token.LEFTPAREN, "expected `(`")
	args := []Node{}
	if !p.match(token.RIGHTPAREN) {
		args = append(args, p.expr())
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTPAREN) {
				break
			}
			args = append(args, p.expr())
		}
	}
	p.consume(token.RIGHTPAREN, "expected `)`")
	return &Call{Func: fun, Args: args}
}

// codata = "{" clause ("," clause)* ","? "}" ;
func (p *Parser) codata() *Codata {
	clauses := []*Clause{p.clause()}
	for p.match(token.COMMA) {
		p.advance()
		if p.match(token.RIGHTBRACE) {
			break
		}
		clauses = append(clauses, p.clause())
	}
	p.consume(token.RIGHTBRACE, "expected `}`")
	return &Codata{Clauses: clauses}
}

// clause = pattern "->" expr (";" expr)* ";"? ;
func (p *Parser) clause() *Clause {
	pattern := p.pattern()
	p.consume(token.ARROW, "expected `->`")
	exprs := []Node{p.expr()}
	for p.match(token.SEMICOLON) {
		p.advance()
		if p.match(token.RIGHTBRACE) {
			break
		}
		exprs = append(exprs, p.expr())
	}
	return &Clause{Pattern: pattern, Exprs: exprs}
}

// pattern = accessPat ;
func (p *Parser) pattern() Node {
	if p.IsAtEnd() {
		p.recover(errorAt(p.peek(), "expected pattern"))
		return nil
	}
	return p.accessPat()
}

// accessPat = callPat ("." token.IDENTIFIER)* ;
func (p *Parser) accessPat() Node {
	pat := p.callPat()
	for p.match(token.DOT) {
		p.advance()
		name := p.consume(token.IDENT, "expected identifier")
		pat = &Access{Receiver: pat, Name: name}
	}
	return pat
}

// callPat = atomPat ("(" ")" | "(" pattern ("," pattern)* ","? ")")* ;
func (p *Parser) callPat() Node {
	pat := p.atomPat()
	for p.match(token.LEFTPAREN) {
		pat = p.finishCallPat(pat)
	}
	return pat
}

func (p *Parser) finishCallPat(fun Node) *Call {
	p.consume(token.LEFTPAREN, "expected `(`")
	args := []Node{}
	if !p.match(token.RIGHTPAREN) {
		args = append(args, p.pattern())
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTPAREN) {
				break
			}
			args = append(args, p.pattern())
		}
	}
	p.consume(token.RIGHTPAREN, "expected `)`")
	return &Call{Func: fun, Args: args}
}

// atomPat = token.IDENT | INTEGER | STRING | "(" pattern ("," pattern)* ","? ")" ;
func (p *Parser) atomPat() Node {
	switch t := p.advance(); t.Kind {
	case token.SHARP:
		return &This{Token: t}
	case token.IDENT:
		return &Var{Name: t}
	case token.INTEGER, token.STRING:
		return &Literal{Token: t}
	case token.LEFTPAREN:
		if p.match(token.RIGHTPAREN) {
			p.advance()
			return &Paren{}
		}
		patterns := []Node{p.pattern()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTPAREN) {
				break
			}
			patterns = append(patterns, p.pattern())
		}
		p.consume(token.RIGHTPAREN, "expected `)`")
		return &Paren{Elems: patterns}
	default:
		p.recover(errorAt(t, "expected variable, literal, or parenthesized pattern"))
		return nil
	}
}

// type = binopType ;
func (p *Parser) typ() Node {
	if p.IsAtEnd() {
		p.recover(errorAt(p.peek(), "expected type"))
		return nil
	}
	return p.binopType()
}

// binopType = callType (operator callType)* ;
func (p *Parser) binopType() Node {
	typ := p.callType()
	for p.match(token.OPERATOR) || p.match(token.ARROW) {
		op := p.advance()
		right := p.callType()
		typ = &Binary{Left: typ, Op: op, Right: right}
	}
	return typ
}

// callType = atomType ("(" ")" | "(" type ("," type)* ","? ")")* ;
func (p *Parser) callType() Node {
	typ := p.atomType()
	for p.match(token.LEFTPAREN) {
		typ = p.finishCallType(typ)
	}
	return typ
}

func (p *Parser) finishCallType(fun Node) *Call {
	p.consume(token.LEFTPAREN, "expected `(`")
	args := []Node{}
	if !p.match(token.RIGHTPAREN) {
		args = append(args, p.typ())
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTPAREN) {
				break
			}
			args = append(args, p.typ())
		}
	}
	p.consume(token.RIGHTPAREN, "expected `)`")
	return &Call{Func: fun, Args: args}
}

// atomType = token.IDENT | "{" fieldType ("," fieldType)* ","? "}" | "(" type ("," type)* ","? ")" ;
func (p *Parser) atomType() Node {
	switch t := p.advance(); t.Kind {
	case token.IDENT:
		return &Var{Name: t}
	case token.LEFTBRACE:
		fields := []*Field{p.fieldType()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTBRACE) {
				break
			}
			fields = append(fields, p.fieldType())
		}
		p.consume(token.RIGHTBRACE, "expected `}`")
		return &Object{Fields: fields}
	case token.LEFTPAREN:
		if p.match(token.RIGHTPAREN) {
			p.advance()
			return &Paren{}
		}
		types := []Node{p.typ()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTPAREN) {
				break
			}
			types = append(types, p.typ())
		}
		p.consume(token.RIGHTPAREN, "expected `)`")
		return &Paren{Elems: types}
	default:
		p.recover(errorAt(t, "expected variable or parenthesized type"))
		return nil
	}
}

// fieldType = token.IDENTIFIER ":" type ;
func (p *Parser) fieldType() *Field {
	name := p.consume(token.IDENT, "expected identifier")
	p.consume(token.COLON, "expected `:`")
	typ := p.typ()
	return &Field{Name: name.Lexeme, Exprs: []Node{typ}}
}

func (p *Parser) recover(err error) {
	p.err = errors.Join(p.err, err)
}

func (p Parser) peek() token.Token {
	return p.tokens[p.current]
}

func (p *Parser) advance() token.Token {
	if !p.IsAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p Parser) previous() token.Token {
	return p.tokens[p.current-1]
}

func (p Parser) IsAtEnd() bool {
	return p.peek().Kind == token.EOF
}

func (p Parser) match(kind token.TokenKind) bool {
	if p.IsAtEnd() {
		return false
	}
	return p.peek().Kind == kind
}

func (p *Parser) consume(kind token.TokenKind, message string) token.Token {
	if p.match(kind) {
		return p.advance()
	}

	p.err = errors.Join(p.err, errorAt(p.peek(), message))
	return p.peek()
}
