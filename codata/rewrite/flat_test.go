package rewrite_test

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
		if expected, ok := testcase.Expected["codata"]; ok {
			completeFlat(t, testcase.Label, testcase.Input, expected)
		} else {
			completeFlat(t, testcase.Label, testcase.Input, "no expected value")
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
				completeFlat(b, testcase.Label, testcase.Input, testcase.Expected["codata"])
			}
		})
	}
}

type reporter interface {
	Errorf(format string, args ...interface{})
}

func completeFlat(t reporter, label, input, expected string) {
	r := driver.NewPassRunner()
	r.AddPass(&rewrite.Flat{})

	nodes, err := r.RunSource(input)
	if err != nil {
		t.Errorf("Flat %s returned error: %v", label, err)
		return
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

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("Flat %s mismatch (-want +got):\n%s", label, diff)
	}
}
