package template

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadRealGMS95(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	p := filepath.Join(repoRoot, "services", "atlas-configurations", "seed-data", "templates", "template_gms_95_1.json")
	tpl, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if tpl.Region != "GMS" || tpl.MajorVersion != 95 {
		t.Fatalf("region/major: got %s/%d", tpl.Region, tpl.MajorVersion)
	}
	if tpl.ClientVariant != "modified" {
		t.Errorf("default variant: got %q, want modified", tpl.ClientVariant)
	}
}
