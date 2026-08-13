package cash

import (
	"strconv"
)

type RestModel struct {
	Id              uint32 `json:"-"`
	StateChangeItem uint32 `json:"stateChangeItem"`
	BgmPath         string `json:"bgmPath"`
	ProtectTime     uint32 `json:"protectTime"`
	// Meso is the 0520 meso-sack award amount (atlas-data info/meso). Absent
	// or 0 means "no payout" and the type-19 handler rejects the use.
	Meso uint32 `json:"meso"`
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
