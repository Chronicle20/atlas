package opregistry

import (
	"path/filepath"
	"testing"
)

func TestLoadVersion(t *testing.T) {
	v, err := LoadVersion(filepath.Join("testdata", "good_version.yaml"))
	if err != nil {
		t.Fatalf("LoadVersion: %v", err)
	}
	e, ok := v.Lookup("LOGIN_STATUS", DirClientbound)
	if !ok {
		t.Fatalf("LOGIN_STATUS clientbound not found")
	}
	if e.Opcode != 0x000 || e.FName != "CLogin::OnCheckPasswordResult" {
		t.Errorf("entry = %+v", e)
	}
	if e.Provenance != "csv-import" {
		t.Errorf("provenance = %q", e.Provenance)
	}
	// fname_alts round-trip (multiline CSV cells)
	alt, _ := v.Lookup("SERVERLIST_REREQUEST", DirServerbound)
	if len(alt.FNameAlts) != 1 || alt.FNameAlts[0] != "CLogin::ChangeStepImmediate" {
		t.Errorf("fname_alts = %v", alt.FNameAlts)
	}
}

func TestLoadVersionDuplicate(t *testing.T) {
	_, err := LoadVersion(filepath.Join("testdata", "dup_version.yaml"))
	if err == nil {
		t.Fatal("expected duplicate (op,direction) error")
	}
}

func TestRegistryApplicability(t *testing.T) {
	r := Registry{Versions: map[string]*VersionFile{
		"gms_v83": mustLoad(t, "good_version.yaml"),
	}}
	if got := r.Applicability("LOGIN_STATUS", DirClientbound, "gms_v83"); got != Present {
		t.Errorf("present op = %v", got)
	}
	if got := r.Applicability("NOPE", DirClientbound, "gms_v83"); got != Absent {
		t.Errorf("missing op in existing file = %v", got)
	}
	if got := r.Applicability("LOGIN_STATUS", DirClientbound, "gms_v99"); got != Unknown {
		t.Errorf("missing version file = %v", got)
	}
}

func mustLoad(t *testing.T, name string) *VersionFile {
	t.Helper()
	v, err := LoadVersion(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return v
}
