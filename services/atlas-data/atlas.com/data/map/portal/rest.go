package portal

import "strconv"

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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          strconv.Itoa(int(m.Id)),
		Name:        m.Name,
		Target:      m.Target,
		Type:        m.PortalType,
		X:           m.X,
		Y:           m.Y,
		TargetMapId: m.TargetMapId,
		ScriptName:  m.ScriptName,
	}, nil
}
