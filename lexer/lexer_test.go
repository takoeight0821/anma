package lexer_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/takoeight0821/anma/lexer"
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

		tokens, err := lexer.Lex(testfile, string(source))
		if err != nil {
			t.Errorf("%s returned error: %v", testfile, err)

			return
		}

		var builder strings.Builder
		for _, token := range tokens {
			fmt.Fprintf(&builder, "%v %q %v\n", token.Kind, token.Lexeme, token.Location)
		}

		g := goldie.New(t)
		g.Assert(t, testfile, []byte(builder.String()))
	}
}
