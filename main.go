package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/peterh/liner"
	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/eval"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/nameresolve"
	"github.com/takoeight0821/anma/token"
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
		// If no input file is specified, run the REPL.
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

var HISTORY = filepath.Join(xdg.DataHome, "anma", ".anma_history")

// writeHistory writes the history of the REPL to a file.
func writeHistory(line *liner.State) {
	// Create the directory for the history file if it does not exist.
	if err := os.MkdirAll(filepath.Dir(HISTORY), os.ModePerm); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	// Write the history file.
	// If the file does not exist, it will be created automatically.
	// If the file exists, it will be overwritten.
	if f, err := os.Create(HISTORY); err == nil {
		defer f.Close()
		if _, err := line.WriteHistory(f); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	line.Close()
}

// readHistory reads the history of the REPL from a file.
func readHistory(line *liner.State) {
	if f, err := os.Open(HISTORY); err == nil {
		defer f.Close()
		if _, err := line.ReadHistory(f); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

// RunPrompt runs the REPL.
func RunPrompt() error {
	line := liner.NewLiner()
	defer writeHistory(line)
	readHistory(line)

	r := driver.NewPassRunner()
	r.AddPass(codata.Flat{})
	r.AddPass(infix.NewInfixResolver())
	r.AddPass(nameresolve.NewResolver())

	ev := eval.NewEvaluator()

	for {
		input, err := line.Prompt("> ")
		if err != nil {
			return fmt.Errorf("prompt: %w", err)
		}
		line.AppendHistory(input)

		nodes, err := r.RunSource(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		// Evaluate all nodes.
		for _, node := range nodes {
			value, err := ev.Eval(node)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			fmt.Println(value)
		}
	}
}

// RunFile runs the specified file.
func RunFile(path string) error {
	r := driver.NewPassRunner()
	r.AddPass(codata.Flat{})
	r.AddPass(infix.NewInfixResolver())
	r.AddPass(nameresolve.NewResolver())

	// Read the source code from the file.
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	nodes, err := r.RunSource(string(bytes))
	if err != nil {
		return fmt.Errorf("run file: %w", err)
	}

	ev := eval.NewEvaluator()
	// Evaluate all nodes for loading definitions.
	for _, node := range nodes {
		_, err := ev.Eval(node)
		if err != nil {
			return fmt.Errorf("run file: %w", err)
		}
	}

	main, ok := ev.SearchMain()
	if !ok {
		return noMainError{}
	}
	// top is a dummy token.
	top := token.Token{Kind: token.IDENT, Lexeme: "toplevel", Line: 0, Literal: -1}
	_, err = main.Apply(top)
	if err != nil {
		return fmt.Errorf("run file: %w", err)
	}

	return nil
}

type noMainError struct{}

func (noMainError) Error() string {
	return "no main function"
}
