package cash

import (
	"strconv"
)

type RestModel struct {
	Id              uint32 `json:"-"`
	StateChangeItem uint32 `json:"stateChangeItem"`
	BgmPath         string `json:"bgmPath"`
}

func (r RestModel) GetName() string {
	return "cash_items"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
