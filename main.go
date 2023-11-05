package main

import (
	"errors"
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
	"github.com/takoeight0821/anma/rename"
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
	r.AddPass(rename.NewRenamer())
	for {
		input, err := line.Prompt("> ")
		if err != nil {
			return err
		}
		line.AppendHistory(input)
		nodes, err := r.RunSource(input)
		if err != nil {
			var wrappedErr interface{ Unwrap() []error }
			if errors.As(err, &wrappedErr) {
				for _, err := range wrappedErr.Unwrap() {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}

		ev := eval.NewEvaluator()
		for _, node := range nodes {
			value, err := ev.Eval(node)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			fmt.Println(value)
		}
	}
}

func RunFile(path string) error {
	r := driver.NewPassRunner()
	r.AddPass(codata.Flat{})
	r.AddPass(infix.NewInfixResolver())
	r.AddPass(rename.NewRenamer())
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	_, err = r.RunSource(string(bytes))
	return err
}
