package portal

import (
	"strconv"

	_map "github.com/Chronicle20/atlas-constants/map"
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
		targetMapId: _map.Id(rm.TargetMapId),
		scriptName:  rm.ScriptName,
	}, nil
}
