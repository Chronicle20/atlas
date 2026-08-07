package buffdurationguard_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Chronicle20/atlas/tools/buffdurationguard"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, buffdurationguard.Analyzer, "bad", "good")
}
