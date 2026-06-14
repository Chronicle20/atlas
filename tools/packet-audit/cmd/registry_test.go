package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

func TestRegistrySeed(t *testing.T) {
	out := t.TempDir()
	code := runRegistry([]string{
		"seed",
		"--clientbound", filepath.Join("..", "internal", "seedcsv", "testdata", "clientbound_excerpt.csv"),
		"--serverbound", filepath.Join("..", "internal", "seedcsv", "testdata", "serverbound_excerpt.csv"),
		"--out", out,
	}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("seed exit = %d", code)
	}
	for _, vk := range []string{"gms_v83", "gms_v84", "gms_v87", "gms_v95", "jms_v185"} {
		vf, err := opregistry.LoadVersion(filepath.Join(out, vk+".yaml"))
		if err != nil {
			t.Fatalf("%s: %v", vk, err)
		}
		if _, ok := vf.Lookup("LOGIN_STATUS", opregistry.DirClientbound); !ok {
			t.Errorf("%s: LOGIN_STATUS missing", vk)
		}
	}
	// v84 mirrors v83 (no CSV column).
	v84, err := opregistry.LoadVersion(filepath.Join(out, "gms_v84.yaml"))
	if err != nil {
		t.Fatalf("gms_v84 load: %v", err)
	}
	e, ok := v84.Lookup("GUEST_LOGIN", opregistry.DirServerbound)
	if !ok || e.Opcode != 0x002 {
		t.Errorf("v84 GUEST_LOGIN = %+v ok=%v (want copy of v83)", e, ok)
	}
	// ACCOUNT_INFO absent in jms_v185 → no entry.
	jms, err := opregistry.LoadVersion(filepath.Join(out, "jms_v185.yaml"))
	if err != nil {
		t.Fatalf("jms_v185 load: %v", err)
	}
	if _, ok := jms.Lookup("ACCOUNT_INFO", opregistry.DirClientbound); ok {
		t.Errorf("jms_v185 must not contain ACCOUNT_INFO")
	}
	// n/a rows in serverbound_excerpt must produce UNNAMED_R<line> with empty fname,
	// pass self-validation (LoadVersion accepts them), and not duplicate each other.
	v83, err := opregistry.LoadVersion(filepath.Join(out, "gms_v83.yaml"))
	if err != nil {
		t.Fatalf("gms_v83 load: %v", err)
	}
	unnamedEntry, unnamedOk := v83.Lookup("UNNAMED_R5", opregistry.DirServerbound)
	if !unnamedOk {
		t.Error("gms_v83: expected UNNAMED_R5 serverbound entry for n/a row")
	} else {
		if unnamedEntry.FName != "" {
			t.Errorf("UNNAMED_R5 fname = %q, want empty", unnamedEntry.FName)
		}
		if len(unnamedEntry.FNameAlts) != 0 {
			t.Errorf("UNNAMED_R5 fname_alts = %v, want nil/empty", unnamedEntry.FNameAlts)
		}
		if !strings.Contains(unnamedEntry.Note, "op name unknown") {
			t.Errorf("UNNAMED_R5 note %q missing expected text", unnamedEntry.Note)
		}
	}
	// Determinism: seeding twice produces identical bytes.
	b1, err := os.ReadFile(filepath.Join(out, "gms_v83.yaml"))
	if err != nil {
		t.Fatalf("ReadFile gms_v83.yaml: %v", err)
	}
	out2 := t.TempDir()
	code2 := runRegistry([]string{"seed",
		"--clientbound", filepath.Join("..", "internal", "seedcsv", "testdata", "clientbound_excerpt.csv"),
		"--serverbound", filepath.Join("..", "internal", "seedcsv", "testdata", "serverbound_excerpt.csv"),
		"--out", out2}, &bytes.Buffer{})
	if code2 != 0 {
		t.Fatalf("second seed exit = %d", code2)
	}
	b2, err := os.ReadFile(filepath.Join(out2, "gms_v83.yaml"))
	if err != nil {
		t.Fatalf("ReadFile gms_v83.yaml (second run): %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Error("seed output not deterministic")
	}
}
