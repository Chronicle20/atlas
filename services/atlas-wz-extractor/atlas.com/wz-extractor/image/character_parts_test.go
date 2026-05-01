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

func TestBuildPartSidecar(t *testing.T) {
	children := []property.Property{
		property.NewVector("origin", 19, 32),
		property.NewString("z", "body"),
		property.NewString("group", "skin"),
		property.NewInt("delay", 180),
		property.NewShort("face", 1),
		property.NewSub("map", []property.Property{
			property.NewVector("neck", -4, -32),
			property.NewVector("navel", -6, -20),
		}),
	}
	got := buildPartSidecar(children)
	if got.Origin != (vec{X: 19, Y: 32}) {
		t.Fatalf("origin = %+v", got.Origin)
	}
	if got.Z != "body" || got.Group != "skin" || got.Delay != 180 || got.Face != 1 {
		t.Fatalf("scalar mismatch: %+v", got)
	}
	if got.Map["neck"] != (vec{X: -4, Y: -32}) {
		t.Fatalf("map.neck = %+v", got.Map["neck"])
	}
	if got.Map["navel"] != (vec{X: -6, Y: -20}) {
		t.Fatalf("map.navel = %+v", got.Map["navel"])
	}
}
