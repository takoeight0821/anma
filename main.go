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

	r := driver.NewPassRunner()
	r.AddPass(codata.Flat{})
	r.AddPass(infix.NewInfixResolver())
	r.AddPass(nameresolve.NewResolver())

	ev := eval.NewEvaluator()
	ev.SetErrorHandler(func(evErr error) {
		fmt.Fprintf(os.Stderr, "Error: %v\n", evErr)
	})

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
		for _, node := range nodes {
			value := ev.Eval(node)
			fmt.Println(value)
			ev.ResetError()
		}
	}
}

func RunFile(path string) error {
	r := driver.NewPassRunner()
	r.AddPass(codata.Flat{})
	r.AddPass(infix.NewInfixResolver())
	r.AddPass(nameresolve.NewResolver())
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	nodes, err := r.RunSource(string(bytes))
	if err != nil {
		return fmt.Errorf("run file: %w", err)
	}

	ev := eval.NewEvaluator()
	for _, node := range nodes {
		ev.Eval(node)
	}

	main, ok := ev.SearchMain()
	if !ok {
		return noMainError{}
	}
	top := token.Token{Kind: token.IDENT, Lexeme: "toplevel", Line: 0, Literal: -1}
	main.SetErrorHandler(func(evErr error) {
		err = evErr
	})
	main.Apply(top)

	return fmt.Errorf("run file: %w", err)
}

type noMainError struct{}

func (noMainError) Error() string {
	return "no main function"
}
