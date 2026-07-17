package workers

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("<imgdir/>"), 0o644); err != nil {
		t.Fatalf("touch %s: %v", name, err)
	}
}

func TestResolveStringSourcesModern(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "Consume.img.xml")
	touch(t, dir, "Eqp.img.xml")
	touch(t, dir, "Item.img.xml") // legacy present too — modern must win
	src := resolveStringSources(dir)
	if len(src.flat) != 1 || filepath.Base(src.flat[0]) != "Consume.img.xml" {
		t.Fatalf("flat = %v, want [Consume.img.xml]", src.flat)
	}
	if src.eqp == "" {
		t.Fatalf("eqp not resolved")
	}
	if src.legacyItem != "" {
		t.Fatalf("legacy adapter must not engage when modern images exist")
	}
}

func TestResolveStringSourcesLegacy(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "Item.img.xml")
	src := resolveStringSources(dir)
	if len(src.flat) != 0 || src.eqp != "" {
		t.Fatalf("modern sources = %v/%q, want none", src.flat, src.eqp)
	}
	if filepath.Base(src.legacyItem) != "Item.img.xml" {
		t.Fatalf("legacyItem = %q, want Item.img.xml", src.legacyItem)
	}
}

func TestResolveStringSourcesEmpty(t *testing.T) {
	src := resolveStringSources(t.TempDir())
	if len(src.flat) != 0 || src.eqp != "" || src.legacyItem != "" {
		t.Fatalf("empty dir must resolve no sources, got %+v", src)
	}
}
