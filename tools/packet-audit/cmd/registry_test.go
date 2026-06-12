package cmd

import (
	"bytes"
	"os"
	"path/filepath"
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
	v84, _ := opregistry.LoadVersion(filepath.Join(out, "gms_v84.yaml"))
	e, ok := v84.Lookup("GUEST_LOGIN", opregistry.DirServerbound)
	if !ok || e.Opcode != 0x002 {
		t.Errorf("v84 GUEST_LOGIN = %+v ok=%v (want copy of v83)", e, ok)
	}
	// ACCOUNT_INFO absent in jms_v185 → no entry.
	jms, _ := opregistry.LoadVersion(filepath.Join(out, "jms_v185.yaml"))
	if _, ok := jms.Lookup("ACCOUNT_INFO", opregistry.DirClientbound); ok {
		t.Errorf("jms_v185 must not contain ACCOUNT_INFO")
	}
	// Determinism: seeding twice produces identical bytes.
	b1, _ := os.ReadFile(filepath.Join(out, "gms_v83.yaml"))
	out2 := t.TempDir()
	runRegistry([]string{"seed",
		"--clientbound", filepath.Join("..", "internal", "seedcsv", "testdata", "clientbound_excerpt.csv"),
		"--serverbound", filepath.Join("..", "internal", "seedcsv", "testdata", "serverbound_excerpt.csv"),
		"--out", out2}, &bytes.Buffer{})
	b2, _ := os.ReadFile(filepath.Join(out2, "gms_v83.yaml"))
	if !bytes.Equal(b1, b2) {
		t.Error("seed output not deterministic")
	}
}
