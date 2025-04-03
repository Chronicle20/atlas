package asset

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id          uint32    `json:"-"`
	Slot        int16     `json:"slot"`
	TemplateId  uint32    `json:"templateId"`
	Expiration  time.Time `json:"expiration"`
	ReferenceId uint32    `json:"referenceId"`
}

func (r RestModel) GetName() string {
	return "assets"
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
		Id:          m.id,
		Slot:        m.slot,
		TemplateId:  m.templateId,
		Expiration:  m.expiration,
		ReferenceId: m.referenceId,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		slot:        rm.Slot,
		templateId:  rm.TemplateId,
		expiration:  rm.Expiration,
		referenceId: rm.ReferenceId,
	}, nil
}
