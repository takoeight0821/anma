package parser_test

import (
	"strings"
	"testing"

	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/utils"
)

func TestParseFromTestData(t *testing.T) {
	t.Parallel()
	testcases := utils.ReadTestData()
	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["parser"]; ok {
			completeParse(t, testcase.Label, testcase.Input, expected)
		} else {
			completeParse(t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func BenchmarkFromTestData(b *testing.B) {
	testcases := utils.ReadTestData()

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

func completeParse(t reporter, label, input, expected string) {
	r := driver.NewPassRunner()

	nodes, err := r.RunSource(input)
	if err != nil {
		t.Errorf("Parser %s returned error: %v", label, err)
	}

	if _, ok := t.(*testing.B); ok {
		// do nothing for benchmark
		return
	}

	var b strings.Builder
	for _, node := range nodes {
		b.WriteString(node.String())
		b.WriteString("\n")
	}

	actual := b.String()

	if diff := utils.Diff(expected, actual); diff != "" {
		t.Errorf("Parser %s mismatch (-want +got):\n%s", label, diff)
	}
}
