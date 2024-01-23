package nameresolve_test

import (
	"os"
	"testing"

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
			for i := 0; i < b.N; i++ {
				runner := driver.NewPassRunner()
				runner.AddPass(codata.Flat{})
				runner.AddPass(infix.NewInfixResolver())
				runner.AddPass(nameresolve.NewResolver())

				utils.RunTest(runner, b, testcase.Label, testcase.Input, testcase.Expected["nameresolve"])
			}
		})
	}
}
