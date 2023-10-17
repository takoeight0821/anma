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
		dump    = flag.Bool("dump", false, "dump")
		in      = flag.String("in", "", "input file")
		out     = flag.String("out", "", "output file")
	)
	flag.Parse()

	if *comment {
		commentFile(*in, *out)
		return
	} else if *dump {
		dumpFile(*in)
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

	comments := make(map[ast.Node]*ast.CommentGroup)
	commentsList := make([]ast.Node, 0)
	ast.Inspect(node, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.GenDecl:
			for _, spec := range n.Specs {
				comments[spec] = n.Doc
				commentsList = append(commentsList, spec)
			}
		case *ast.FuncDecl:
			comments[n] = n.Doc
			commentsList = append(commentsList, n)
		}
		return true
	})

	output, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer output.Close()

	fmt.Fprintf(output, "(* Code generated by go generate; DO NOT EDIT. *)\n\n")

comments:
	for _, n := range commentsList {
		c := comments[n]
		if c == nil {
			continue
		}
		for _, c := range c.List {
			if strings.Contains(c.Text, "tool:ignore") {
				continue comments
			}
		}
		str := strings.TrimSuffix(c.Text(), "\n")
		switch n := n.(type) {
		case *ast.TypeSpec:
			fmt.Fprintf(output, "%s (* type %v *)\n\n", str, n.Name)
		case *ast.ValueSpec:
			fmt.Fprintf(output, "%s (* var", str)
			for _, name := range n.Names {
				fmt.Fprintf(output, " %v", name)
			}
			fmt.Fprintf(output, " *)\n\n")
		case *ast.FuncDecl:
			fmt.Fprintf(output, "%s (* func %v *)\n\n", str, n.Name)
		}
	}
}

func dumpFile(in string) {
	fset := new(token.FileSet)
	f, _ := parser.ParseFile(fset, in, nil, parser.ParseComments)
	ast.Print(fset, f)
}
