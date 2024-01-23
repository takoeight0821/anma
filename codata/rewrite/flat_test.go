package rewrite_test

import (
	"os"
	"testing"

	"github.com/takoeight0821/anma/codata/rewrite"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/utils"
)

func TestFlatFromTestData(t *testing.T) {
	t.Skip()
	t.Parallel()
	s, err := os.ReadFile("../../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}
	testcases := utils.ReadTestData(s)

	for _, testcase := range testcases {
		runner := driver.NewPassRunner()
		runner.AddPass(&rewrite.Flat{})
		if expected, ok := testcase.Expected["codata"]; ok {
			utils.RunTest(runner, t, testcase.Label, testcase.Input, expected)
		} else {
			utils.RunTest(runner, t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func BenchmarkFromTestData(b *testing.B) {
	s, err := os.ReadFile("../../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}
	testcases := utils.ReadTestData(s)

	for _, testcase := range testcases {
		b.Run(testcase.Label, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				runner := driver.NewPassRunner()
				runner.AddPass(&rewrite.Flat{})
				utils.RunTest(runner, b, testcase.Label, testcase.Input, testcase.Expected["codata"])
			}
		})
	}
}
