package wishlist

import (
	"strconv"
)

type RestModel struct {
	Id           uint32 `json:"-"`
	SerialNumber uint32 `json:"serialNumber"`
}

func (r RestModel) GetName() string {
	return "wishlist_items"
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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:           m.id,
		SerialNumber: m.serialNumber,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:           rm.Id,
		serialNumber: rm.SerialNumber,
	}, nil
}
