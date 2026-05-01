package image

import (
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractInfoBlock(t *testing.T) {
	props := []property.Property{
		property.NewSub("info", []property.Property{
			property.NewString("islot", "Cp"),
			property.NewString("vslot", "Cp"),
			property.NewInt("cash", 0),
		}),
	}
	got := extractInfoBlock(props)
	if got.Islot != "Cp" || got.Vslot != "Cp" || got.Cash != 0 {
		t.Fatalf("unexpected info: %+v", got)
	}
}

func TestWriteInfoJSON(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "1002357")
	if err := writeInfoJSON(target, templateInfo{Islot: "Cp", Vslot: "Cp", Cash: 0}); err != nil {
		t.Fatalf("write: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(target, "info.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var ti templateInfo
	if err := json.Unmarshal(b, &ti); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ti.Islot != "Cp" {
		t.Fatalf("islot = %q", ti.Islot)
	}
}
