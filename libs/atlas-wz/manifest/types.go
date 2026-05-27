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
	Z       int              `json:"z"`
}

const SchemaVersion = 1
