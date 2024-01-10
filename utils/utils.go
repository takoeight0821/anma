package utils

import (
	"fmt"

	"github.com/takoeight0821/anma/token"
	"gopkg.in/yaml.v3"
)

type ErrorAt struct {
	Where token.Token
	Err   error
}

func (e ErrorAt) Error() string {
	if e.Where.Kind == token.EOF {
		return fmt.Sprintf("at end: %s", e.Err.Error())
	}
	return fmt.Sprintf("at %d: `%s`, %s", e.Where.Line, e.Where.Lexeme, e.Err.Error())
}

type TestData struct {
	Label    string
	Enable   bool
	Input    string
	Expected map[string]string
}

func ReadTestData(s []byte) []TestData {
	var data []TestData
	if err := yaml.Unmarshal(s, &data); err != nil {
		panic(err)
	}

	// Remove disabled test cases.
	i := 0
	for _, d := range data {
		if d.Enable {
			data[i] = d
			i++
		}
	}
	data = data[:i]

	return data
}
