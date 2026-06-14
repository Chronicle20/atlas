package opregistry

import (
	"os"
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

// TestByFName covers exact match, fname_alts match, and wrong-direction miss.
func TestByFName(t *testing.T) {
	v := mustLoad(t, "good_version.yaml")

	// Exact fname match.
	e, ok := v.ByFName("CLogin::OnCheckPasswordResult", DirClientbound)
	if !ok {
		t.Fatalf("ByFName exact: not found")
	}
	if e.Op != "LOGIN_STATUS" {
		t.Errorf("ByFName exact: op = %q, want LOGIN_STATUS", e.Op)
	}

	// fname_alts match.
	e, ok = v.ByFName("CLogin::ChangeStepImmediate", DirServerbound)
	if !ok {
		t.Fatalf("ByFName fname_alts: not found")
	}
	if e.Op != "SERVERLIST_REREQUEST" {
		t.Errorf("ByFName fname_alts: op = %q, want SERVERLIST_REREQUEST", e.Op)
	}

	// Wrong direction: clientbound FName looked up as serverbound → miss.
	_, ok = v.ByFName("CLogin::OnCheckPasswordResult", DirServerbound)
	if ok {
		t.Error("ByFName wrong direction: expected miss")
	}
}

// TestLoadDir verifies: missing file skipped (Unknown applicability); present
// file loaded; a bad file returns an error.
func TestLoadDir(t *testing.T) {
	dir := t.TempDir()

	// Copy good_version.yaml into the temp dir as gms_v83.yaml.
	goodContent, err := os.ReadFile(filepath.Join("testdata", "good_version.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gms_v83.yaml"), goodContent, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	// gms_v84 is intentionally absent.
	keys := []string{"gms_v83", "gms_v84"}
	r, err := LoadDir(dir, keys)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	// gms_v83 loaded.
	if r.Versions["gms_v83"] == nil {
		t.Error("gms_v83 not loaded")
	}
	// gms_v84 absent → Unknown applicability.
	if got := r.Applicability("LOGIN_STATUS", DirClientbound, "gms_v84"); got != Unknown {
		t.Errorf("missing file applicability = %v, want Unknown", got)
	}

	// Bad file causes LoadDir to return an error.
	badContent, err := os.ReadFile(filepath.Join("testdata", "bad_yaml.yaml"))
	if err != nil {
		t.Fatalf("read bad fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gms_v87.yaml"), badContent, 0o644); err != nil {
		t.Fatalf("write bad fixture: %v", err)
	}
	_, err = LoadDir(dir, []string{"gms_v87"})
	if err == nil {
		t.Error("LoadDir with bad YAML: expected error")
	}
}

// TestAllOpsDeterministic verifies AllOps returns clientbound before serverbound
// and that op names are sorted within each direction group.
func TestAllOpsDeterministic(t *testing.T) {
	r := Registry{Versions: map[string]*VersionFile{
		"gms_v83": mustLoad(t, "good_version.yaml"),
	}}
	ops := r.AllOps()
	if len(ops) == 0 {
		t.Fatal("AllOps: empty")
	}
	// clientbound entries must precede serverbound.
	seenServerbound := false
	for _, o := range ops {
		if o.Dir == DirServerbound {
			seenServerbound = true
		}
		if seenServerbound && o.Dir == DirClientbound {
			t.Errorf("AllOps ordering: clientbound entry %q after serverbound", o.Op)
		}
	}
	// Within each direction group the op names must be sorted.
	for i := 1; i < len(ops); i++ {
		if ops[i].Dir == ops[i-1].Dir && ops[i].Op < ops[i-1].Op {
			t.Errorf("AllOps not sorted: %q > %q in %s group", ops[i-1].Op, ops[i].Op, ops[i].Dir)
		}
	}
}

// TestValidationErrors verifies that invalid provenance and ida-discovered
// without ida.address are rejected.
func TestValidationErrors(t *testing.T) {
	t.Run("invalid_provenance", func(t *testing.T) {
		_, err := LoadVersion(filepath.Join("testdata", "bad_provenance.yaml"))
		if err == nil {
			t.Fatal("expected error for invalid provenance")
		}
	})
	t.Run("ida_no_address", func(t *testing.T) {
		_, err := LoadVersion(filepath.Join("testdata", "ida_no_address.yaml"))
		if err == nil {
			t.Fatal("expected error for ida-discovered without ida.address")
		}
	})
	t.Run("malformed_yaml", func(t *testing.T) {
		_, err := LoadVersion(filepath.Join("testdata", "bad_yaml.yaml"))
		if err == nil {
			t.Fatal("expected error for malformed YAML")
		}
	})
}

func mustLoad(t *testing.T, name string) *VersionFile {
	t.Helper()
	v, err := LoadVersion(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return v
}
