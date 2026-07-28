package goroutineguard_test

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/goroutineguard"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, goroutineguard.Analyzer,
		"bad", "good", "github.com/Chronicle20/atlas/libs/atlas-routine")
}
