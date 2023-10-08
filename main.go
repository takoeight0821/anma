package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
		RunPrompt()
	} else {
		if err := RunFile(inputPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

var history = filepath.Join(os.TempDir(), ".tenchi_history")

func RunPrompt() {
	line := liner.NewLiner()
	defer line.Close()

	if f, err := os.Open(history); err == nil {
		defer f.Close()
		line.ReadHistory(f)
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
	tokens, err := Lex(source)
	if err != nil {
		return err
	}

	for _, token := range tokens {
		fmt.Println(token)
	}

	p := NewParser(tokens)
	expr, err := p.Parse()
	if err != nil {
		return err
	}

	fmt.Println(expr)

	return nil
}
