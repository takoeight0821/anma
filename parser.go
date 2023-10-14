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
func (p *Parser) expr() Expr {
	return p.let()
}

// let = "let" pattern "=" assert | "fn" pattern "{" expr (";" expr)* ";"? "}" | assert
func (p *Parser) let() Expr {
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
		exprs := []Expr{p.expr()}
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
func (p *Parser) assert() Expr {
	expr := p.binop()
	for p.match(COLON) {
		p.advance()
		typ := p.typ()
		expr = Assert{expr, typ}
	}
	return expr
}

// pattern = binop
func (p *Parser) pattern() Pattern {
	e := p.binop()
	if e.ValidPattern() {
		return e
	}
	p.recover(parseError(e.Base(), "expected pattern"))
	return nil
}

// type = binop
func (p *Parser) typ() Type {
	e := p.binop()
	if e.ValidType() {
		return e
	}
	p.recover(parseError(e.Base(), "expected type"))
	return nil
}

// binop = access (operator access)*
func (p *Parser) binop() Expr {
	expr := p.access()
	for p.match(OPERATOR) {
		op := p.advance()
		right := p.access()
		expr = Binary{expr, op, right}
	}
	return expr
}

// access = call ("." IDENTIFIER)+
func (p *Parser) access() Expr {
	expr := p.call()
	for p.match(DOT) {
		p.advance()
		name := p.consume(IDENT, "expected identifier")
		expr = Access{expr, name}
	}
	return expr
}

// call = atom finishCall*
func (p *Parser) call() Expr {
	expr := p.atom()
	for p.match(LEFT_PAREN) {
		expr = p.finishCall(expr)
	}
	return expr
}

// finishCall = "(" ")" | "(" expr ("," expr)* ","? ")"
func (p *Parser) finishCall(fun Expr) Expr {
	p.consume(LEFT_PAREN, "expected `(`")
	args := []Expr{}
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

// atom = IDENT | INTEGER | STRING | "(" expr ")"
func (p *Parser) atom() Expr {
	switch t := p.advance(); t.Kind {
	case SHARP:
		return This{t}
	case IDENT:
		return Var{t}
	case INTEGER, STRING:
		return Literal{t}
	case LEFT_BRACE:
		return p.codata()
	case LEFT_PAREN:
		expr := p.expr()
		p.consume(RIGHT_PAREN, "expected `)`")
		return Paren{expr}
	default:
		p.recover(parseError(t, "expected variable, literal, or parenthesized expression"))
		return nil
	}
}

// codata = "{" clause ("," clause)* ","? "}"
func (p *Parser) codata() Expr {
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
	exprs := []Expr{p.expr()}
	for p.match(SEMICOLON) {
		p.advance()
		if p.match(RIGHT_BRACE) {
			break
		}
		exprs = append(exprs, p.expr())
	}
	return Clause{pattern, exprs}
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

type Expr interface {
	Node

	ValidPattern() bool // Check if the node can be used as a pattern.
	ValidType() bool    // Check if the node can be used as a type.
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

func (v Var) ValidType() bool {
	return true
}

func (v Var) ValidPattern() bool {
	return true
}

var _ Expr = Var{}

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

func (l Literal) ValidType() bool {
	return false
}

func (l Literal) ValidPattern() bool {
	return true
}

var _ Expr = Literal{}

// paren := "(" expr ")"
type Paren struct {
	Expr
}

func (p Paren) String() string {
	return parenthesize("paren", p.Expr)
}

var _ Expr = Paren{}

// access := expr "." IDENTIFIER
type Access struct {
	Expr
	Name Token
}

func (a Access) String() string {
	return parenthesize("access", a.Expr, a.Name)
}

func (a Access) Base() Token {
	return a.Name
}

func (a Access) ValidType() bool {
	return false
}

func (a Access) ValidPattern() bool {
	return a.Expr.ValidPattern()
}

var _ Expr = Access{}

// call := expr "(" ")" | expr "(" expr ("," expr)* ","? ")"
type Call struct {
	Func Expr
	Args []Expr
}

func (c Call) String() string {
	return parenthesize("call", prepend(c.Func, squash(c.Args))...)
}

func (c Call) Base() Token {
	return c.Func.Base()
}

func (c Call) ValidType() bool {
	if !c.Func.ValidType() {
		return false
	}

	for _, arg := range c.Args {
		if !arg.ValidType() {
			return false
		}
	}

	return true
}

func (c Call) ValidPattern() bool {
	if !c.Func.ValidPattern() {
		return false
	}

	for _, arg := range c.Args {
		if !arg.ValidPattern() {
			return false
		}
	}

	return true
}

var _ Expr = Call{}

// binary := expr operator expr
type Binary struct {
	Left  Expr
	Op    Token
	Right Expr
}

func (b Binary) String() string {
	return parenthesize("binary", b.Left, b.Op, b.Right)
}

func (b Binary) Base() Token {
	return b.Op
}

func (b Binary) ValidType() bool {
	return b.Left.ValidType() && b.Right.ValidType()
}

func (b Binary) ValidPattern() bool {
	return false
}

var _ Expr = Binary{}

// assert := expr ":" type
type Assert struct {
	Expr
	Type
}

func (a Assert) String() string {
	return parenthesize("assert", a.Expr, a.Type)
}

func (a Assert) Base() Token {
	return a.Expr.Base()
}

func (a Assert) ValidType() bool {
	return false
}

func (a Assert) ValidPattern() bool {
	return false
}

var _ Expr = Assert{}

// let := "let" pattern "=" expr
type Let struct {
	Pattern
	Expr
}

func (l Let) String() string {
	return parenthesize("let", l.Pattern, l.Expr)
}

func (l Let) Base() Token {
	return l.Pattern.Base()
}

func (l Let) ValidType() bool {
	return false
}

func (l Let) ValidPattern() bool {
	return false
}

var _ Expr = Let{}

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

func (c Codata) ValidType() bool {
	return false
}

func (c Codata) ValidPattern() bool {
	return false
}

var _ Expr = Codata{}

// clause := pattern "->" expr (";" expr)* ";"?
type Clause struct {
	Pattern
	Exprs []Expr // len(Exprs) > 0
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
	Pattern
	Exprs []Expr // len(Exprs) > 0
}

func (l Lambda) String() string {
	return parenthesize("lambda", prepend(l.Pattern, squash(l.Exprs))...)
}

func (l Lambda) Base() Token {
	return l.Pattern.Base()
}

func (l Lambda) ValidType() bool {
	return false
}

func (l Lambda) ValidPattern() bool {
	return false
}

var _ Expr = Lambda{}

// case := "case" expr "{" clause ("," clause)* ","? "}"
type Case struct {
	Expr
	Clauses []Clause // len(Clauses) > 0
}

func (c Case) String() string {
	return parenthesize("case", prepend(c.Expr, squash(c.Clauses))...)
}

func (c Case) Base() Token {
	return c.Expr.Base()
}

func (c Case) ValidType() bool {
	return false
}

func (c Case) ValidPattern() bool {
	return false
}

var _ Expr = Case{}

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

func (o Object) ValidType() bool {
	for _, field := range o.Fields {
		if !field.Expr.ValidType() {
			return false
		}
	}
	return true
}

func (o Object) ValidPattern() bool {
	for _, field := range o.Fields {
		if !field.Expr.ValidPattern() {
			return false
		}
	}
	return true
}

var _ Expr = Object{}

// field := IDENTIFIER ":" expr
type Field struct {
	Name Token
	Expr
}

func (f Field) String() string {
	return parenthesize("field", f.Name, f.Expr)
}

func (f Field) Base() Token {
	return f.Name
}

var _ Node = Field{}

type Stmt interface {
	Node
}

// typeDecl := "type" IDENTIFIER "=" type
type TypeDecl struct {
	Name Token
	Type
}

func (t TypeDecl) String() string {
	return parenthesize("type", t.Name, t.Type)
}

func (t TypeDecl) Base() Token {
	return t.Name
}

var _ Stmt = TypeDecl{}

// varDecl := "def" identifier "=" expr | "def" identifier ":" type | "def" identifier ":" type "=" expr
type VarDecl struct {
	Name Token
	Type
	Expr
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

var _ Stmt = VarDecl{}

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

var _ Stmt = InfixDecl{}

type Type interface {
	Expr
}

type Pattern interface {
	Expr
}

type This struct {
	Token
}

func (t This) String() string {
	return parenthesize("this", t.Token)
}

func (t This) Base() Token {
	return t.Token
}

func (t This) ValidType() bool {
	return false
}

func (t This) ValidPattern() bool {
	return true
}

var _ Pattern = This{}

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
