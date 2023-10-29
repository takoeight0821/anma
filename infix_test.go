package main_test

import (
	"testing"

	. "github.com/takoeight0821/anma"
)

func completeInfix(t *testing.T, input1, input2, expected string) {
	runner := NewPassRunner()
	runner.AddPass(Flat{})
	runner.AddPass(NewInfixResolver())

	_, err := runner.RunSource(input1)
	if err != nil {
		t.Errorf("RunSource returned error: %v", err)
	}

	node, err := runner.RunSource(input2)
	if err != nil {
		t.Errorf("RunSource returned error: %v", err)
	}

	actual := node[0].String()

	if actual != expected {
		t.Errorf("InfixResolver returned\n%q, expected\n%q", actual, expected)
	}
}

func TestInfix(t *testing.T) {
	testcases := []struct {
		input1   string
		input2   string
		expected string
	}{
		{"infixl 6 +\ninfixl 8 *", "1 + 2 * 3", "(binary (literal 1) + (binary (literal 2) * (literal 3)))"},
		{"infixl 6 +\ninfixl 8 *", "1 * 2 + 3", "(binary (binary (literal 1) * (literal 2)) + (literal 3))"},
	}
	for _, testcase := range testcases {
		completeInfix(t, testcase.input1, testcase.input2, testcase.expected)
	}
}
