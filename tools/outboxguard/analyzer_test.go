package outboxguard_test

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/outboxguard"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, outboxguard.Analyzer, "guardtest")
}
