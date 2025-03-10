package pet

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id              uint32    `json:"-"`
	InventoryItemId uint32    `json:"inventoryItemId"`
	TemplateId      uint32    `json:"templateId"`
	Name            string    `json:"name"`
	Level           byte      `json:"level"`
	Tameness        uint16    `json:"tameness"`
	Fullness        byte      `json:"fullness"`
	Expiration      time.Time `json:"expiration"`
	OwnerId         uint32    `json:"ownerId"`
	Lead            bool      `json:"lead"`
	Slot            byte      `json:"slot"`
}

func (r RestModel) GetName() string {
	return "pets"
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

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:              rm.Id,
		inventoryItemId: rm.InventoryItemId,
		templateId:      rm.TemplateId,
		name:            rm.Name,
		level:           rm.Level,
		tameness:        rm.Tameness,
		fullness:        rm.Fullness,
		expiration:      rm.Expiration,
		ownerId:         rm.OwnerId,
		lead:            rm.Lead,
		slot:            rm.Slot,
	}, nil
}
