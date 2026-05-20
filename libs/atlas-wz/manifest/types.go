package manifest

// Manifest is the schema-versioned JSON sidecar produced alongside a sprite
// atlas PNG. PRD §6.2.
type Manifest struct {
	Version   int      `json:"version"`
	ID        uint32   `json:"id"`
	PartClass string   `json:"partClass"`
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
