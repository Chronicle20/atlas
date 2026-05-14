package csv

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadRealCSVs(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	cb := filepath.Join(repoRoot, "docs", "packets", "MapleStory Ops - ClientBound.csv")
	sb := filepath.Join(repoRoot, "docs", "packets", "MapleStory Ops - ServerBound.csv")

	cbm, err := Load(cb, DirClientbound)
	if err != nil {
		t.Fatalf("clientbound: %v", err)
	}
	if _, ok := cbm.ByFName("CLogin::OnCheckPasswordResult"); !ok {
		t.Errorf("clientbound real CSV: CLogin::OnCheckPasswordResult missing")
	}

	if _, err := Load(sb, DirServerbound); err != nil {
		t.Fatalf("serverbound: %v", err)
	}
}
