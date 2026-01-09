package pet

import "strconv"

type RestModel struct {
	Id          string `json:"-"`
	OwnerId     uint32 `json:"ownerId"`
	CashId      int64  `json:"cashId,string"`
	Flag        uint16 `json:"flag"`
	PurchasedBy uint32 `json:"purchasedBy"`
	Name        string `json:"name"`
	Level       byte   `json:"level"`
	Closeness   uint16 `json:"closeness"`
	Fullness    byte   `json:"fullness"`
	Slot        int8   `json:"slot"`
}

func (r RestModel) GetName() string {
	return "pets"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Extract(rm RestModel) (Model, error) {
	id, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:          uint32(id),
		ownerId:     rm.OwnerId,
		cashId:      rm.CashId,
		flag:        rm.Flag,
		purchasedBy: rm.PurchasedBy,
		name:        rm.Name,
		level:       rm.Level,
		closeness:   rm.Closeness,
		fullness:    rm.Fullness,
		slot:        rm.Slot,
	}, nil
}
