package nameresolve_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/nameresolve"
	"github.com/takoeight0821/anma/utils"
)

func TestResolve(t *testing.T) {
	t.Parallel()
	s, err := os.ReadFile("../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}
	testcases := utils.ReadTestData(s)
	for _, testcase := range testcases {
		runner := driver.NewPassRunner()
		runner.AddPass(codata.Flat{})
		runner.AddPass(infix.NewInfixResolver())
		runner.AddPass(nameresolve.NewResolver())

		if expected, ok := testcase.Expected["nameresolve"]; ok {
			utils.RunTest(runner, t, testcase.Label, testcase.Input, expected)
		} else {
			utils.RunTest(runner, t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func BenchmarkFromTestData(b *testing.B) {
	s, err := os.ReadFile("../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}
	testcases := utils.ReadTestData(s)

	for _, testcase := range testcases {
		b.Run(testcase.Label, func(b *testing.B) {
			for range b.N {
				runner := driver.NewPassRunner()
				runner.AddPass(codata.Flat{})
				runner.AddPass(infix.NewInfixResolver())
				runner.AddPass(nameresolve.NewResolver())

				utils.RunTest(runner, b, testcase.Label, testcase.Input, testcase.Expected["nameresolve"])
			}
		})
	}
}

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
		runner.AddPass(codata.Flat{})
		runner.AddPass(infix.NewInfixResolver())
		runner.AddPass(nameresolve.NewResolver())

		nodes, err := runner.RunSource(string(source))
		if err != nil {
			t.Errorf("%s returned error: %v", testfile, err)
			return
		}

		var builder strings.Builder
		for _, node := range nodes {
			builder.WriteString(node.String())
			builder.WriteString("\n")
		}

		g := goldie.New(t)
		g.Assert(t, filepath.Base(testfile), []byte(builder.String()))
	}
}
