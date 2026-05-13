package atlaspacket

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestAuthSuccessEncodeExtracts(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	p := filepath.Join(repoRoot, "libs", "atlas-packet", "login", "clientbound", "auth_success.go")
	calls, err := AnalyzeFile(p, "AuthSuccess", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) < 10 {
		t.Errorf("calls: got %d, want >=10", len(calls))
	}
}
