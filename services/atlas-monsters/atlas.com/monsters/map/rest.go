package _map

import "strconv"

type RestModel struct {
	Id string `json:"-"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "characters"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func Extract(rm RestModel) (uint32, error) {
	id, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}
