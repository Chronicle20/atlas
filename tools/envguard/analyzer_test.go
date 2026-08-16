package envguard_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Chronicle20/atlas/tools/envguard"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, envguard.Analyzer, "domainimport", "ok/kafka")
}
