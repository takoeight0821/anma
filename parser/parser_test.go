package parser_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/takoeight0821/anma/driver"
	"github.com/takoeight0821/anma/utils"

	"github.com/sebdah/goldie/v2"
)

func TestParseFromTestData(t *testing.T) {
	t.Parallel()
	s, err := os.ReadFile("../testdata/testcase.yaml")
	if err != nil {
		panic(err)
	}
	testcases := utils.ReadTestData(s)
	for _, testcase := range testcases {
		runner := driver.NewPassRunner()
		if expected, ok := testcase.Expected["parser"]; ok {
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
				utils.RunTest(runner, b, testcase.Label, testcase.Input, testcase.Expected["parser"])
			}
		})
	}
}

func TestGolden(t *testing.T) {
	t.Parallel()

	var testfiles []string
	filepath.WalkDir("../testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".anma" {
			testfiles = append(testfiles, path)
		}
		return nil
	})

	runner := driver.NewPassRunner()

	for _, testfile := range testfiles {
		source, err := os.ReadFile(testfile)
		if err != nil {
			t.Errorf("failed to read %s: %v", testfile, err)
			return
		}

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
