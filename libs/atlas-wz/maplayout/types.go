package maplayout

// Layout describes a parsed Map.img layout — the inputs to the lazy map
// renderer. Pure types; no WZ dependency.
type Layout struct {
	Version   int        `json:"version"`
	MapID     uint32     `json:"mapId"`
	Bounds    Bounds     `json:"bounds"`
	Layers    []Layer    `json:"layers"`
	Footholds []Foothold `json:"footholds"`
	Portals   []Portal   `json:"portals"`
	NPCs      []NPC      `json:"npcs"`
	ZMap      []string   `json:"zmap"`
}

type Bounds struct {
	Left, Top, Right, Bottom int
}

type Layer struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Z      int    `json:"z"`
	Source string `json:"source"` // bucket key suffix to the PNG
}

type Foothold struct {
	ID         int `json:"id"`
	X1, Y1     int
	X2, Y2     int
	Prev, Next int
}

type Portal struct {
	Name   string `json:"name"`
	Type   int    `json:"type"`
	Target uint32 `json:"target"`
	X, Y   int
}

type NPC struct {
	ID       uint32 `json:"id"`
	X, Y     int
	Foothold int `json:"foothold"`
}

const SchemaVersion = 1
