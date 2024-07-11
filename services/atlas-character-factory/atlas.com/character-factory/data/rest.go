package data

type RestModel struct {
	Id   string `json:"-"`
	Name string `json:"name"`
	WZ   string `json:"wz"`
	Slot int16  `json:"slot"`
}

func (r *RestModel) GetName() string {
	return "slots"
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Extract(m RestModel) (Model, error) {
	return Model{
		name: m.Name,
		wz:   m.WZ,
		slot: m.Slot,
	}, nil
}
