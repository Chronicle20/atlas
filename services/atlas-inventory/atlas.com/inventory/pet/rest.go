package pet

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id         uint32    `json:"-"`
	CashId     int64     `json:"cashId"`
	TemplateId uint32    `json:"templateId"`
	Name       string    `json:"name"`
	Level      byte      `json:"level"`
	Closeness  uint16    `json:"closeness"`
	Fullness   byte      `json:"fullness"`
	Expiration time.Time `json:"expiration"`
	OwnerId    uint32    `json:"ownerId"`
	Slot       int8      `json:"slot"`
	Flag       uint16    `json:"flag"`
	PurchaseBy uint32    `json:"purchaseBy"`
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
		Id:         m.id,
		CashId:     m.CashId(),
		TemplateId: m.TemplateId(),
		Name:       m.Name(),
		Level:      m.Level(),
		Closeness:  m.Closeness(),
		Fullness:   m.Fullness(),
		Expiration: m.Expiration(),
		OwnerId:    m.OwnerId(),
		Slot:       m.Slot(),
		Flag:       m.Flag(),
		PurchaseBy: m.PurchaseBy(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:         rm.Id,
		cashId:     rm.CashId,
		templateId: rm.TemplateId,
		name:       rm.Name,
		level:      rm.Level,
		closeness:  rm.Closeness,
		fullness:   rm.Fullness,
		expiration: rm.Expiration,
		ownerId:    rm.OwnerId,
		slot:       rm.Slot,
		flag:       rm.Flag,
		purchaseBy: rm.PurchaseBy,
	}, nil
}
