# Syntax

## Comments

```haskell
-- This is a comment
{- This is a
   multiline comment -}
```

## Identifiers

Identifiers are used to name functions, variables, and types. They can start with a letter or underscore, followed by any number of letters, underscores, or digits.

```haskell
foo
bar123
_
名前
```

### Reserved words

```haskell
type
def
infix
infixl
infixr
fn
case
```

## Operators

Operators can be made up of unicode symbols or punctuation characters.

```haskell
+
<>
>==
📛
```

### Reserved operators

```haskell
->
:
```

## Literals

### Numeric literals

```haskell
123
0x123
0o123
0b101
123.456
```

### Rune literals

```haskell
'a'
'あ'
'\n'
'\x0a'
'\u{1f4a9}'
```

### String literals

```haskell
"Hello, world!"
`some "random" string`
``raw `string` literal``
```

### BNF

```bnf
program = statement* ;

statement = typeDecl | varDecl | infixDecl ;

typeDecl = "type" identifier "=" type ;

varDecl = "def" identifier "=" expr | "def" identifier ":" type "=" expr | "def" identifier ":" type ;

infixDecl = "infix" operator int | "infixl" operator int | "infixr" operator int ;

type = <subset of expr> ;

expr
    = identifier
    | literal
    | "(" expr ("," expr)* ","? ")"
    | expr "." identifier
    | expr "(" ")"
    | expr "(" expr ("," expr)* ","? ")"
    | expr operator expr
    | expr ":" type
    | "let" pattern "=" expr
    | codata
    | lambda
    | case
    | object ;

codata = "{" clause ("," clause)* ","? "}" ;

lambda = "fn" pattern "{" expr (";" expr)* ";"? "}" ;

case = "case" expr "{" clause ("," clause)* ","? "}" ;

object = "{" field ("," field)* ","? "}" ;

field = identifier ":" expr (";" expr)* ";"? ;

clause = pattern "->" expr (";" expr)* ";"? ;

pattern = <subset of expr> | "#" ;
```