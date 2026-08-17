package producerseamguard_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Chronicle20/atlas/tools/producerseamguard"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, producerseamguard.Analyzer, "directproduce")
}
