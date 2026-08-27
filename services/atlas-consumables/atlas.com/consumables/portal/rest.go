package portal

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type RestModel struct {
	Id          string  `json:"-"`
	Name        string  `json:"name"`
	Target      string  `json:"target"`
	Type        uint8   `json:"type"`
	X           int16   `json:"x"`
	Y           int16   `json:"y"`
	TargetMapId _map.Id `json:"targetMapId"`
	ScriptName  string  `json:"scriptName"`
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

func Extract(rm RestModel) (Model, error) {
	id, err := strconv.Atoi(rm.Id)
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:          uint32(id),
		name:        rm.Name,
		target:      rm.Target,
		portalType:  rm.Type,
		x:           rm.X,
		y:           rm.Y,
		targetMapId: rm.TargetMapId,
		scriptName:  rm.ScriptName,
	}, nil
}

// Transform is the inverse of Extract: it converts the domain Model back into
// the RestModel representation, rendering Id as its string form.
func Transform(m Model) RestModel {
	return RestModel{
		Id:          strconv.Itoa(int(m.id)),
		Name:        m.name,
		Target:      m.target,
		Type:        m.portalType,
		X:           m.x,
		Y:           m.y,
		TargetMapId: m.targetMapId,
		ScriptName:  m.scriptName,
	}
}
