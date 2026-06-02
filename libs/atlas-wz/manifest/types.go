package manifest

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
	// Z is a legacy numeric field. Character part z-order is NOT carried here:
	// v83 Character.wz encodes draw order as a named layer string resolved via
	// Base.wz/zmap.img, not a per-canvas integer, so this is 0 in practice.
	// atlas-renders orders character parts by Part (the layer name) against the
	// zmap.json sidecar — see renders/character/composite.go zIndex. Retained
	// for wire compatibility with existing manifests; do not reintroduce as a
	// sort key.
	Z int `json:"z"`
}

const SchemaVersion = 1
