package portal

import (
	"strconv"
)

type RestModel struct {
	Id          string `json:"-"`
	Name        string `json:"name"`
	Target      string `json:"target"`
	Type        uint8  `json:"type"`
	X           int16  `json:"x"`
	Y           int16  `json:"y"`
	TargetMapId uint32 `json:"targetMapId"`
	ScriptName  string `json:"scriptName"`
}

func (r RestModel) GetName() string {
	return "portals"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

type Model struct {
	id          uint32
	name        string
	portalType  uint8
	x           int16
	y           int16
	targetMapId uint32
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) Name() string {
	return m.name
}

// Type is the WZ portal type. 0 = spawn point ("sp"), 1 = teleport portal
// (in-map up/dn), 2 = map portal (exit). Store placement only cares about
// teleport portals (see IsNearPortal).
func (m Model) Type() uint8 {
	return m.portalType
}

// TargetMapId is the destination map, or 999999999 (MapId.NONE) when the portal
// has no target (spawn points and dead-end script portals).
func (m Model) TargetMapId() uint32 {
	return m.targetMapId
}

func Extract(rm RestModel) (Model, error) {
	id, err := strconv.Atoi(rm.Id)
	if err != nil {
		return Model{}, err
	}
	return Model{
		id:          uint32(id),
		name:        rm.Name,
		portalType:  rm.Type,
		x:           rm.X,
		y:           rm.Y,
		targetMapId: rm.TargetMapId,
	}, nil
}
