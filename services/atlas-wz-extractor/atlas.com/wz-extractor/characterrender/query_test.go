package characterrender

import (
	"net/url"
	"reflect"
	"testing"
)

func TestParseRenderQueryDefaults(t *testing.T) {
	got, err := ParseRenderQuery(url.Values{
		"skin": {"0"}, "hair": {"30030"}, "face": {"20000"},
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Stance != "stand1" || got.Frame != 0 || got.Resize != 2 {
		t.Fatalf("defaults wrong: %+v", got)
	}
	if !reflect.DeepEqual(got.Items, []int{}) && got.Items != nil {
		t.Fatalf("items default should be empty: %+v", got.Items)
	}
}

func TestParseRenderQueryItems(t *testing.T) {
	got, err := ParseRenderQuery(url.Values{
		"skin": {"3"}, "hair": {"30030"}, "face": {"20000"},
		"stance": {"stand2"}, "frame": {"1"}, "resize": {"4"},
		"items": {"1442024,1002357,1402024"},
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Stance != "stand2" || got.Frame != 1 || got.Resize != 4 {
		t.Fatalf("scalars: %+v", got)
	}
	want := []int{1442024, 1002357, 1402024}
	if !reflect.DeepEqual(got.Items, want) {
		t.Fatalf("items = %v, want %v", got.Items, want)
	}
}

func TestParseRenderQueryRejectsResizeOutOfRange(t *testing.T) {
	_, err := ParseRenderQuery(url.Values{
		"skin": {"0"}, "hair": {"30030"}, "face": {"20000"}, "resize": {"7"},
	})
	if err == nil {
		t.Fatal("expected error on resize=7")
	}
}
