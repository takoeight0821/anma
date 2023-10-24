package main

import "fmt"

// Runner manages informations of the current running program and executes it.
type Runner struct {
	program []Node
	infix   *InfixResolver
	rename  *Renamer
}

func NewRunner() *Runner {
	return &Runner{program: []Node{}, infix: NewInfixResolver(), rename: NewRenamer()}
}

// Load parses the source code and adds it to the program.
func (r *Runner) Load(source string, isFile bool) error {
	tokens, err := Lex(source)
	if err != nil {
		return err
	}

	var program []Node
	if decls, err := NewParser(tokens).ParseDecl(); err == nil {
		program = decls
	} else if expr, err := NewParser(tokens).ParseExpr(); err == nil {
		program = []Node{expr}
	} else {
		return err
	}

	for i, node := range program {
		program[i] = Flat(node)
	}

	for _, node := range program {
		r.infix.Load(node)
	}

	if !isFile {
		// in REPL, toplevel definitions are overridable
		for _, node := range program {
			Transform(node, func(n Node) Node {
				switch n := node.(type) {
				case *TypeDecl:
					r.rename.AddConfig(Override(n.Name))
				case *VarDecl:
					r.rename.AddConfig(Override(n.Name))
				}
				return n
			})
		}
	}

	for i, node := range program {
		program[i] = r.rename.Solve(r.infix.Resolve(node))
	}

	if err = r.rename.PopError(); err != nil {
		return err
	}

	r.program = append(r.program, program...)

	return nil
}

func (r *Runner) Run(source string, isFile bool) error {
	if err := r.Load(source, isFile); err != nil {
		return err
	}

	for _, node := range r.program {
		fmt.Println(node)
	}

	return nil
}
