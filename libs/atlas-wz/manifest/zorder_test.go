package manifest

import (
	"encoding/json"
	"testing"
)

// TestZOrderUnmarshalJSON locks in the legacy-tolerance contract: schema v1
// manifests stored a numeric `z` (always 0, the dropped donor field), while v2
// stores a string render-layer label. ZOrder.UnmarshalJSON must accept both —
// a number decodes to "" rather than erroring — so atlas-renders can read
// not-yet-reingested manifests without 500ing the render path.
func TestZOrderUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want ZOrder
	}{
		{"string label", `"weaponOverGlove"`, "weaponOverGlove"},
		{"empty string", `""`, ""},
		{"legacy numeric zero", `0`, ""},
		{"legacy numeric nonzero", `3`, ""},
		{"negative", `-1`, ""},
		{"float", `1.5`, ""},
		{"null", `null`, ""},
		{"whitespace-padded string", ` "head" `, "head"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var z ZOrder
			if err := json.Unmarshal([]byte(c.in), &z); err != nil {
				t.Fatalf("Unmarshal(%s) error: %v", c.in, err)
			}
			if z != c.want {
				t.Errorf("Unmarshal(%s) = %q, want %q", c.in, z, c.want)
			}
		})
	}
}

// TestZOrderRoundTrip confirms a string label marshals back to a JSON string
// (ZOrder is a bare string type, so the default marshal applies).
func TestZOrderRoundTrip(t *testing.T) {
	z := ZOrder("weaponBelowArm")
	b, err := json.Marshal(z)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"weaponBelowArm"` {
		t.Fatalf("marshal = %s, want \"weaponBelowArm\"", b)
	}
}

// TestUnmarshalLegacyV1Manifest decodes a full v1-shaped manifest (numeric z)
// through the public Unmarshal without error — the un-reingested read path that
// atlas-renders hits between deploy and re-ingest. The legacy numeric z must
// land as an empty ZOrder (→ insertion-order fallback), never an error.
func TestUnmarshalLegacyV1Manifest(t *testing.T) {
	v1 := `{"version":1,"id":1302000,"partClass":"Weapon","sheet":{"width":256,"height":256},` +
		`"sprites":[{"stance":"stand1","frame":0,"part":"weapon","rect":{"x":0,"y":0,"w":41,"h":13},` +
		`"origin":{"x":25,"y":7},"anchors":{"hand":{"x":6,"y":-5}},"z":0}]}`
	m, err := Unmarshal([]byte(v1))
	if err != nil {
		t.Fatalf("Unmarshal v1 manifest: %v", err)
	}
	if len(m.Sprites) != 1 {
		t.Fatalf("got %d sprites, want 1", len(m.Sprites))
	}
	if m.Sprites[0].Z != "" {
		t.Errorf("legacy numeric z should decode to empty ZOrder, got %q", m.Sprites[0].Z)
	}
	if m.Sprites[0].Part != "weapon" {
		t.Errorf("part = %q, want weapon", m.Sprites[0].Part)
	}
}

// TestUnmarshalV2Manifest decodes a v2 manifest carrying a string z-label.
func TestUnmarshalV2Manifest(t *testing.T) {
	v2 := `{"version":2,"id":1302000,"partClass":"Weapon","sheet":{"width":256,"height":256},` +
		`"sprites":[{"stance":"alert","frame":0,"part":"weapon","rect":{"x":0,"y":0,"w":41,"h":14},` +
		`"origin":{"x":23,"y":10},"anchors":{"hand":{"x":11,"y":1}},"z":"weaponOverHand"}]}`
	m, err := Unmarshal([]byte(v2))
	if err != nil {
		t.Fatalf("Unmarshal v2 manifest: %v", err)
	}
	if m.Sprites[0].Z != "weaponOverHand" {
		t.Errorf("z = %q, want weaponOverHand", m.Sprites[0].Z)
	}
}
