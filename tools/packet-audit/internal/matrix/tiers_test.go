package matrix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTierMembership(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tiers.yaml"), []byte(
		"opaque_types:\n  - MovePath\npacket_prefixes:\n  - party/\npackets:\n  - login/clientbound/Special\n"), 0o644)
	tiers, err := LoadTiers(filepath.Join(dir, "tiers.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{
		"party/clientbound/Invite":     true,  // prefix
		"login/clientbound/Special":    true,  // explicit
		"login/clientbound/AuthResult": false, // tier 0
	}
	for pkt, want := range cases {
		if got := tiers.IsTier1(pkt, nil); got != want {
			t.Errorf("IsTier1(%s) = %v, want %v", pkt, got, want)
		}
	}
	// opaque recursion: recurseTypes lists the packet's transitive RecurseType set
	if !tiers.IsTier1("monster/clientbound/Move", []string{"MovePath"}) {
		t.Error("opaque-type recursion must be tier 1")
	}
}

func TestLoadTiersMissingFile(t *testing.T) {
	tiers, err := LoadTiers(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	// Everything is tier 0 when no tiers file exists.
	if tiers.IsTier1("party/clientbound/Invite", nil) {
		t.Error("empty tiers must not classify anything as tier-1")
	}
}
