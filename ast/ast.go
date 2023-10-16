package ast

import (
	"fmt"
	"strings"

	"github.com/takoeight0821/anma/token"
)

// AST

type Node interface {
	fmt.Stringer
	Base() token.Token
}

// var := IDENTIFIER
type Var struct {
	Name token.Token
}

func (v Var) String() string {
	return parenthesize("var", v.Name)
}

func (v Var) Base() token.Token {
	return v.Name
}

var _ Node = Var{}

// literal := INTEGER | FLOAT | RUNE | STRING
type Literal struct {
	token.Token
}

func (l Literal) String() string {
	return parenthesize("literal", l.Token)
}

func (l Literal) Base() token.Token {
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

func (p Paren) Base() token.Token {
	if len(p.Elems) == 0 {
		return token.Token{}
	}
	return p.Elems[0].Base()
}

var _ Node = Paren{}

// access := expr "." IDENTIFIER
type Access struct {
	Receiver Node
	Name     token.Token
}

func (a Access) String() string {
	return parenthesize("access", a.Receiver, a.Name)
}

func (a Access) Base() token.Token {
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

func (c Call) Base() token.Token {
	return c.Func.Base()
}

var _ Node = Call{}

// binary := expr operator expr
type Binary struct {
	Left  Node
	Op    token.Token
	Right Node
}

func (b Binary) String() string {
	return parenthesize("binary", b.Left, b.Op, b.Right)
}

func (b Binary) Base() token.Token {
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

func (a Assert) Base() token.Token {
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

func (l Let) Base() token.Token {
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

func (c Codata) Base() token.Token {
	if len(c.Clauses) == 0 {
		return token.Token{}
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

func (c Clause) Base() token.Token {
	if c.Pattern == nil {
		return token.Token{}
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

func (l Lambda) Base() token.Token {
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

func (c Case) Base() token.Token {
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

func (o Object) Base() token.Token {
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

func (f Field) Base() token.Token {
	return f.Exprs[0].Base()
}

var _ Node = Field{}

// typeDecl := "type" IDENTIFIER "=" type
type TypeDecl struct {
	Name token.Token
	Type Node
}

func (t TypeDecl) String() string {
	return parenthesize("type", t.Name, t.Type)
}

func (t TypeDecl) Base() token.Token {
	return t.Name
}

var _ Node = TypeDecl{}

// varDecl := "def" identifier "=" expr | "def" identifier ":" type | "def" identifier ":" type "=" expr
type VarDecl struct {
	Name token.Token
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

func (v VarDecl) Base() token.Token {
	return v.Name
}

var _ Node = VarDecl{}

// infixDecl := ("infix" | "infixl" | "infixr") INTEGER IDENTIFIER
type InfixDecl struct {
	Assoc      token.Token
	Precedence token.Token
	Name       token.Token
}

func (i InfixDecl) String() string {
	return parenthesize("infix", i.Assoc, i.Precedence, i.Name)
}

func (i InfixDecl) Base() token.Token {
	return i.Assoc
}

var _ Node = InfixDecl{}

type This struct {
	token.Token
}

func (t This) String() string {
	return parenthesize("this", t.Token)
}

func (t This) Base() token.Token {
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

type Kind int

const (
	KExpr Kind = 0b0001
	KPat  Kind = 0b0010
	KType Kind = 0b0100
	KStmt Kind = 0b1000
	Other Kind = 0b0000
	Any   Kind = 0b1111
)

func (k Kind) String() string {
	var b strings.Builder
	if IsExpr(k) {
		b.WriteString("expr")
	}
	if IsPat(k) {
		if b.Len() > 0 {
			b.WriteString("|")
		}
		b.WriteString("pat")
	}
	if IsType(k) {
		if b.Len() > 0 {
			b.WriteString("|")
		}
		b.WriteString("type")
	}
	if IsStmt(k) {
		if b.Len() > 0 {
			b.WriteString("|")
		}
		b.WriteString("stmt")
	}
	if IsOther(k) {
		if b.Len() > 0 {
			b.WriteString("|")
		}
		b.WriteString("outer")
	}
	return b.String()
}

func IsExpr(k Kind) bool {
	return k&KExpr != 0
}

func IsPat(k Kind) bool {
	return k&KPat != 0
}

func IsType(k Kind) bool {
	return k&KType != 0
}

func IsStmt(k Kind) bool {
	return k&KStmt != 0
}

func IsOther(k Kind) bool {
	return k == Other
}

// Traverse the [Node] in depth-first order.
// f is called for each node with the node and its [Kind].
// If n has children, f is called for each child before n and the argument of f for n is allocated newly.
// Otherwise, n is directly applied to f and the result of Traverse(n, f) is the result of f(n).
func Traverse(n Node, f func(Node, Kind) Node, k Kind) Node {
	switch n := n.(type) {
	case Var:
		return f(n, (KExpr|KPat|KType)&k)
	case Literal:
		return f(n, (KExpr|KPat)&k)
	case Paren:
		elems := make([]Node, len(n.Elems))
		for i, elem := range n.Elems {
			elems[i] = Traverse(elem, f, k)
		}
		return f(Paren{Elems: elems}, (KExpr|KPat|KType)&k)
	case Access:
		return f(Access{Receiver: Traverse(n.Receiver, f, k), Name: n.Name}, (KExpr|KPat)&k)
	case Call:
		fun := Traverse(n.Func, f, k)
		args := make([]Node, len(n.Args))
		for i, arg := range n.Args {
			args[i] = Traverse(arg, f, k)
		}
		return f(Call{Func: fun, Args: args}, (KExpr|KPat|KType)&k)
	case Binary:
		return f(Binary{Left: Traverse(n.Left, f, k), Op: n.Op, Right: Traverse(n.Right, f, k)}, (KExpr|KType)&k)
	case Assert:
		return f(Assert{Expr: Traverse(n.Expr, f, KExpr), Type: Traverse(n.Type, f, KType)}, KExpr)
	case Let:
		return f(Let{Bind: Traverse(n.Bind, f, KPat), Body: Traverse(n.Body, f, KExpr)}, KExpr)
	case Codata:
		clauses := make([]Clause, len(n.Clauses))
		for i, clause := range n.Clauses {
			clauses[i] = Traverse(clause, f, Other).(Clause)
		}
		return f(Codata{Clauses: clauses}, KExpr)
	case Clause:
		pat := Traverse(n.Pattern, f, KPat)
		exprs := make([]Node, len(n.Exprs))
		for i, expr := range n.Exprs {
			exprs[i] = Traverse(expr, f, KExpr)
		}
		return f(Clause{Pattern: pat, Exprs: exprs}, Other)
	case Lambda:
		pat := Traverse(n.Pattern, f, KPat)
		exprs := make([]Node, len(n.Exprs))
		for i, expr := range n.Exprs {
			exprs[i] = Traverse(expr, f, KExpr)
		}
		return f(Lambda{Pattern: pat, Exprs: exprs}, KExpr)
	case Case:
		scrutinee := Traverse(n.Scrutinee, f, KExpr)
		clauses := make([]Clause, len(n.Clauses))
		for i, clause := range n.Clauses {
			clauses[i] = Traverse(clause, f, Other).(Clause)
		}
		return f(Case{Scrutinee: scrutinee, Clauses: clauses}, KExpr)
	case Object:
		fields := make([]Field, len(n.Fields))
		for i, field := range n.Fields {
			fields[i] = Traverse(field, f, Other).(Field)
		}
		return f(Object{Fields: fields}, KExpr)
	case Field:
		exprs := make([]Node, len(n.Exprs))
		for i, expr := range n.Exprs {
			exprs[i] = Traverse(expr, f, KExpr)
		}
		return f(Field{Name: n.Name, Exprs: exprs}, Other)
	case TypeDecl:
		return f(TypeDecl{Name: n.Name, Type: Traverse(n.Type, f, KType)}, KStmt)
	case VarDecl:
		return f(VarDecl{Name: n.Name, Type: Traverse(n.Type, f, KType), Expr: Traverse(n.Expr, f, KExpr)}, KStmt)
	case InfixDecl:
		return f(n, KStmt)
	case This:
		return f(n, KPat)
	default:
		return f(n, Other)
	}
}
