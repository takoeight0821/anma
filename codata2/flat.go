package codata2

import "github.com/takoeight0821/anma/ast"

// Flat converts Copatterns ([Access] and [This] in [Pattern]) into [Object] and [Lambda].
type Flat struct{}

func (Flat) Name() string {
	return "codata.Flat"
}

func (Flat) Init([]ast.Node) error {
	return nil
}
