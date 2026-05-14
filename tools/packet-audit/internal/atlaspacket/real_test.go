package atlaspacket

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestServerListEntryAnalyze(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	p := filepath.Join(repoRoot, "libs", "atlas-packet", "login", "clientbound", "server_list_entry.go")
	calls, err := AnalyzeFile(p, "ServerListEntry", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	hasRepeat := false
	for _, c := range calls {
		if c.Kind == KindRepeat {
			hasRepeat = true
		}
	}
	if !hasRepeat {
		t.Error("ServerListEntry.Encode should produce a KindRepeat for channelLoads loop")
	}
}

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

func TestAuthSuccessGMSV95Variant(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	p := filepath.Join(repoRoot, "libs", "atlas-packet", "login", "clientbound", "auth_success.go")
	calls, err := AnalyzeFile(p, "AuthSuccess", "Encode")
	if err != nil {
		t.Fatal(err)
	}
	ctx := GuardContext{Region: "GMS", MajorVersion: 95}
	active := 0
	for _, c := range calls {
		if c.Guard == nil || c.Guard.Eval(ctx) {
			active++
		}
	}
	t.Logf("total calls=%d, GMS v95 active=%d", len(calls), active)
	for i, c := range calls {
		guardStr := "<nil>"
		if c.Guard != nil {
			guardStr = c.Guard.String()
		}
		t.Logf("  call[%d] op=%s line=%d guard=%s", i, c.Op, c.Line, guardStr)
	}
	if active < 10 {
		t.Errorf("GMS v95 should activate >=10 calls; got %d", active)
	}
}
