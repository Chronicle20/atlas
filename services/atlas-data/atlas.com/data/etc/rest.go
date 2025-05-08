package etc

import (
	"strconv"
)

type RestModel struct {
	Id        uint32 `json:"-"`
	Price     uint32 `json:"price"`
	UnitPrice uint32 `json:"unitPrice"`
	SlotMax   uint32 `json:"slotMax"`
}

func (r RestModel) GetName() string {
	return "etcs"
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