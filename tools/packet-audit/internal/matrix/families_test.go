package matrix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFamilies(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "families.yaml"), []byte(
		"dispatchers:\n  - CCashShop::OnCashItemResult\n  - CShopDlg::OnPacket\n"), 0o644)
	fams, err := LoadFamilies(filepath.Join(dir, "families.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	set := fams.Set()
	if !set["CCashShop::OnCashItemResult"] || !set["CShopDlg::OnPacket"] {
		t.Errorf("dispatcher fnames missing from set: %v", set)
	}
	if set["CMob::OnDamaged"] {
		t.Error("non-dispatcher must not be in the family set")
	}
}

func TestLoadFamiliesMissingFile(t *testing.T) {
	fams, err := LoadFamilies(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if len(fams.Set()) != 0 {
		t.Error("empty families must cap nothing")
	}
}
