package wishlist

import (
	"github.com/google/uuid"
)

type RestModel struct {
	Id           uuid.UUID `json:"-"`
	CharacterId  uint32    `json:"characterId"`
	SerialNumber uint32    `json:"serialNumber"`
}

func (r RestModel) GetName() string {
	return "items"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:           m.id,
		CharacterId:  m.characterId,
		SerialNumber: m.serialNumber,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:           rm.Id,
		characterId:  rm.CharacterId,
		serialNumber: rm.SerialNumber,
	}, nil
}
