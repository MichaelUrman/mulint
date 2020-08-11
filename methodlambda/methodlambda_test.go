package methodlambda_test

import (
	"testing"

	"github.com/MichaelUrman/mulint/methodlambda"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, methodlambda.Analyzer, "a")
}
