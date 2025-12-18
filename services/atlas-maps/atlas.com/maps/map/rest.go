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

func Transform(id uint32) (RestModel, error) {
	return RestModel{Id: strconv.Itoa(int(id))}, nil
}
