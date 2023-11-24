package codata_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/takoeight0821/anma/codata"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/utils"
)

func TestFlatFromTestData(t *testing.T) {
	testcases := utils.ReadTestData()

	for _, testcase := range testcases {
		if expected, ok := testcase.Expected["codata"]; ok {
			newCompleteFlat(t, testcase.Label, testcase.Input, expected)
		} else {
			newCompleteFlat(t, testcase.Label, testcase.Input, "no expected value")
		}
	}
}

func newCompleteFlat(t *testing.T, label, input, expected string) {
	r := driver.NewPassRunner()
	r.AddPass(codata.Flat{})

	nodes, err := r.RunSource(input)
	if err != nil {
		t.Errorf("Flat %s returned error: %v", label, err)
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
