package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/MichaelUrman/mulint/constlit"
	"github.com/MichaelUrman/mulint/methodlambda"
)

func main() {
	multichecker.Main(
		constlit.Analyzer,
		methodlambda.Analyzer,
	)
}
