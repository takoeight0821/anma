package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// Examples:
// $ ./a.out -comment -in ../ast.go -out ../docs/ast.ebnf
func main() {
	var (
		comment = flag.Bool("comment", false, "comment")
		in      = flag.String("in", "", "input file")
		out     = flag.String("out", "", "output file")
	)
	flag.Parse()

	if *in == "" || *out == "" {
		flag.Usage()
		return
	}

	if *comment {
		commentFile(*in, *out)
		return
	} else {
		flag.Usage()
	}
}

// Get all comments from the file and print them.
func commentFile(in, out string) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, in, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	comments := make(map[*ast.TypeSpec]*ast.CommentGroup)
	commentsList := make([]*ast.TypeSpec, 0)
	ast.Inspect(node, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.GenDecl:
			if n.Tok == token.TYPE {
				for _, spec := range n.Specs {
					spec := spec.(*ast.TypeSpec)
					comments[spec] = n.Doc
					commentsList = append(commentsList, spec)
				}
			}
		}
		return true
	})

	output, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer output.Close()

	fmt.Fprintf(output, "(* Code generated by go generate; DO NOT EDIT. *)\n\n")

	for _, n := range commentsList {
		c := comments[n]
		if c == nil {
			continue
		}
		str := strings.TrimSuffix(c.Text(), "\n")
		if strings.HasPrefix("tool:ignore", str) {
			continue
		}
		fmt.Fprintf(output, "%s (* type %v *)\n\n", strings.TrimSuffix(c.Text(), "\n"), n.Name)
	}
}
