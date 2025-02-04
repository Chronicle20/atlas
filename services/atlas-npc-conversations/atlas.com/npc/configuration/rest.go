package configuration

import (
	"atlas-npc-conversations/configuration/npcconversation"
	"github.com/google/uuid"
)

type RestModel struct {
	Id      uuid.UUID                   `json:"-"`
	Servers []npcconversation.RestModel `json:"servers"`
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
