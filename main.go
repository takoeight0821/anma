package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/peterh/liner"
	"github.com/takoeight0821/anma/flat"
	"github.com/takoeight0821/anma/parser"
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
		RunPrompt()
	} else {
		if err := RunFile(inputPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

var history = filepath.Join(xdg.DataHome, "anma", ".anma_history")

func RunPrompt() {
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

	for {
		if input, err := line.Prompt("> "); err == nil {
			line.AppendHistory(input)
			err := Run(input)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		} else {
			break
		}
	}
}

func RunFile(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return Run(string(bytes))
}

func Run(source string) error {
	tokens, err := parser.Lex(source)
	if err != nil {
		return err
	}

	p := parser.NewParser(tokens)
	expr, err := p.Parse()
	if err != nil {
		return err
	}

	fmt.Println(expr)
	fmt.Printf("flat:\n%v\n", flat.Flat(expr))
	fmt.Printf("original:\n%v\n", expr)

	return nil
}
