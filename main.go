package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/peterh/liner"
)

func main() {
	const (
		inputUsage = "input file path"
	)
	var inputPath string
	flag.StringVar(&inputPath, "input", "", inputUsage)
	flag.StringVar(&inputPath, "i", "", inputUsage+" (shorthand)")

	flag.Parse()

	if inputPath == "" {
		err := RunPrompt()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		if err := RunFile(inputPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

var history = filepath.Join(xdg.DataHome, "anma", ".anma_history")

func RunPrompt() error {
	line := liner.NewLiner()
	defer func() {
		if err := os.MkdirAll(filepath.Dir(history), os.ModePerm); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if f, err := os.Create(history); err == nil {
			defer f.Close()
			if _, err := line.WriteHistory(f); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
		line.Close()
	}()

	if f, err := os.Open(history); err == nil {
		defer f.Close()
		if _, err := line.ReadHistory(f); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	r := NewRunner()
	for {
		input, err := line.Prompt("> ")
		if err != nil {
			return err
		}
		println(input)
		line.AppendHistory(input)
		err = r.Run(input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func RunFile(path string) error {
	r := NewRunner()
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return r.Run(string(bytes))
}

type Runner struct {
	nodes []Node
	infix *InfixResolver
}

func NewRunner() *Runner {
	return &Runner{nodes: []Node{}, infix: NewInfixResolver()}
}

func (r *Runner) Run(source string) error {
	tokens, err := Lex(source)
	if err != nil {
		return err
	}

	var newNodes []Node
	if decls, err := NewParser(tokens).ParseDecl(); err == nil {
		newNodes = decls
	} else if expr, err := NewParser(tokens).ParseExpr(); err == nil {
		newNodes = []Node{expr}
	} else {
		return err
	}

	for _, node := range newNodes {
		fmt.Println(node)
	}

	fmt.Println("flat:")

	for i, node := range newNodes {
		newNodes[i] = Flat(node)
		fmt.Println(newNodes[i])
	}

	fmt.Println("resolve:")

	for _, node := range newNodes {
		r.infix.Load(node)
	}

	for i, node := range newNodes {
		newNodes[i] = r.infix.Resolve(node)
		fmt.Println(newNodes[i])
	}

	r.nodes = append(r.nodes, newNodes...)

	return nil
}
