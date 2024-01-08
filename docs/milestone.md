- Integer, Arithmetic, Bit Manipulation
- String
- Function
- Object
- Rune, Stream
- Variant, Pattern Matching
- Copattern
- Type

```
#(n).tail.head -> #(n), #.tail, #.head
#.print(x) -> #.print , #(x)
```

Cutting copatterns
```
{ #.print(x) -> ... }
=>
{ #.print -> { #(x) -> ... } }
```