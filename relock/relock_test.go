package relock_test

import (
	"testing"

	"github.com/MichaelUrman/mulint/relock"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, relock.Analyzer, "a")
}
