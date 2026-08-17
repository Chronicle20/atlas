package rediskeyguard_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Chronicle20/atlas/tools/rediskeyguard"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, rediskeyguard.Analyzer, "bad", "good", "bareconstructor")
}
