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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:              m.id,
		InventoryItemId: m.inventoryItemId,
		TemplateId:      m.templateId,
		Name:            m.name,
		Level:           m.level,
		Tameness:        m.tameness,
		Fullness:        m.fullness,
		Expiration:      m.expiration,
		OwnerId:         m.ownerId,
		Lead:            m.lead,
		Slot:            m.slot,
	}, nil
}
