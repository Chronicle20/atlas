package configuration

import (
	"atlas-drops-information/configuration/server/drops/information"
	"github.com/google/uuid"
)

type RestModel struct {
	Id      uuid.UUID               `json:"-"`
	Servers []information.RestModel `json:"servers"`
}

func (r RestModel) GetName() string {
	return "configurations"
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
