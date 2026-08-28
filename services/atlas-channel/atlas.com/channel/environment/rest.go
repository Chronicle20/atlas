package environment

type RestModel struct {
	Id    string `json:"-"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	State uint32 `json:"state"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "environment-objects"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}
