package main

import (
	"errors"
	"fmt"
	"strings"
)

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

// expr = let
func (p *Parser) expr() Node {
	return p.let()
}

// let = "let" pattern "=" assert | "fn" pattern "{" expr (";" expr)* ";"? "}" | assert
func (p *Parser) let() Node {
	if p.match(LET) {
		p.advance()
		pattern := p.pattern()
		p.consume(EQUAL, "expected `=`")
		expr := p.assert()
		return Let{pattern, expr}
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
		return Lambda{pattern, exprs}
	}
	return p.assert()
}

// assert = binop (":" type)*
func (p *Parser) assert() Node {
	expr := p.binop()
	for p.match(COLON) {
		p.advance()
		typ := p.typ()
		expr = Assert{expr, typ}
	}
	return expr
}

// binop = access (operator access)*
func (p *Parser) binop() Node {
	expr := p.access()
	for p.match(OPERATOR) {
		op := p.advance()
		right := p.access()
		expr = Binary{expr, op, right}
	}
	return expr
}

// access = call ("." IDENTIFIER)*
func (p *Parser) access() Node {
	expr := p.call()
	for p.match(DOT) {
		p.advance()
		name := p.consume(IDENT, "expected identifier")
		expr = Access{expr, name}
	}
	return expr
}

// call = atom finishCall*
func (p *Parser) call() Node {
	expr := p.atom()
	for p.match(LEFT_PAREN) {
		expr = p.finishCall(expr)
	}
	return expr
}

// finishCall = "(" ")" | "(" expr ("," expr)* ","? ")"
func (p *Parser) finishCall(fun Node) Node {
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
	return Call{fun, args}
}

// atom = IDENT | INTEGER | STRING | codata | "(" expr ("," expr)* ","? ")" | "(" ")"
func (p *Parser) atom() Node {
	switch t := p.advance(); t.Kind {
	case IDENT:
		return Var{t}
	case INTEGER, STRING:
		return Literal{t}
	case LEFT_BRACE:
		return p.codata()
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
		return Paren{elems}
	default:
		p.recover(parseError(t, "expected variable, literal, or parenthesized expression"))
		return nil
	}
}

// codata = "{" clause ("," clause)* ","? "}"
func (p *Parser) codata() Node {
	clauses := []Clause{p.clause()}
	for p.match(COMMA) {
		p.advance()
		if p.match(RIGHT_BRACE) {
			break
		}
		clauses = append(clauses, p.clause())
	}
	p.consume(RIGHT_BRACE, "expected `}`")
	return Codata{clauses}
}

// clause = pattern "->" expr (";" expr)* ";"?
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
	return Clause{pattern, exprs}
}

// pattern = accessPat
func (p *Parser) pattern() Node {
	return p.accessPat()
}

// accessPat = callPat ("." IDENTIFIER)*
func (p *Parser) accessPat() Node {
	pat := p.callPat()
	for p.match(DOT) {
		p.advance()
		name := p.consume(IDENT, "expected identifier")
		pat = Access{pat, name}
	}
	return pat
}

// callPat = atomPat finishCallPat*
func (p *Parser) callPat() Node {
	pat := p.atomPat()
	for p.match(LEFT_PAREN) {
		pat = p.finishCallPat(pat)
	}
	return pat
}

// finishCallPat = "(" ")" | "(" pattern ("," pattern)* ","? ")"
func (p *Parser) finishCallPat(fun Node) Node {
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
	return Call{fun, args}
}

// atomPat = IDENT | INTEGER | STRING | "(" pattern ")"
func (p *Parser) atomPat() Node {
	switch t := p.advance(); t.Kind {
	case SHARP:
		return This{t}
	case IDENT:
		return Var{t}
	case INTEGER, STRING:
		return Literal{t}
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
		return Paren{patterns}
	default:
		p.recover(parseError(t, "expected variable, literal, or parenthesized pattern"))
		return nil
	}
}

// type = binopType
func (p *Parser) typ() Node {
	return p.binopType()
}

// binopType = callType (operator callType)*
func (p *Parser) binopType() Node {
	typ := p.callType()
	for p.match(OPERATOR) {
		op := p.advance()
		right := p.callType()
		typ = Binary{typ, op, right}
	}
	return typ
}

// callType = atomType finishCallType*
func (p *Parser) callType() Node {
	typ := p.atomType()
	for p.match(LEFT_PAREN) {
		typ = p.finishCallType(typ)
	}
	return typ
}

// finishCallType = "(" ")" | "(" type ("," type)* ","? ")"
func (p *Parser) finishCallType(fun Node) Node {
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
	return Call{fun, args}
}

// atomType = IDENT | "(" type ("," type)* ","? ")"
func (p *Parser) atomType() Node {
	switch t := p.advance(); t.Kind {
	case IDENT:
		return Var{t}
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
		return Paren{types}
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

func parseError(token Token, message string) error {
	if token.Kind == EOF {
		return fmt.Errorf("at end: %s", message)
	}
	return fmt.Errorf("at %d: `%s`, %s", token.Line, token.Lexeme, message)
}

// AST

type Node interface {
	fmt.Stringer
	Base() Token
}

// var := IDENTIFIER
type Var struct {
	Name Token
}

func (v Var) String() string {
	return parenthesize("var", v.Name)
}

func (v Var) Base() Token {
	return v.Name
}

var _ Node = Var{}

// literal := INTEGER | FLOAT | RUNE | STRING
type Literal struct {
	Token
}

func (l Literal) String() string {
	return parenthesize("literal", l.Token)
}

func (l Literal) Base() Token {
	return l.Token
}

var _ Node = Literal{}

// paren := "(" expr ("," expr)* ","? ")" | "(" ")"
// If len(Exprs) == 0, it is an empty tuple.
// If len(Exprs) == 1, it is a parenthesized expression.
// Otherwise, it is a tuple.
type Paren struct {
	Elems []Node
}

func (p Paren) String() string {
	ss := make([]fmt.Stringer, len(p.Elems))
	for i, elem := range p.Elems {
		ss[i] = elem
	}
	return parenthesize("paren", ss...)
}

func (p Paren) Base() Token {
	if len(p.Elems) == 0 {
		return Token{}
	}
	return p.Elems[0].Base()
}

var _ Node = Paren{}

// access := expr "." IDENTIFIER
type Access struct {
	Receiver Node
	Name     Token
}

func (a Access) String() string {
	return parenthesize("access", a.Receiver, a.Name)
}

func (a Access) Base() Token {
	return a.Name
}

var _ Node = Access{}

// call := expr "(" ")" | expr "(" expr ("," expr)* ","? ")"
type Call struct {
	Func Node
	Args []Node
}

func (c Call) String() string {
	return parenthesize("call", prepend(c.Func, squash(c.Args))...)
}

func (c Call) Base() Token {
	return c.Func.Base()
}

var _ Node = Call{}

// binary := expr operator expr
type Binary struct {
	Left  Node
	Op    Token
	Right Node
}

func (b Binary) String() string {
	return parenthesize("binary", b.Left, b.Op, b.Right)
}

func (b Binary) Base() Token {
	return b.Op
}

var _ Node = Binary{}

// assert := expr ":" type
type Assert struct {
	Expr Node
	Type Node
}

func (a Assert) String() string {
	return parenthesize("assert", a.Expr, a.Type)
}

func (a Assert) Base() Token {
	return a.Expr.Base()
}

var _ Node = Assert{}

// let := "let" pattern "=" expr
type Let struct {
	Bind Node
	Body Node
}

func (l Let) String() string {
	return parenthesize("let", l.Bind, l.Body)
}

func (l Let) Base() Token {
	return l.Bind.Base()
}

var _ Node = Let{}

// codata := "{" clause ("," clause)* ","? "}"
type Codata struct {
	Clauses []Clause // len(Clauses) > 0
}

func (c Codata) String() string {
	return parenthesize("codata", squash(c.Clauses)...)
}

func (c Codata) Base() Token {
	if len(c.Clauses) == 0 {
		return Token{}
	}
	return c.Clauses[0].Base()
}

var _ Node = Codata{}

// clause := pattern "->" expr (";" expr)* ";"?
type Clause struct {
	Pattern Node
	Exprs   []Node // len(Exprs) > 0
}

func (c Clause) String() string {
	return parenthesize("clause", prepend(c.Pattern, squash(c.Exprs))...)
}

func (c Clause) Base() Token {
	if c.Pattern == nil {
		return Token{}
	}
	return c.Pattern.Base()
}

var _ Node = Clause{}

// lambda := "fn" pattern "{" expr (";" expr)* ";"? "}"
type Lambda struct {
	Pattern Node
	Exprs   []Node // len(Exprs) > 0
}

func (l Lambda) String() string {
	return parenthesize("lambda", prepend(l.Pattern, squash(l.Exprs))...)
}

func (l Lambda) Base() Token {
	return l.Pattern.Base()
}

var _ Node = Lambda{}

// case := "case" expr "{" clause ("," clause)* ","? "}"
type Case struct {
	Scrutinee Node
	Clauses   []Clause // len(Clauses) > 0
}

func (c Case) String() string {
	return parenthesize("case", prepend(c.Scrutinee, squash(c.Clauses))...)
}

func (c Case) Base() Token {
	return c.Scrutinee.Base()
}

var _ Node = Case{}

// object := "{" field ("," field)* ","? "}"
type Object struct {
	Fields []Field // len(Fields) > 0
}

func (o Object) String() string {
	return parenthesize("object", squash(o.Fields)...)
}

func (o Object) Base() Token {
	return o.Fields[0].Base()
}

var _ Node = Object{}

// field := IDENTIFIER ":" expr
type Field struct {
	Name  string
	Exprs []Node
}

func (f Field) String() string {
	return parenthesize("field "+f.Name, squash(f.Exprs)...)
}

func (f Field) Base() Token {
	return f.Exprs[0].Base()
}

var _ Node = Field{}

// typeDecl := "type" IDENTIFIER "=" type
type TypeDecl struct {
	Name Token
	Type Node
}

func (t TypeDecl) String() string {
	return parenthesize("type", t.Name, t.Type)
}

func (t TypeDecl) Base() Token {
	return t.Name
}

var _ Node = TypeDecl{}

// varDecl := "def" identifier "=" expr | "def" identifier ":" type | "def" identifier ":" type "=" expr
type VarDecl struct {
	Name Token
	Type Node
	Expr Node
}

func (v VarDecl) String() string {
	if v.Type == nil {
		return parenthesize("var", v.Name, v.Expr)
	}
	if v.Expr == nil {
		return parenthesize("var", v.Name, v.Type)
	}
	return parenthesize("var", v.Name, v.Type, v.Expr)
}

func (v VarDecl) Base() Token {
	return v.Name
}

var _ Node = VarDecl{}

// infixDecl := ("infix" | "infixl" | "infixr") INTEGER IDENTIFIER
type InfixDecl struct {
	Assoc      Token
	Precedence Token
	Name       Token
}

func (i InfixDecl) String() string {
	return parenthesize("infix", i.Assoc, i.Precedence, i.Name)
}

func (i InfixDecl) Base() Token {
	return i.Assoc
}

var _ Node = InfixDecl{}

type This struct {
	Token
}

func (t This) String() string {
	return parenthesize("this", t.Token)
}

func (t This) Base() Token {
	return t.Token
}

var _ Node = This{}

func parenthesize(head string, nodes ...fmt.Stringer) string {
	var b strings.Builder
	b.WriteString("(")
	b.WriteString(head)
	for _, node := range nodes {
		b.WriteString(" ")
		if node == nil {
			b.WriteString("<nil>")
		} else {
			b.WriteString(node.String())
		}
	}
	b.WriteString(")")
	return b.String()
}

func squash[T fmt.Stringer](elems []T) []fmt.Stringer {
	nodes := make([]fmt.Stringer, len(elems))
	for i, elem := range elems {
		nodes[i] = elem
	}
	return nodes
}

func prepend(elem fmt.Stringer, slice []fmt.Stringer) []fmt.Stringer {
	return append([]fmt.Stringer{elem}, slice...)
}
