package main_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	. "github.com/takoeight0821/anma"
	"github.com/takoeight0821/anma/internal/codata"
	"github.com/takoeight0821/anma/internal/driver"
)

func TestEvaluator(t *testing.T) {
	completeEval(t, "1", "1")
}

func completeEval(t *testing.T, input, expected string) {
	runner := driver.NewPassRunner()
	runner.AddPass(codata.Flat{})
	runner.AddPass(NewInfixResolver())
	runner.AddPass(NewRenamer())
	runner.AddPass(NewEvaluator())

	var err error
	result := captureStdout(t, func() {
		_, err = runner.RunSource(input)
	})
	if err != nil {
		t.Errorf("RunSource returned error: %v", err)
	}
	if strings.TrimRight(result, "\n") != expected {
		t.Errorf("RunSource returned %q, want %q", result, expected)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	defer func() { os.Stdout = old }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}

	os.Stdout = w
	fn()
	w.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("buf.ReadFrom() returned error: %v", err)
	}

	return buf.String()
}
