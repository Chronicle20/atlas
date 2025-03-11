package pet

import (
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

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:              rm.Id,
		inventoryItemId: rm.InventoryItemId,
		templateId:      rm.TemplateId,
		name:            rm.Name,
		level:           rm.Level,
		closeness:       rm.Closeness,
		fullness:        rm.Fullness,
		expiration:      rm.Expiration,
		ownerId:         rm.OwnerId,
		lead:            rm.Lead,
		slot:            rm.Slot,
		x:               rm.X,
		y:               rm.Y,
		stance:          rm.Stance,
		fh:              rm.FH,
	}, nil
}
