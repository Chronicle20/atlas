package pet

import (
	"errors"
	"strconv"
	"time"
)

type RestModel struct {
	Id              uint64    `json:"-"`
	InventoryItemId uint32    `json:"inventoryItemId"`
	TemplateId      uint32    `json:"templateId"`
	Name            string    `json:"name"`
	Level           byte      `json:"level"`
	Closeness       uint16    `json:"closeness"`
	Fullness        byte      `json:"fullness"`
	Expiration      time.Time `json:"expiration"`
	OwnerId         uint32    `json:"ownerId"`
	Lead            bool      `json:"lead"`
	Slot            int8      `json:"slot"`
	X               int16     `json:"x"`
	Y               int16     `json:"y"`
	Stance          byte      `json:"stance"`
	FH              int16     `json:"fh"`
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
	r.Id = uint64(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	tm := GetTemporalRegistry().GetById(m.Id())
	if tm == nil {
		return RestModel{}, errors.New("temporal data not found")
	}

	return RestModel{
		Id:              m.id,
		InventoryItemId: m.inventoryItemId,
		TemplateId:      m.templateId,
		Name:            m.name,
		Level:           m.level,
		Closeness:       m.closeness,
		Fullness:        m.fullness,
		Expiration:      m.expiration,
		OwnerId:         m.ownerId,
		Lead:            m.Lead(),
		Slot:            m.slot,
		X:               tm.X(),
		Y:               tm.Y(),
		Stance:          tm.Stance(),
		FH:              tm.FH(),
	}, nil
}
