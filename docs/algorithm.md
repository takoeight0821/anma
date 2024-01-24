# ネストした余パターンマッチの展開アルゴリズム

## 式の走査

```
Flat(program []Node) -> []Node:
    for i, node in program:
        program[i] = Traverse(node, flatEach) 
    return program

flatEach(node Node) -> Node:
    if node is Codata:
        return flatCodata(node)
    else:
        return node
```

## パターンリストの構築

```
NotChecked = -2
NoArgs = -1
ZeroArgs = 0

flatCodata(codata Codata) -> Node:
    arity = NotChecked
    clauses = [] as []plistClause
    for i, clause in codata.clauses:
        plist = PatternList(
            accessors(clause.pattern),
            guard(clause.pattern))
        clauses[i] = plistClause(plist, clause.body)

        if arity == NotChecked:
            arity = plist.Arity()

        if !checkArity(arity, plist.Arity()):
            error("arity mismatch")

    return build(arity, clauses)

plistClause(plist PatternList, body Node) 
PatternList(fields []string, guards []Node)

accessors(pattern) -> []string:
    if pattern is Access:
        return append(accessors(pattern.receiver), pattern.name)

    return []

guard(pattern) -> []Node:
    if pattern is Access:
        return guard(pattern)
    if pattern is Call:
        if pattern.Func is not This:
            error("expect This") 
        
        return pattern.args
    
    return []
```

## 無名関数式の構築

```
build(arity int, clauses []plistClause) -> Node:
    if arity == NoArgs:
        return buildObject([], clauses)
    else:
        return buildFunction(arity, clauses)

buildObject(scrutinees []string, clauses) -> Node:
    // それぞれの節から，先頭のフィールドを取り出す
    next = groupClausesByHeadField(clauses)
    fields = [] as []Field
    for field, clauses in next:
        body = fieldBody(clauses)
        fields = append(fields, Field(field, newCase(scrutinees, body)))
    
    return Object(fields)

groupClausesByHeadField: TODO
newCase: TODO

buildLambda(arity, clauses):
    scrutinees = [] as []string
    for arity times:
        scrutinees = append(scrutinees, internVar())
    
    // いずれかのclauseにフィールドが残っているなら，bodyはObject
    for clause in clauses:
        if hasAccess(clause):
            obj = buildObject(scrutinees, clauses)
            
            return newLambda(scrutinees, obj)
    
    // そうでなければ，bodyはCase
    caseClauses = []CaseClause
    for clause in clauses:
        caseClauses = append(caseClauses, toCaseClause(clause))
    
    return newLambda(scrutinees, newCase(scrutinees, caseClauses))

internVar: TODO
hasAccess: TODO

newLambda: TODO
toCaseClause: TODO
```


## ASTの定義と補助関数

```
Node:
    Var(name string)
    Const(value string)
    Tuple(elems []Node)
    Access(receiver Node, name string)
    Call(func Node, args []Node)
    Let(name string, value Node)
    Seq(nodes []Node)
    Codata(clauses []CodataClause)
    CodataClause(pattern Pattern, body Node)
    Object(fields []Field)
    Field(name string, value Node)
    Function(params []string, body Node)
    Case(clauses []CaseClause)
    CaseClause(patterns []Pattern, body Node)

Pattern:
    This
    Var(name string)
    Const(value string)
    Tuple(elems []Pattern)
    Access(receiver Pattern, name string)
    Call(func Pattern, args []Pattern)

Plate(Var(name), f):
    return Var(name)
Plate(Const(value), f):
    return Const(value)
Plate(Access(receiver, name), f):
    return Access(f(receiver), name)
Plate(Call(func, args), f):
    return Call(f(func), map(f, args))
Plate(other): ...

Traverse(node, f):
    return f(Plate(node, fn(child): Traverse(child, f)))
```
