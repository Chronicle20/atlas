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

// TestResolveStringSourcesModern is the JMS v185 / v83+ shape: per-category
// images, Eqp nested, NO single Item.img. The modern branch must engage.
func TestResolveStringSourcesModern(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "Consume.img.xml")
	touch(t, dir, "Cash.img.xml")
	touch(t, dir, "Etc.img.xml")
	touch(t, dir, "Ins.img.xml")
	touch(t, dir, "Pet.img.xml")
	touch(t, dir, "Eqp.img.xml")
	src := resolveStringSources(dir)
	if len(src.flat) != 5 {
		t.Fatalf("flat = %v, want 5 modern images", src.flat)
	}
	if src.eqp == "" {
		t.Fatalf("eqp not resolved")
	}
	if src.legacyItem != "" {
		t.Fatalf("legacy adapter must not engage when there is no Item.img")
	}
}

// TestResolveStringSourcesLegacyItemImgWins is the GMS v12/v48 regression:
// those ship a single Item.img (Con/Ins/Etc/Eqp/Pet) AND a standalone Pet.img.
// The presence of Pet.img.xml must NOT be mistaken for a modern layout —
// Item.img is the authoritative source and must win, or every non-pet item
// name is dropped (task-172 C-4, caught in E2E).
func TestResolveStringSourcesLegacyItemImgWins(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "Item.img.xml")
	touch(t, dir, "Pet.img.xml") // standalone pet strings coexist with legacy Item.img
	src := resolveStringSources(dir)
	if filepath.Base(src.legacyItem) != "Item.img.xml" {
		t.Fatalf("legacyItem = %q, want Item.img.xml", src.legacyItem)
	}
	if len(src.flat) != 0 || src.eqp != "" {
		t.Fatalf("modern sources = %v/%q, want none when Item.img present", src.flat, src.eqp)
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
