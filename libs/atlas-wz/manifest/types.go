package manifest

import (
	"bytes"
	"encoding/json"
)

// Manifest is the schema-versioned JSON sidecar produced alongside a sprite
// atlas PNG. PRD §6.2.
//
// Vslot is the equipment template's vslot string (e.g. "Hp" / "CpHnHd"),
// extracted from the .img's info sub-property. Atlas-renders uses it to drive
// equipment-vs-hair occlusion (helmets hiding bangs, etc.). Omitted from the
// JSON encoding when empty so the determinism of pre-existing manifests is
// preserved.
type Manifest struct {
	Version   int      `json:"version"`
	ID        uint32   `json:"id"`
	PartClass string   `json:"partClass"`
	Vslot     string   `json:"vslot,omitempty"`
	Sheet     Size     `json:"sheet"`
	Sprites   []Sprite `json:"sprites"`
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Sprite struct {
	Stance  string           `json:"stance"`
	Frame   int              `json:"frame"`
	Part    string           `json:"part"`
	Rect    Rect             `json:"rect"`
	Origin  Point            `json:"origin"`
	Anchors map[string]Point `json:"anchors"`
	// Z is the WZ render-layer label for this sprite — the value of the part
	// canvas's `z` child (e.g. "weaponOverGlove", "armBelowHead"). It is the
	// key atlas-renders resolves against Base.wz/zmap.img (zmap.json) for draw
	// order and against smap.img (smap.json) for vslot occlusion.
	//
	// This is NOT the same as Part: Part is the canvas NAME (often generic,
	// e.g. "weapon"), while Z is the specific render layer the sprite occupies
	// (which varies by stance/frame). Sorting by Part instead of Z mislayers
	// any part whose canvas name differs from its z-label.
	Z ZOrder `json:"z"`
}

// ZOrder is a WZ render-layer label (a zmap.img key). It is encoded as a JSON
// string. UnmarshalJSON tolerates the legacy v1 schema, which stored a numeric
// `z` (always 0, the dropped donor field): a JSON number decodes to "" rather
// than failing, so atlas-renders can read not-yet-reingested manifests without
// erroring (they simply fall back to insertion order until re-ingest).
type ZOrder string

func (z *ZOrder) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*z = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*z = ZOrder(s)
		return nil
	}
	// Legacy numeric z (schema v1) carries no layer label.
	*z = ""
	return nil
}

// SchemaVersion 2 carries the string render-layer label in Sprite.Z (was a
// dropped numeric in v1). atlas-renders reads both via ZOrder.UnmarshalJSON.
const SchemaVersion = 2
