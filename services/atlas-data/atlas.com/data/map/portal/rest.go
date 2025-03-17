package portal

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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          m.Id,
		Name:        m.Name,
		Target:      m.Target,
		Type:        m.Type,
		X:           m.X,
		Y:           m.Y,
		TargetMapId: m.TargetMapId,
		ScriptName:  m.ScriptName,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		Id:          rm.Id,
		Name:        rm.Name,
		Target:      rm.Target,
		Type:        rm.Type,
		X:           rm.X,
		Y:           rm.Y,
		TargetMapId: rm.TargetMapId,
		ScriptName:  rm.ScriptName,
	}, nil
}
