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

	for {
		input, err := line.Prompt("> ")
		if err != nil {
			return err
		}
		println(input)
		line.AppendHistory(input)
		err = Run(input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
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

	p := NewParser(tokens)

	if decls, err := p.ParseDecl(); err == nil {
		for _, decl := range decls {
			fmt.Println(decl)
		}
		fmt.Println("flat:")
		for _, decl := range decls {
			fmt.Printf("%v\n", Flat(decl))
		}
	} else if expr, err := p.ParseExpr(); err == nil {
		fmt.Println(expr)
		fmt.Printf("flat:\n%v\n", Flat(expr))
	} else {
		return err
	}

	return nil
}
