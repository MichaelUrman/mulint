package methodlambda

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
)

type sourcer struct {
	Fset *token.FileSet
	Node ast.Node
}

var _ fmt.Formatter = sourcer{}

func (s sourcer) Format(f fmt.State, r rune) {
	if f.Flag('-') {
		fmt.Fprint(f, s.Fset.Position(s.Node.Pos()).String())
	} else {
		printer.Fprint(f, s.Fset, s.Node)
	}
}
