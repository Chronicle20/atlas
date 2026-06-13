package matrix

import (
	"path/filepath"
	"testing"
)

func TestLoadReports(t *testing.T) {
	reps, err := LoadReports(filepath.Join("testdata", "audits", "gms_v83"))
	if err != nil {
		t.Fatalf("LoadReports: %v", err)
	}
	r, ok := reps["Invite"]
	if !ok {
		t.Fatalf("Invite report missing; got %v", keysOf(reps))
	}
	if r.IDAName != "CWvsContext::OnFriendResult#Invite" {
		t.Errorf("IDAName = %q", r.IDAName)
	}
	// Packet id derived from AtlasFile + WriterName, with the legacy ../../
	// prefix normalized away.
	if got := PacketID(r); got != "buddy/clientbound/Invite" {
		t.Errorf("PacketID = %q", got)
	}
}

func keysOf(m map[string]LoadedReport) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
