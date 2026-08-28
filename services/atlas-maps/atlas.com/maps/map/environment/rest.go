package environment

import "fmt"

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

func Transform(e ObjectEntry) (RestModel, error) {
	return RestModel{
		Id:    fmt.Sprintf("%s:%s", e.Kind, e.Name),
		Kind:  string(e.Kind),
		Name:  e.Name,
		State: e.State,
	}, nil
}
