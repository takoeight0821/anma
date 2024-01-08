package rewrite

import (
	"fmt"

	"github.com/takoeight0821/anma/ast"
)

type InvalidPatternError struct {
	Patterns []ast.Node
}

func (e *InvalidPatternError) Error() string {
	return fmt.Sprintf("invalid pattern: %v", e.Patterns)
}

func NewInvalidPatternError(patterns ...ast.Node) *InvalidPatternError {
	return &InvalidPatternError{Patterns: patterns}
}
