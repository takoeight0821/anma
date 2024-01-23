package parser_test

import (
	"os"
	"strings"
	"testing"

	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/utils"
)

func TestParseFromTestData(t *testing.T) {
	t.Parallel()
	s, err := os.ReadFile("../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}
	testcases := utils.ReadTestData(s)
	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["parser"]; ok {
			completeParse(t, testcase.Label, testcase.Input, expected)
		} else {
			completeParse(t, testcase.Label, testcase.Input, "no expected value")
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
				completeParse(b, testcase.Label, testcase.Input, testcase.Expected["parser"])
			}
		})
	}
}

type reporter interface {
	Errorf(format string, args ...interface{})
}

func completeParse(test reporter, label, input, expected string) {
	r := driver.NewPassRunner()

	nodes, err := r.RunSource(input)
	if err != nil {
		test.Errorf("Parser %s returned error: %v", label, err)
	}

	if _, ok := test.(*testing.B); ok {
		// do nothing for benchmark
		return
	}

	var builder strings.Builder
	for _, node := range nodes {
		builder.WriteString(node.String())
		builder.WriteString("\n")
	}

	actual := builder.String()

	if diff := utils.Diff(expected, actual); diff != "" {
		test.Errorf("Parser %s mismatch (-want +got):\n%s", label, diff)
	}
}
