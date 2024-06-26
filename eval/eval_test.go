package eval_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/desugarwith"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/eval"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/nameresolve"
	"github.com/takoeight0821/anma/token"
	"github.com/takoeight0821/anma/utils"
)

func TestGolden(t *testing.T) {
	t.Parallel()

	testfiles, err := utils.FindSourceFiles("../testdata")
	if err != nil {
		t.Errorf("failed to find test files: %v", err)

		return
	}

	for _, testfile := range testfiles {
		source, err := os.ReadFile(testfile)
		if err != nil {
			t.Errorf("failed to read %s: %v", testfile, err)

			return
		}

		runner := driver.NewPassRunner()
		runner.AddPass(&desugarwith.DesugarWith{})
		runner.AddPass(&codata.Flat{})
		runner.AddPass(infix.NewInfixResolver())
		runner.AddPass(nameresolve.NewResolver())

		nodes, err := runner.RunSource(testfile, string(source))
		if err != nil {
			t.Errorf("%s returned error: %v", testfile, err)

			return
		}

		evaluator := eval.NewEvaluator()
		var builder strings.Builder
		evaluator.Stdout = &builder
		evaluator.Stdin = strings.NewReader("test input\n")
		values := make([]eval.Value, len(nodes))

		for i, node := range nodes {
			values[i], err = evaluator.Eval(node)
			if err != nil {
				t.Errorf("%s returned error: %v", testfile, err)

				return
			}
		}

		if main, ok := evaluator.SearchMain(); ok {
			top := token.Token{Kind: token.IDENT, Lexeme: "toplevel", Location: token.Location{}, Literal: -1}
			ret, err := main.Apply(top)
			var exitErr eval.ExitError
			if errors.As(err, &exitErr) {
				fmt.Fprintf(&builder, "exit => %d\n", exitErr.Code)
			} else if err != nil {
				fmt.Fprintf(&builder, "error => %v\n", err)
			}
			if ret != nil {
				fmt.Fprintf(&builder, "result => %s\n", ret.String())
			}

			g := goldie.New(t)
			g.Assert(t, testfile, []byte(builder.String()))
		} else {
			t.Errorf("%s does not have a main function", testfile)
		}
	}
}
