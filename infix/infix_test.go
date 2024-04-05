package infix_test

import (
	"os"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/infix"
	"github.com/takoeight0821/anma/newcodata"
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
		runner.AddPass(&newcodata.Flat{})
		runner.AddPass(infix.NewInfixResolver())

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
		g.Assert(t, testfile, []byte(builder.String()))
	}
}
