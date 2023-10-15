package main

import (
	"errors"
	"fmt"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
)

type Parser struct {
	tokens  []token.Token
	current int
	err     error
}

func NewParser(tokens []token.Token) *Parser {
	return &Parser{tokens, 0, nil}
}

func (p *Parser) Parse() (ast.Node, error) {
	p.err = nil
	node := p.expr()
	return node, p.err
}

// expr = let
func (p *Parser) expr() ast.Node {
	return p.let()
}

// let = "let" pattern "=" assert | "fn" pattern "{" expr (";" expr)* ";"? "}" | assert
func (p *Parser) let() ast.Node {
	if p.match(token.LET) {
		p.advance()
		pattern := p.pattern()
		p.consume(token.EQUAL, "expected `=`")
		expr := p.assert()
		return ast.Let{pattern, expr}
	}
	if p.match(token.FN) {
		p.advance()
		pattern := p.pattern()
		p.consume(token.LEFT_BRACE, "expected `{`")
		exprs := []ast.Node{p.expr()}
		for p.match(token.SEMICOLON) {
			p.advance()
			if p.match(token.RIGHT_BRACE) {
				break
			}
			exprs = append(exprs, p.expr())
		}
		p.consume(token.RIGHT_BRACE, "expected `}`")
		return ast.Lambda{pattern, exprs}
	}
	return p.assert()
}

// assert = binop (":" type)*
func (p *Parser) assert() ast.Node {
	expr := p.binop()
	for p.match(token.COLON) {
		p.advance()
		typ := p.typ()
		expr = ast.Assert{expr, typ}
	}
	return expr
}

// binop = access (operator access)*
func (p *Parser) binop() ast.Node {
	expr := p.access()
	for p.match(token.OPERATOR) {
		op := p.advance()
		right := p.access()
		expr = ast.Binary{expr, op, right}
	}
	return expr
}

// access = call ("." IDENTIFIER)*
func (p *Parser) access() ast.Node {
	expr := p.call()
	for p.match(token.DOT) {
		p.advance()
		name := p.consume(token.IDENT, "expected identifier")
		expr = ast.Access{expr, name}
	}
	return expr
}

// call = atom finishCall*
func (p *Parser) call() ast.Node {
	expr := p.atom()
	for p.match(token.LEFT_PAREN) {
		expr = p.finishCall(expr)
	}
	return expr
}

// finishCall = "(" ")" | "(" expr ("," expr)* ","? ")"
func (p *Parser) finishCall(fun ast.Node) ast.Node {
	p.consume(token.LEFT_PAREN, "expected `(`")
	args := []ast.Node{}
	if !p.match(token.RIGHT_PAREN) {
		args = append(args, p.expr())
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHT_PAREN) {
				break
			}
			args = append(args, p.expr())
		}
	}
	p.consume(token.RIGHT_PAREN, "expected `)`")
	return ast.Call{fun, args}
}

// atom = IDENT | INTEGER | STRING | codata | "(" expr ("," expr)* ","? ")" | "(" ")"
func (p *Parser) atom() ast.Node {
	switch t := p.advance(); t.Kind {
	case token.IDENT:
		return ast.Var{t}
	case token.INTEGER, token.STRING:
		return ast.Literal{t}
	case token.LEFT_BRACE:
		return p.codata()
	case token.LEFT_PAREN:
		if p.match(token.RIGHT_PAREN) {
			p.advance()
			return ast.Paren{}
		}
		elems := []ast.Node{p.expr()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHT_PAREN) {
				break
			}
			elems = append(elems, p.expr())
		}
		p.consume(token.RIGHT_PAREN, "expected `)`")
		return ast.Paren{elems}
	default:
		p.recover(parseError(t, "expected variable, literal, or parenthesized expression"))
		return nil
	}
}

// codata = "{" clause ("," clause)* ","? "}"
func (p *Parser) codata() ast.Node {
	clauses := []ast.Clause{p.clause()}
	for p.match(token.COMMA) {
		p.advance()
		if p.match(token.RIGHT_BRACE) {
			break
		}
		clauses = append(clauses, p.clause())
	}
	p.consume(token.RIGHT_BRACE, "expected `}`")
	return ast.Codata{clauses}
}

// clause = pattern "->" expr (";" expr)* ";"?
func (p *Parser) clause() ast.Clause {
	pattern := p.pattern()
	p.consume(token.ARROW, "expected `->`")
	exprs := []ast.Node{p.expr()}
	for p.match(token.SEMICOLON) {
		p.advance()
		if p.match(token.RIGHT_BRACE) {
			break
		}
		exprs = append(exprs, p.expr())
	}
	return ast.Clause{pattern, exprs}
}

// pattern = accessPat
func (p *Parser) pattern() ast.Node {
	return p.accessPat()
}

// accessPat = callPat ("." IDENTIFIER)*
func (p *Parser) accessPat() ast.Node {
	pat := p.callPat()
	for p.match(token.DOT) {
		p.advance()
		name := p.consume(token.IDENT, "expected identifier")
		pat = ast.Access{pat, name}
	}
	return pat
}

// callPat = atomPat finishCalltoken.Pat*
func (p *Parser) callPat() ast.Node {
	pat := p.atomPat()
	for p.match(token.LEFT_PAREN) {
		pat = p.finishCallPat(pat)
	}
	return pat
}

// finishCallPat = "(" ")" | "(" pattern ("," pattern)* ","? ")"
func (p *Parser) finishCallPat(fun ast.Node) ast.Node {
	p.consume(token.LEFT_PAREN, "expected `(`")
	args := []ast.Node{}
	if !p.match(token.RIGHT_PAREN) {
		args = append(args, p.pattern())
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHT_PAREN) {
				break
			}
			args = append(args, p.pattern())
		}
	}
	p.consume(token.RIGHT_PAREN, "expected `)`")
	return ast.Call{fun, args}
}

// atomPat = IDENT | INTEGER | STRING | "(" pattern ")"
func (p *Parser) atomPat() ast.Node {
	switch t := p.advance(); t.Kind {
	case token.SHARP:
		return ast.This{t}
	case token.IDENT:
		return ast.Var{t}
	case token.INTEGER, token.STRING:
		return ast.Literal{t}
	case token.LEFT_PAREN:
		if p.match(token.RIGHT_PAREN) {
			p.advance()
			return ast.Paren{}
		}
		patterns := []ast.Node{p.pattern()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHT_PAREN) {
				break
			}
			patterns = append(patterns, p.pattern())
		}
		p.consume(token.RIGHT_PAREN, "expected `)`")
		return ast.Paren{patterns}
	default:
		p.recover(parseError(t, "expected variable, literal, or parenthesized pattern"))
		return nil
	}
}

// type = binopType
func (p *Parser) typ() ast.Node {
	return p.binopType()
}

// binopType = callType (operator callType)*
func (p *Parser) binopType() ast.Node {
	typ := p.callType()
	for p.match(token.OPERATOR) {
		op := p.advance()
		right := p.callType()
		typ = ast.Binary{typ, op, right}
	}
	return typ
}

// callType = atomType finishCallType*
func (p *Parser) callType() ast.Node {
	typ := p.atomType()
	for p.match(token.LEFT_PAREN) {
		typ = p.finishCallType(typ)
	}
	return typ
}

// finishCallType = "(" ")" | "(" type ("," type)* ","? ")"
func (p *Parser) finishCallType(fun ast.Node) ast.Node {
	p.consume(token.LEFT_PAREN, "expected `(`")
	args := []ast.Node{}
	if !p.match(token.RIGHT_PAREN) {
		args = append(args, p.typ())
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHT_PAREN) {
				break
			}
			args = append(args, p.typ())
		}
	}
	p.consume(token.RIGHT_PAREN, "expected `)`")
	return ast.Call{fun, args}
}

// atomType = IDENT | "(" type ("," type)* ","? ")"
func (p *Parser) atomType() ast.Node {
	switch t := p.advance(); t.Kind {
	case token.IDENT:
		return ast.Var{t}
	case token.LEFT_PAREN:
		if p.match(token.RIGHT_PAREN) {
			p.advance()
			return ast.Paren{}
		}
		types := []ast.Node{p.typ()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHT_PAREN) {
				break
			}
			types = append(types, p.typ())
		}
		p.consume(token.RIGHT_PAREN, "expected `)`")
		return ast.Paren{types}
	default:
		p.recover(parseError(t, "expected variable or parenthesized type"))
		return nil
	}
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

func (p Parser) match(kind token.Kind) bool {
	if p.IsAtEnd() {
		return false
	}
	return p.peek().Kind == kind
}

func (p *Parser) consume(kind token.Kind, message string) token.Token {
	if p.match(kind) {
		return p.advance()
	}

	p.err = errors.Join(p.err, parseError(p.peek(), message))
	return p.peek()
}

func parseError(t token.Token, message string) error {
	if t.Kind == token.EOF {
		return fmt.Errorf("at end: %s", message)
	}
	return fmt.Errorf("at %d: `%s`, %s", t.Line, t.Lexeme, message)
}
