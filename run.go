package main

type Pass interface {
	Init([]Node) error
	Run([]Node) ([]Node, error)
}

type PassRunner struct {
	passes []Pass
}

func NewPassRunner() *PassRunner {
	return &PassRunner{}
}

// AddPass adds a pass to the end of the pass list.
func (r *PassRunner) AddPass(pass Pass) {
	r.passes = append(r.passes, pass)
}

// Run executes passes in order.
// If an error occurs, it stops the execution and returns the current program.
func (r *PassRunner) Run(program []Node) ([]Node, error) {
	for _, pass := range r.passes {
		err := pass.Init(program)
		if err != nil {
			return program, err
		}
		program, err = pass.Run(program)
		if err != nil {
			return program, err
		}
	}

	return program, nil
}

// RunSource parses the source code and executes passes in order.
func (r *PassRunner) RunSource(source string) ([]Node, error) {
	tokens, err := Lex(source)
	if err != nil {
		return nil, err
	}

	var program []Node
	if decls, err := NewParser(tokens).ParseDecl(); err == nil {
		program = decls
	} else if expr, err := NewParser(tokens).ParseExpr(); err == nil {
		program = []Node{expr}
	} else {
		return nil, err
	}

	return r.Run(program)
}

func (r *PassRunner) Predefined() {
	r.AddPass(Flat{})
	r.AddPass(NewInfixResolver())
	r.AddPass(NewRenamer())
}

/*

// Runner manages informations of the current running program and executes it.
// Runner doen't hold the program itself. Each sub-module holds the program and specific informations.
type Runner struct {
	infix  *InfixResolver
	rename *Renamer
}

func NewRunner() *Runner {
	return &Runner{infix: NewInfixResolver(), rename: NewRenamer()}
}

// Load parses the source code and adds it to the program.
func (r *Runner) Load(source string) error {
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
		fmt.Println(program[i])
	}

	for _, node := range program {
		r.infix.Load(node)
	}

	for i, node := range program {
		program[i] = r.rename.Solve(r.infix.Resolve(node))
	}

	if err = r.rename.PopError(); err != nil {
		return err
	}

	return nil
}

func (r *Runner) Run(source string) error {
	if err := r.Load(source); err != nil {
		return err
	}

	return nil
}

*/
