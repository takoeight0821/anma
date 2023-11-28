package parser_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/utils"
)

func TestParseFromTestData(t *testing.T) {
	t.Parallel()
	testcases := utils.ReadTestData()
	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["parser"]; ok {
			newCompleteParse(t, testcase.Label, testcase.Input, expected)
		} else {
			newCompleteParse(t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func newCompleteParse(t *testing.T, label, input, expected string) {
	r := driver.NewPassRunner()

	nodes, err := r.RunSource(input)
	if err != nil {
		t.Errorf("Parser %s returned error: %v", label, err)
	}

	var b strings.Builder
	for _, node := range nodes {
		b.WriteString(node.String())
		b.WriteString("\n")
	}

	actual := b.String()

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("Parser %s mismatch (-want +got):\n%s", label, diff)
	}
}
