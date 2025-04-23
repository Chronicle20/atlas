package pet

import (
	"atlas-pets/pet/exclude"
	"errors"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-model/model"
)

type RestModel struct {
	Id              uint32              `json:"-"`
	InventoryItemId uint32              `json:"inventoryItemId"`
	TemplateId      uint32              `json:"templateId"`
	Name            string              `json:"name"`
	Level           byte                `json:"level"`
	Closeness       uint16              `json:"closeness"`
	Fullness        byte                `json:"fullness"`
	Expiration      time.Time           `json:"expiration"`
	OwnerId         uint32              `json:"ownerId"`
	Lead            bool                `json:"lead"`
	Slot            int8                `json:"slot"`
	X               int16               `json:"x"`
	Y               int16               `json:"y"`
	Stance          byte                `json:"stance"`
	FH              int16               `json:"fh"`
	Excludes        []exclude.RestModel `json:"excludes"`
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
	tm := GetTemporalRegistry().GetById(m.Id())
	if tm == nil {
		return RestModel{}, errors.New("temporal data not found")
	}
	es, err := model.SliceMap(exclude.Transform)(model.FixedProvider(m.Excludes()))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
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
		Excludes:        es,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	es, err := model.SliceMap(exclude.Extract)(model.FixedProvider(rm.Excludes))(model.ParallelMap())()
	if err != nil {
		return Model{}, nil
	}

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
		slot:            rm.Slot,
		excludes:        es,
	}, nil
}
