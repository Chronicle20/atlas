package charparts

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// TestExtractSmapHappyPath builds a synthetic Base.wz with a small smap.img
// and confirms the returned map matches the layer-name → slot-codes shape
// the donor's writeSmapFromProps emitted.
func TestExtractSmapHappyPath(t *testing.T) {
	smapImg := wz.NewParsedImage("smap.img", []property.Property{
		property.NewString("cap", "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe"),
		property.NewString("hair", "H2"),
		property.NewString("hairOverHead", "H1"),
	})
	root := wz.NewDirectory("Base", nil, []*wz.Image{smapImg})
	f := wz.NewFileWithRoot("Base", root)

	got, err := ExtractSmap(f)
	if err != nil {
		t.Fatalf("ExtractSmap err: %v", err)
	}
	want := map[string]string{
		"cap":          "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe",
		"hair":         "H2",
		"hairOverHead": "H1",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("smap[%q] = %q, want %q", k, got[k], v)
		}
	}
}

// TestExtractSmapMissing returns ErrSmapMissing when the Base.wz file is
// present but does not contain an smap.img child at its root.
func TestExtractSmapMissing(t *testing.T) {
	root := wz.NewDirectory("Base", nil, []*wz.Image{
		wz.NewParsedImage("zmap.img", nil),
	})
	f := wz.NewFileWithRoot("Base", root)

	_, err := ExtractSmap(f)
	if !errors.Is(err, ErrSmapMissing) {
		t.Fatalf("expected ErrSmapMissing, got %v", err)
	}
}

// TestExtractSmapNilFile guards against caller misuse.
func TestExtractSmapNilFile(t *testing.T) {
	if _, err := ExtractSmap(nil); err == nil {
		t.Fatal("expected error for nil wz.File")
	}
}

// TestExtractSmapIgnoresNonStringChildren mirrors the donor's behaviour of
// dropping non-StringProperty children (defensive: only valid smap.img
// entries are strings, but synthetic fixtures should not panic on stray
// types).
func TestExtractSmapIgnoresNonStringChildren(t *testing.T) {
	smapImg := wz.NewParsedImage("smap.img", []property.Property{
		property.NewString("cap", "Cp"),
		property.NewInt("garbage", 42), // dropped
	})
	root := wz.NewDirectory("Base", nil, []*wz.Image{smapImg})
	f := wz.NewFileWithRoot("Base", root)

	got, err := ExtractSmap(f)
	if err != nil {
		t.Fatalf("ExtractSmap err: %v", err)
	}
	if len(got) != 1 || got["cap"] != "Cp" {
		t.Errorf("unexpected smap: %+v", got)
	}
}

// TestMarshalSmapDeterministic confirms repeated marshals of the same map
// produce identical bytes (Go's json.Marshal sorts map[string]string keys
// since 1.12; this test locks that property in so a stdlib regression here
// would be caught immediately).
func TestMarshalSmapDeterministic(t *testing.T) {
	m := map[string]string{
		"hair":         "H2",
		"cap":          "Cp",
		"hairOverHead": "H1",
		"body":         "Bd",
	}
	a, err := MarshalSmap(m)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		b, err := MarshalSmap(m)
		if err != nil {
			t.Fatal(err)
		}
		if string(a) != string(b) {
			t.Fatalf("non-deterministic marshal\n a=%s\n b=%s", a, b)
		}
	}
	// Sanity-check the shape parses back to the original map.
	var rt map[string]string
	if err := json.Unmarshal(a, &rt); err != nil {
		t.Fatal(err)
	}
	if len(rt) != len(m) {
		t.Fatalf("round-trip length mismatch: got %d, want %d", len(rt), len(m))
	}
}

// TestMarshalSmapNil treats a nil input as an empty map, producing "{}" so
// the downstream PUT never writes "null" bytes.
func TestMarshalSmapNil(t *testing.T) {
	b, err := MarshalSmap(nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "{}" {
		t.Errorf("nil smap marshal = %s, want {}", b)
	}
}
