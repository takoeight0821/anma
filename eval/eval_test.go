package eval_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/takoeight0821/anma/codata"
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
		runner.AddPass(&codata.Flat{})
		runner.AddPass(infix.NewInfixResolver())
		runner.AddPass(nameresolve.NewResolver())

		nodes, err := runner.RunSource(string(source))
		if err != nil {
			t.Errorf("%s returned error: %v", testfile, err)
			return
		}

		evaluator := eval.NewEvaluator()
		var builder strings.Builder
		evaluator.Stdout = &builder
		values := make([]eval.Value, len(nodes))

		for i, node := range nodes {
			values[i], err = evaluator.Eval(node)
			if err != nil {
				t.Errorf("%s returned error: %v", testfile, err)
				return
			}
		}

		if main, ok := evaluator.SearchMain(); ok {
			top := token.Token{Kind: token.IDENT, Lexeme: "toplevel", Line: 0, Literal: -1}
			ret, err := main.Apply(top)
			if err != nil {
				t.Errorf("%s returned error: %v", testfile, err)
				return
			}
			fmt.Fprintf(&builder, "result => %s\n", ret.String())

			g := goldie.New(t)
			g.Assert(t, testfile, []byte(builder.String()))
		} else {
			t.Errorf("%s does not have a main function", testfile)
		}
	}
}
