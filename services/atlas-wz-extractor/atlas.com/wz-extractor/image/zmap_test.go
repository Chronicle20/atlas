package image

import (
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteZmapFromProps(t *testing.T) {
	dir := t.TempDir()
	props := []property.Property{
		property.NewNull("body"),
		property.NewNull("arm"),
		property.NewNull("hairOverHead"),
		property.NewNull("weapon"),
	}
	if err := writeZmapFromProps(props, dir); err != nil {
		t.Fatalf("writeZmapFromProps: %v", err)
	}
	var got []string
	readJSON(t, filepath.Join(dir, "zmap.json"), &got)
	want := []string{"body", "arm", "hairOverHead", "weapon"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("zmap = %v, want %v", got, want)
	}
}

func TestWriteSmapFromProps(t *testing.T) {
	dir := t.TempDir()
	props := []property.Property{
		property.NewString("capOverHair", "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe"),
		property.NewString("weaponOverArm", "WpAr"),
		property.NewNull("ignored-non-string"),
	}
	if err := writeSmapFromProps(props, dir); err != nil {
		t.Fatalf("writeSmapFromProps: %v", err)
	}
	var got map[string]string
	readJSON(t, filepath.Join(dir, "smap.json"), &got)
	want := map[string]string{
		"capOverHair":   "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe",
		"weaponOverArm": "WpAr",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("smap = %v, want %v", got, want)
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}
