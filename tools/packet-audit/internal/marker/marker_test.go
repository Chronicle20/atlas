package marker

import (
	"os"
	"path/filepath"
	"testing"
)

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestScanFile(t *testing.T) {
	ms, errs := scanReader(mustOpen(t, filepath.Join("testdata", "invite_test.go.txt")), "invite_test.go")
	if len(ms) != 2 {
		t.Fatalf("markers = %d, want 2 (%v)", len(ms), ms)
	}
	if ms[0].Packet != "buddy/clientbound/Invite" || ms[0].Version != "gms_v83" || ms[0].Address != "0xa3e31c" {
		t.Errorf("marker0 = %+v", ms[0])
	}
	if ms[0].File != "invite_test.go" || ms[0].Line != 5 {
		t.Errorf("marker0 location = %s:%d", ms[0].File, ms[0].Line)
	}
	// Malformed marker (missing version/ida) is an error, not a silent skip.
	if len(errs) != 1 {
		t.Errorf("errs = %v, want 1 malformed-marker error", errs)
	}
}

func TestScanDuplicateMarkerIsError(t *testing.T) {
	ms, errs := scanString("// packet-audit:verify packet=a/b/C version=gms_v83 ida=0x1\n// packet-audit:verify packet=a/b/C version=gms_v83 ida=0x2\n", "x_test.go")
	_ = ms
	if len(errs) == 0 {
		t.Error("duplicate (packet,version) across markers must error (design §7: one marker per cell)")
	}
}
