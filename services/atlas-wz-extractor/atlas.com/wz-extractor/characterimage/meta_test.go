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
	// Delete the file; cached value must still be returned.
	_ = os.RemoveAll(tmpl)
	b, err := c.info(dir, "1002357")
	if err != nil {
		t.Fatalf("second call errored: %v", err)
	}
	if a != b {
		t.Fatalf("cache miss: %+v vs %+v", a, b)
	}
}
