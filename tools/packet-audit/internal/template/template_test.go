package template

import "testing"

func TestLoadResolveHandler(t *testing.T) {
	tpl, err := Load("testdata/template_gms_95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Region != "GMS" || tpl.MajorVersion != 95 {
		t.Fatalf("region/major: got %s/%d", tpl.Region, tpl.MajorVersion)
	}
	if h, ok := tpl.Handler(0x01); !ok || h != "LoginHandle" {
		t.Errorf("handler 0x01: ok=%v name=%q", ok, h)
	}
	if w, ok := tpl.Writer(0x00); !ok || w != "AuthSuccess" {
		t.Errorf("writer 0x00: ok=%v name=%q", ok, w)
	}
}
