// Package parser is a parser for Anma language
package parser

import (
	"errors"

	"github.com/takoeight0821/anma/internal/ast"
	"github.com/takoeight0821/anma/internal/token"
	"github.com/takoeight0821/anma/internal/utils"
)

//go:generate go run ../../tools/main.go -comment -in parser.go -out ../../docs/syntax.ebnf

type Parser struct {
	tokens  []token.Token
	current int
	err     error
}

func NewParser(tokens []token.Token) *Parser {
	return &Parser{tokens, 0, nil}
}

// ParseExpr parses an expression.
func (p *Parser) ParseExpr() (ast.Node, error) {
	p.err = nil
	node := p.expr()
	return node, p.err
}

// ParseDecl parses declarations.
func (p *Parser) ParseDecl() ([]ast.Node, error) {
	p.err = nil
	nodes := []ast.Node{}
	for !p.IsAtEnd() {
		nodes = append(nodes, p.decl())
	}
	return nodes, p.err
}

// decl = typeDecl | varDecl | infixDecl ;
func (p *Parser) decl() ast.Node {
	if p.match(token.TYPE) {
		return p.typeDecl()
	}
	if p.match(token.DEF) {
		return p.varDecl()
	}
	return p.infixDecl()
}

// typeDecl = "type" token.IDENTIFIER "=" type ;
func (p *Parser) typeDecl() *ast.TypeDecl {
	p.consume(token.TYPE, "expected `type`")
	name := p.consume(token.IDENT, "expected identifier")
	p.consume(token.EQUAL, "expected `=`")
	typ := p.typ()
	return &ast.TypeDecl{Name: name, Type: typ}
}

// varDecl = "def" identifier "=" expr | "def" identifier ":" type | "def" identifier ":" type "=" expr ;
func (p *Parser) varDecl() *ast.VarDecl {
	p.consume(token.DEF, "expected `def`")
	name := p.consume(token.IDENT, "expected identifier")
	var typ ast.Node
	var expr ast.Node
	if p.match(token.COLON) {
		p.advance()
		typ = p.typ()
	}
	if p.match(token.EQUAL) {
		p.advance()
		expr = p.expr()
	}
	return &ast.VarDecl{Name: name, Type: typ, Expr: expr}
}

// infixDecl = ("infix" | "infixl" | "infixr") INTEGER token.OPERATOR ;
func (p *Parser) infixDecl() *ast.InfixDecl {
	kind := p.advance()
	if kind.Kind != token.INFIX && kind.Kind != token.INFIXL && kind.Kind != token.INFIXR {
		p.recover(utils.ErrorAt(kind, "expected `infix`, `infixl`, or `infixr`"))
		return &ast.InfixDecl{}
	}
	precedence := p.consume(token.INTEGER, "expected integer")
	name := p.consume(token.OPERATOR, "expected operator")
	return &ast.InfixDecl{Assoc: kind, Prec: precedence, Name: name}
}

// expr = let | fn | assert ;
func (p *Parser) expr() ast.Node {
	if p.IsAtEnd() {
		p.recover(utils.ErrorAt(p.peek(), "expected expression"))
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
func (p *Parser) let() *ast.Let {
	p.advance()
	pattern := p.pattern()
	p.consume(token.EQUAL, "expected `=`")
	expr := p.assert()
	return &ast.Let{Bind: pattern, Body: expr}
}

// fn = "fn" pattern "{" expr (";" expr)* ";"? "}" ;
func (p *Parser) fn() *ast.Lambda {
	p.advance()
	pattern := p.pattern()
	p.consume(token.LEFTBRACE, "expected `{`")
	exprs := []ast.Node{p.expr()}
	for p.match(token.SEMICOLON) {
		p.advance()
		if p.match(token.RIGHTBRACE) {
			break
		}
		exprs = append(exprs, p.expr())
	}
	p.consume(token.RIGHTBRACE, "expected `}`")
	return &ast.Lambda{Pattern: pattern, Exprs: exprs}
}

// atom = var | literal | paren | codata ;
func (p *Parser) atom() ast.Node {
	switch t := p.advance(); t.Kind {
	case token.IDENT:
		return &ast.Var{Name: t}
	case token.INTEGER, token.STRING:
		return &ast.Literal{Token: t}
	case token.LEFTPAREN:
		if p.match(token.RIGHTPAREN) {
			p.advance()
			return &ast.Paren{}
		}
		elems := []ast.Node{p.expr()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTPAREN) {
				break
			}
			elems = append(elems, p.expr())
		}
		p.consume(token.RIGHTPAREN, "expected `)`")
		return &ast.Paren{Elems: elems}
	case token.LEFTBRACE:
		return p.codata()
	default:
		p.recover(utils.ErrorAt(t, "expected variable, literal, or parenthesized expression"))
		return nil
	}
}

// assert = binop (":" type)* ;
func (p *Parser) assert() ast.Node {
	expr := p.binary()
	for p.match(token.COLON) {
		p.advance()
		typ := p.typ()
		expr = &ast.Assert{Expr: expr, Type: typ}
	}
	return expr
}

// binary = access (operator access)* ;
func (p *Parser) binary() ast.Node {
	expr := p.access()
	for p.match(token.OPERATOR) {
		op := p.advance()
		right := p.access()
		expr = &ast.Binary{Left: expr, Op: op, Right: right}
	}
	return expr
}

// access = call ("." token.IDENTIFIER)* ;
func (p *Parser) access() ast.Node {
	expr := p.call()
	for p.match(token.DOT) {
		p.advance()
		name := p.consume(token.IDENT, "expected identifier")
		expr = &ast.Access{Receiver: expr, Name: name}
	}
	return expr
}

// call = atom ("(" ")" | "(" expr ("," expr)* ","? ")")* ;
func (p *Parser) call() ast.Node {
	expr := p.atom()
	for p.match(token.LEFTPAREN) {
		expr = p.finishCall(expr)
	}
	return expr
}

func (p *Parser) finishCall(fun ast.Node) *ast.Call {
	p.consume(token.LEFTPAREN, "expected `(`")
	args := []ast.Node{}
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
	return &ast.Call{Func: fun, Args: args}
}

// codata = "{" clause ("," clause)* ","? "}" ;
func (p *Parser) codata() *ast.Codata {
	clauses := []*ast.Clause{p.clause()}
	for p.match(token.COMMA) {
		p.advance()
		if p.match(token.RIGHTBRACE) {
			break
		}
		clauses = append(clauses, p.clause())
	}
	p.consume(token.RIGHTBRACE, "expected `}`")
	return &ast.Codata{Clauses: clauses}
}

// clause = pattern "->" expr (";" expr)* ";"? ;
func (p *Parser) clause() *ast.Clause {
	pattern := p.pattern()
	p.consume(token.ARROW, "expected `->`")
	exprs := []ast.Node{p.expr()}
	for p.match(token.SEMICOLON) {
		p.advance()
		if p.match(token.RIGHTBRACE) {
			break
		}
		exprs = append(exprs, p.expr())
	}
	return &ast.Clause{Pattern: pattern, Exprs: exprs}
}

// pattern = accessPat ;
func (p *Parser) pattern() ast.Node {
	if p.IsAtEnd() {
		p.recover(utils.ErrorAt(p.peek(), "expected pattern"))
		return nil
	}
	return p.accessPat()
}

// accessPat = callPat ("." token.IDENTIFIER)* ;
func (p *Parser) accessPat() ast.Node {
	pat := p.callPat()
	for p.match(token.DOT) {
		p.advance()
		name := p.consume(token.IDENT, "expected identifier")
		pat = &ast.Access{Receiver: pat, Name: name}
	}
	return pat
}

// callPat = atomPat ("(" ")" | "(" pattern ("," pattern)* ","? ")")* ;
func (p *Parser) callPat() ast.Node {
	pat := p.atomPat()
	for p.match(token.LEFTPAREN) {
		pat = p.finishCallPat(pat)
	}
	return pat
}

func (p *Parser) finishCallPat(fun ast.Node) *ast.Call {
	p.consume(token.LEFTPAREN, "expected `(`")
	args := []ast.Node{}
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
	return &ast.Call{Func: fun, Args: args}
}

// atomPat = token.IDENT | INTEGER | STRING | "(" pattern ("," pattern)* ","? ")" ;
func (p *Parser) atomPat() ast.Node {
	switch t := p.advance(); t.Kind {
	case token.SHARP:
		return &ast.This{Token: t}
	case token.IDENT:
		return &ast.Var{Name: t}
	case token.INTEGER, token.STRING:
		return &ast.Literal{Token: t}
	case token.LEFTPAREN:
		if p.match(token.RIGHTPAREN) {
			p.advance()
			return &ast.Paren{}
		}
		patterns := []ast.Node{p.pattern()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTPAREN) {
				break
			}
			patterns = append(patterns, p.pattern())
		}
		p.consume(token.RIGHTPAREN, "expected `)`")
		return &ast.Paren{Elems: patterns}
	default:
		p.recover(utils.ErrorAt(t, "expected variable, literal, or parenthesized pattern"))
		return nil
	}
}

// type = binopType ;
func (p *Parser) typ() ast.Node {
	if p.IsAtEnd() {
		p.recover(utils.ErrorAt(p.peek(), "expected type"))
		return nil
	}
	return p.binopType()
}

// binopType = callType (operator callType)* ;
func (p *Parser) binopType() ast.Node {
	typ := p.callType()
	for p.match(token.OPERATOR) || p.match(token.ARROW) {
		op := p.advance()
		right := p.callType()
		typ = &ast.Binary{Left: typ, Op: op, Right: right}
	}
	return typ
}

// callType = atomType ("(" ")" | "(" type ("," type)* ","? ")")* ;
func (p *Parser) callType() ast.Node {
	typ := p.atomType()
	for p.match(token.LEFTPAREN) {
		typ = p.finishCallType(typ)
	}
	return typ
}

func (p *Parser) finishCallType(fun ast.Node) *ast.Call {
	p.consume(token.LEFTPAREN, "expected `(`")
	args := []ast.Node{}
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
	return &ast.Call{Func: fun, Args: args}
}

// atomType = token.IDENT | "{" fieldType ("," fieldType)* ","? "}" | "(" type ("," type)* ","? ")" ;
func (p *Parser) atomType() ast.Node {
	switch t := p.advance(); t.Kind {
	case token.IDENT:
		return &ast.Var{Name: t}
	case token.LEFTBRACE:
		fields := []*ast.Field{p.fieldType()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTBRACE) {
				break
			}
			fields = append(fields, p.fieldType())
		}
		p.consume(token.RIGHTBRACE, "expected `}`")
		return &ast.Object{Fields: fields}
	case token.LEFTPAREN:
		if p.match(token.RIGHTPAREN) {
			p.advance()
			return &ast.Paren{}
		}
		types := []ast.Node{p.typ()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTPAREN) {
				break
			}
			types = append(types, p.typ())
		}
		p.consume(token.RIGHTPAREN, "expected `)`")
		return &ast.Paren{Elems: types}
	default:
		p.recover(utils.ErrorAt(t, "expected variable or parenthesized type"))
		return nil
	}
}

// fieldType = token.IDENTIFIER ":" type ;
func (p *Parser) fieldType() *ast.Field {
	name := p.consume(token.IDENT, "expected identifier")
	p.consume(token.COLON, "expected `:`")
	typ := p.typ()
	return &ast.Field{Name: name.Lexeme, Exprs: []ast.Node{typ}}
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

	p.err = errors.Join(p.err, utils.ErrorAt(p.peek(), message))
	return p.peek()
}
