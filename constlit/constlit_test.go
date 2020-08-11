package constlit_test

import (
	"testing"

	"github.com/MichaelUrman/mulint/constlit"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, constlit.Analyzer, "a", "b", "c", "d")
}
