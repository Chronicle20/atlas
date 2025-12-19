package _map

import "strconv"

type RestModel struct {
	Id string `json:"-"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(m RestModel) (uint32, error) {
	id, err := strconv.ParseUint(m.Id, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}
