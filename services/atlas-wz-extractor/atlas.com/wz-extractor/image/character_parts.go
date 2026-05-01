package image

// partSidecar is the JSON sidecar emitted next to each part PNG.
type partSidecar struct {
	Origin vec            `json:"origin"`
	Map    map[string]vec `json:"map,omitempty"`
	Z      string         `json:"z,omitempty"`
	Group  string         `json:"group,omitempty"`
	Delay  int            `json:"delay,omitempty"`
	Face   int            `json:"face,omitempty"`
}

type vec struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// templateInfo is the per-img info.json block.
type templateInfo struct {
	Islot string `json:"islot,omitempty"`
	Vslot string `json:"vslot,omitempty"`
	Cash  int    `json:"cash"`
}

// stancesInScope is the explicit allow-list of stances we extract. Skipping
// fly/prone/swing/etc. keeps the on-disk footprint manageable.
var stancesInScope = map[string]struct{}{
	"stand1": {},
	"stand2": {},
	"walk1":  {},
	"alert":  {},
	"jump":   {},
}

// equipmentSubdirs are the Character.wz subdirectories whose .img files we
// extract worn sprites for. Body skin imgs live at the root, not in a subdir.
var equipmentSubdirs = []string{
	"Cap", "Coat", "Longcoat", "Pants", "Shoes", "Glove",
	"Cape", "Shield", "Weapon", "Hair", "Face", "Accessory",
}
