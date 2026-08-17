package envguard_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Chronicle20/atlas/tools/envguard"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, envguard.Analyzer,
		"domainimport",               // violation: not main.go, not kafka/rest/socket, not allowlisted
		"ok/kafka",                   // allowed: kafka/ path segment
		"ok/rest",                    // allowed: rest/ path segment
		"ok/socket",                  // allowed: socket/ path segment (FR-2.2)
		"mainok",                     // allowed: main.go
		"atlas-configurations/scope", // allowed: domainAllowlist hit
	)
}
