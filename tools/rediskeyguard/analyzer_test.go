package rediskeyguard_test

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/rediskeyguard"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, rediskeyguard.Analyzer, "bad", "good")
}
