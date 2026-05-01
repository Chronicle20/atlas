package characterimage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadInfoUnknownTemplateId(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadInfo(dir, "9999999")
	if !errors.Is(err, ErrUnknownTemplateId) {
		t.Fatalf("expected ErrUnknownTemplateId, got %v", err)
	}
}

func TestLoadInfoRoundTrip(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "character-parts", "1002357")
	if err := os.MkdirAll(tmpl, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpl, "info.json"),
		[]byte(`{"islot":"Cp","vslot":"Cp","cash":0}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	ti, err := LoadInfo(dir, "1002357")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if ti.Islot != "Cp" || ti.Vslot != "Cp" {
		t.Fatalf("got %+v", ti)
	}
}

func TestMetaCacheMemoizes(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "character-parts", "1002357")
	_ = os.MkdirAll(tmpl, 0o755)
	_ = os.WriteFile(filepath.Join(tmpl, "info.json"),
		[]byte(`{"islot":"Cp","vslot":"Cp","cash":0}`), 0o644)

	c := newMetaCache()
	a, _ := c.info(dir, "1002357")
	// Delete the file; cached value must still be returned (same assetsRoot).
	_ = os.RemoveAll(tmpl)
	b, err := c.info(dir, "1002357")
	if err != nil {
		t.Fatalf("second call errored: %v", err)
	}
	if a != b {
		t.Fatalf("cache miss: %+v vs %+v", a, b)
	}
}

func TestMetaCachePartitionedByAssetsRoot(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// Write the same templateId to two different assetsRoots with different islot values.
	for _, tc := range []struct {
		dir   string
		islot string
	}{
		{dir1, "Cp"},
		{dir2, "Hr"},
	} {
		tmpl := filepath.Join(tc.dir, "character-parts", "1002357")
		_ = os.MkdirAll(tmpl, 0o755)
		_ = os.WriteFile(filepath.Join(tmpl, "info.json"),
			[]byte(`{"islot":"`+tc.islot+`","vslot":"`+tc.islot+`","cash":0}`), 0o644)
	}

	c := newMetaCache()
	a, err := c.info(dir1, "1002357")
	if err != nil {
		t.Fatalf("dir1 load: %v", err)
	}
	b, err := c.info(dir2, "1002357")
	if err != nil {
		t.Fatalf("dir2 load: %v", err)
	}
	if a.Islot != "Cp" {
		t.Fatalf("dir1: expected islot=Cp, got %q", a.Islot)
	}
	if b.Islot != "Hr" {
		t.Fatalf("dir2: expected islot=Hr, got %q", b.Islot)
	}
	// Each assetsRoot must retain its own cached value after the file is gone.
	_ = os.RemoveAll(filepath.Join(dir1, "character-parts", "1002357"))
	_ = os.RemoveAll(filepath.Join(dir2, "character-parts", "1002357"))
	a2, err := c.info(dir1, "1002357")
	if err != nil {
		t.Fatalf("dir1 second call: %v", err)
	}
	b2, err := c.info(dir2, "1002357")
	if err != nil {
		t.Fatalf("dir2 second call: %v", err)
	}
	if a2.Islot != "Cp" {
		t.Fatalf("dir1 cached: expected Cp, got %q", a2.Islot)
	}
	if b2.Islot != "Hr" {
		t.Fatalf("dir2 cached: expected Hr, got %q", b2.Islot)
	}
}
