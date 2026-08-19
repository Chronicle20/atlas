package cashpackage

import (
	"strconv"
)

type RestModel struct {
	Id            uint32   `json:"-"`
	SerialNumbers []uint32 `json:"serialNumbers"`
}

func (r RestModel) GetName() string {
	return "cashPackages"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
