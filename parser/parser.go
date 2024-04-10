package parser

import (
	"errors"
	"fmt"
	"os"

	"github.com/takoeight0821/anma/ast"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

//go:generate go run ../tools/main.go -comment -in parser.go -out ../docs/syntax.ebnf

type Parser struct {
	tokens  []token.Token
	current int
	err     error
}

func NewParser(tokens []token.Token) *Parser {
	return &Parser{tokens, 0, nil}
}

func (p *Parser) ParseExpr() (ast.Node, error) {
	p.err = nil
	node := p.expr()

	return node, p.err
}

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

// typeDecl = "type" IDENT (typeparams1)? "=" typebody ;
// typeparams1 = "(" IDENT ("," IDENT)* ","? ")" ;
// typebody = "{" constructor ("," constructor)* ","? "}" | type ;
func (p *Parser) typeDecl() *ast.TypeDecl {
	p.consume(token.TYPE)
	typename := p.consume(token.IDENT)

	var def ast.Node
	def = &ast.Var{Name: typename}
	if p.match(token.LEFTPAREN) {
		p.consume(token.LEFTPAREN)
		typeparams := []ast.Node{}
		if !p.match(token.RIGHTPAREN) {
			typeparams = append(typeparams, &ast.Var{Name: p.consume(token.IDENT)})
			for p.match(token.COMMA) {
				p.advance()
				if p.match(token.RIGHTPAREN) {
					break
				}
				typeparams = append(typeparams, &ast.Var{Name: p.consume(token.IDENT)})
			}
		}
		p.consume(token.RIGHTPAREN)
		def = &ast.Call{Func: def, Args: typeparams}
	}

	p.consume(token.EQUAL)
	var types []ast.Node
	if p.match(token.LEFTBRACE) {
		// if typebody is a record, then call p.typ()
		if p.matchNth(1, token.IDENT) && p.matchNth(2, token.COLON) {
			types = append(types, p.typ())
		} else {
			p.consume(token.LEFTBRACE)
			for !p.match(token.RIGHTBRACE) {
				types = append(types, p.constructor())
				if p.match(token.COMMA) {
					p.advance()
				}
			}
			p.consume(token.RIGHTBRACE)
		}
	} else {
		types = append(types, p.typ())
	}

	return &ast.TypeDecl{Def: def, Types: types}
}

// constructor = IDENT "(" typeparams ")" ;
// typeparams = (type ("," type)*)? ;
func (p *Parser) constructor() *ast.Call {
	name := p.consume(token.IDENT)
	p.consume(token.LEFTPAREN)
	typeparams := []ast.Node{}
	if !p.match(token.RIGHTPAREN) {
		typeparams = append(typeparams, p.typ())
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTPAREN) {
				break
			}
			typeparams = append(typeparams, p.typ())
		}
	}
	p.consume(token.RIGHTPAREN)

	return &ast.Call{Func: &ast.Var{Name: name}, Args: typeparams}
}

// varDecl = "def" IDENT "=" expr | "def" IDENT ":" type | "def" IDENT ":" type "=" expr ;
func (p *Parser) varDecl() *ast.VarDecl {
	p.consume(token.DEF)
	var name token.Token
	switch {
	case p.match(token.IDENT):
		name = p.advance()
	case p.match(token.OPERATOR):
		name = p.advance()
	default:
		p.recover(unexpectedToken(p.peek(), "identifier", "operator"))

		return &ast.VarDecl{}
	}
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

// infixDecl = ("infix" | "infixl" | "infixr") INTEGER OPERATOR ;
func (p *Parser) infixDecl() *ast.InfixDecl {
	kind := p.advance()
	if kind.Kind != token.INFIX && kind.Kind != token.INFIXL && kind.Kind != token.INFIXR {
		p.recover(unexpectedToken(p.peek(), "`infix`", "`infixl`", "`infixr`"))

		return nil
	}
	precedence := p.consume(token.INTEGER)
	name := p.consume(token.OPERATOR)

	return &ast.InfixDecl{Assoc: kind, Prec: precedence, Name: name}
}

// expr = let | with | assert ;
func (p *Parser) expr() ast.Node {
	if p.IsAtEnd() {
		p.recover(unexpectedToken(p.peek(), "expression"))

		return nil
	}
	if p.match(token.LET) {
		return p.let()
	}
	if p.match(token.WITH) {
		return p.with()
	}

	return p.assert()
}

// let = "let" pattern "=" assert ;
func (p *Parser) let() *ast.Let {
	p.advance()
	pattern := p.pattern()
	p.consume(token.EQUAL)
	expr := p.assert()

	return &ast.Let{Bind: pattern, Body: expr}
}

// with = "with" pattern "<-" assert | "with" assert ;
func (p *Parser) with() *ast.With {
	p.advance()

	patterns, err := try(p, func() []ast.Node {
		pattern := p.pattern()
		p.consume(token.BACKARROW)

		return []ast.Node{pattern}
	}, func() []ast.Node {
		return []ast.Node{}
	})

	expr := p.assert()

	if _, ok := expr.(*ast.Call); !ok {
		fmt.Fprintf(os.Stderr, "at %d: `%s`, warning: `with` expression should be a function call\n", expr.Base().Line, expr.Base().Lexeme)
	}

	if p.err != nil {
		p.recover(err)
	}

	return &ast.With{Binds: patterns, Body: expr}
}

// atom = var | literal | paren | codata | PRIM "(" IDENT ("," expr)* ","? ")" ;
// var = IDENT ;
// literal = INTEGER | STRING ;
// paren = "(" expr ")" ;
// codata = "{" clause ("," clause)* ","? "}" ;
func (p *Parser) atom() ast.Node {
	//exhaustive:ignore
	switch tok := p.advance(); tok.Kind {
	case token.IDENT:
		return &ast.Var{Name: tok}
	case token.INTEGER, token.STRING:
		return &ast.Literal{Token: tok}
	case token.LEFTPAREN:
		expr := p.expr()
		p.consume(token.RIGHTPAREN)

		return &ast.Paren{Expr: expr}
	case token.LEFTBRACE:
		return p.codata()
	case token.PRIM:
		p.consume(token.LEFTPAREN)
		name := p.consume(token.IDENT)
		args := []ast.Node{}
		if !p.match(token.RIGHTPAREN) {
			for p.match(token.COMMA) {
				p.advance()
				if p.match(token.RIGHTPAREN) {
					break
				}
				args = append(args, p.expr())
			}
		}
		p.consume(token.RIGHTPAREN)

		return &ast.Prim{Name: name, Args: args}
	default:
		p.recover(unexpectedToken(tok, "identifier", "integer", "string", "`(`", "`{`"))

		return nil
	}
}

// assert = binary (":" type)* ;
func (p *Parser) assert() ast.Node {
	expr := p.binary()
	for p.match(token.COLON) {
		p.advance()
		typ := p.typ()
		expr = &ast.Assert{Expr: expr, Type: typ}
	}

	return expr
}

// binary = method (operator method)* ;
func (p *Parser) binary() ast.Node {
	expr := p.method()
	for p.match(token.OPERATOR) {
		op := p.advance()
		right := p.method()
		expr = &ast.Binary{Left: expr, Op: op, Right: right}
	}

	return expr
}

// method = atom (accessTail | callTail)* ;
func (p *Parser) method() ast.Node {
	expr := p.atom()
	for {
		switch {
		case p.match(token.DOT):
			expr = p.accessTail(expr)
		case p.match(token.LEFTPAREN):
			expr = p.callTail(expr)
		default:
			return expr
		}
	}
}

// accessTail = "." IDENT callTail? ;
func (p *Parser) accessTail(receiver ast.Node) ast.Node {
	p.consume(token.DOT)
	name := p.consume(token.IDENT)
	expr := &ast.Access{Receiver: receiver, Name: name}

	if p.match(token.LEFTPAREN) {
		return p.callTail(expr)
	}

	return expr
}

// callTail = "(" ")" | "(" expr ("," expr)* ","? ")" ;
func (p *Parser) callTail(fun ast.Node) ast.Node {
	p.consume(token.LEFTPAREN)
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
	p.consume(token.RIGHTPAREN)

	return &ast.Call{Func: fun, Args: args}
}

// codata = "{" clause ("," clause)* ","? "}" ;
func (p *Parser) codata() *ast.Codata {
	clauses := []*ast.CodataClause{p.clause()}
	for p.match(token.COMMA) {
		p.advance()
		if p.match(token.RIGHTBRACE) {
			break
		}
		clauses = append(clauses, p.clause())
	}
	p.consume(token.RIGHTBRACE)

	return &ast.Codata{Clauses: clauses}
}

// clause = clauseHead "->" clauseBody | clauseBody ;
// clauseHead = "(" ")" | "(" pattern ("," pattern)* ","? ")" | pattern ;
// clauseBody = expr (";" expr)* ";"? ;
func (p *Parser) clause() *ast.CodataClause {
	// try to parse `clauseHead "->"`
	pattern, patternErr := try(p, func() ast.Node {
		var pattern ast.Node
		if p.match(token.SHARP) {
			// if the first token is `#`, then it is a pattern.
			pattern = p.pattern()
		} else if p.match(token.LEFTPAREN) {
			// if the first token is `(`, then it is a pattern list as parameters.
			tok := p.consume(token.LEFTPAREN)
			params := []ast.Node{}
			if !p.match(token.RIGHTPAREN) {
				params = append(params, p.pattern())
				for p.match(token.COMMA) {
					p.advance()
					if p.match(token.RIGHTPAREN) {
						break
					}
					params = append(params, p.pattern())
				}
			}
			p.consume(token.RIGHTPAREN)
			pattern = &ast.Call{Func: &ast.This{Token: tok}, Args: params}
		} else {
			// otherwise, it is a single pattern as a parameter.
			tok := p.peek()
			pattern = &ast.Call{Func: &ast.This{Token: tok}, Args: []ast.Node{p.pattern()}}
		}

		p.consume(token.ARROW)
		return pattern
	}, func() ast.Node {
		// if the parsing is failed, insert `#() ->` as pattern and go back to the original position.
		return &ast.Call{Func: &ast.This{Token: p.peek()}, Args: []ast.Node{}}
	})

	exprs := []ast.Node{p.expr()}
	for p.match(token.SEMICOLON) {
		p.advance()
		if p.match(token.RIGHTBRACE) {
			break
		}
		exprs = append(exprs, p.expr())
	}

	if p.err != nil {
		// if the parsing is failed, add patternErr to the error.
		p.recover(patternErr)
	}

	return &ast.CodataClause{Pattern: pattern, Expr: &ast.Seq{Exprs: exprs}}
}

// pattern = methodPat ;
func (p *Parser) pattern() ast.Node {
	if p.IsAtEnd() {
		p.recover(unexpectedToken(p.peek(), "pattern"))

		return nil
	}

	return p.methodPat()
}

// methodPat = atomPat (accessPatTail | callPatTail)* ;
func (p *Parser) methodPat() ast.Node {
	pat := p.atomPat()
	for {
		switch {
		case p.match(token.DOT):
			pat = p.accessPatTail(pat)
		case p.match(token.LEFTPAREN):
			pat = p.callPatTail(pat)
		default:
			return pat
		}
	}
}

// accessPatTail = "." IDENT callPatTail? ;
func (p *Parser) accessPatTail(receiver ast.Node) ast.Node {
	p.consume(token.DOT)
	name := p.consume(token.IDENT)
	pat := &ast.Access{Receiver: receiver, Name: name}

	if p.match(token.LEFTPAREN) {
		return p.callPatTail(pat)
	}

	return pat
}

// callPatTail = "(" ")" | "(" pattern ("," pattern)* ","? ")" ;
func (p *Parser) callPatTail(fun ast.Node) ast.Node {
	p.consume(token.LEFTPAREN)
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
	p.consume(token.RIGHTPAREN)

	return &ast.Call{Func: fun, Args: args}
}

// atomPat = IDENT | INTEGER | STRING | "(" pattern ")" ;
func (p *Parser) atomPat() ast.Node {
	//exhaustive:ignore
	switch tok := p.advance(); tok.Kind {
	case token.SHARP:
		return &ast.This{Token: tok}
	case token.IDENT:
		return &ast.Var{Name: tok}
	case token.INTEGER, token.STRING:
		return &ast.Literal{Token: tok}
	case token.LEFTPAREN:
		pat := p.pattern()
		p.consume(token.RIGHTPAREN)

		return &ast.Paren{Expr: pat}
	default:
		p.recover(unexpectedToken(tok, "identifier", "integer", "string", "`(`"))

		return nil
	}
}

// type = binopType ;
func (p *Parser) typ() ast.Node {
	if p.IsAtEnd() {
		p.recover(unexpectedToken(p.peek(), "type"))

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

// callType = (PRIM "(" IDENT ("," type)* ","? ")" | atomType) ("(" ")" | "(" type ("," type)* ","? ")")* ;
func (p *Parser) callType() ast.Node {
	var typ ast.Node
	if p.match(token.PRIM) {
		p.advance()
		p.consume(token.LEFTPAREN)
		name := p.consume(token.IDENT)
		args := []ast.Node{}
		if !p.match(token.RIGHTPAREN) {
			for p.match(token.COMMA) {
				p.advance()
				if p.match(token.RIGHTPAREN) {
					break
				}
				args = append(args, p.typ())
			}
		}
		p.consume(token.RIGHTPAREN)
		typ = &ast.Prim{Name: name, Args: args}
	} else {
		typ = p.atomType()
	}

	for p.match(token.LEFTPAREN) {
		typ = p.callTypeTail(typ)
	}

	return typ
}

func (p *Parser) callTypeTail(fun ast.Node) *ast.Call {
	p.consume(token.LEFTPAREN)
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
	p.consume(token.RIGHTPAREN)

	return &ast.Call{Func: fun, Args: args}
}

// atomType = IDENT | "{" fieldType ("," fieldType)* ","? "}" | "(" type ("," type)* ","? ")" ;
func (p *Parser) atomType() ast.Node {
	//exhaustive:ignore
	switch tok := p.advance(); tok.Kind {
	case token.IDENT:
		return &ast.Var{Name: tok}
	case token.LEFTBRACE:
		fields := []*ast.Field{p.fieldType()}
		for p.match(token.COMMA) {
			p.advance()
			if p.match(token.RIGHTBRACE) {
				break
			}
			fields = append(fields, p.fieldType())
		}
		p.consume(token.RIGHTBRACE)

		return &ast.Object{Fields: fields}
	case token.LEFTPAREN:
		typ := p.typ()
		p.consume(token.RIGHTPAREN)

		return &ast.Paren{Expr: typ}
	default:
		p.recover(unexpectedToken(tok, "identifier", "`{`", "`(`"))

		return nil
	}
}

// fieldType = IDENT ":" type ;
func (p *Parser) fieldType() *ast.Field {
	name := p.consume(token.IDENT)
	p.consume(token.COLON)
	typ := p.typ()

	return &ast.Field{Name: name.Lexeme, Expr: typ}
}

func (p *Parser) recover(err error) {
	p.err = errors.Join(err, p.err)
}

func (p Parser) peek() token.Token {
	return p.tokens[p.current]
}

func (p Parser) peekNth(n int) token.Token {
	return p.tokens[p.current+n]
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

func (p Parser) matchNth(n int, kind token.Kind) bool {
	if p.current+n >= len(p.tokens) {
		return false
	}
	if p.tokens[p.current+n].Kind == token.EOF {
		return false
	}

	return p.peekNth(n).Kind == kind
}

func (p *Parser) consume(kind token.Kind) token.Token {
	if p.match(kind) {
		return p.advance()
	}

	p.recover(unexpectedToken(p.peek(), kind.String()))

	return p.peek()
}

type UnexpectedTokenError struct {
	Expected []string
}

func (e UnexpectedTokenError) Error() string {
	var msg string
	if len(e.Expected) >= 1 {
		msg = e.Expected[0]
	}

	for _, ex := range e.Expected[1:] {
		msg = msg + ", " + ex
	}

	return "unexpected token: expected " + msg
}

func unexpectedToken(t token.Token, expected ...string) error {
	return utils.PosError{Where: t, Err: UnexpectedTokenError{Expected: expected}}
}

func try[T any](p *Parser, action func() T, recover func() T) (T, error) {
	savedErr := p.err
	savedCurrent := p.current

	node := action()
	if p.err != nil {
		raisedErr := p.err
		p.err = savedErr
		p.current = savedCurrent

		return recover(), raisedErr
	}

	return node, nil
}
