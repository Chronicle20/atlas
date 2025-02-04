package login

import (
	"atlas-configurations/configuration/server/login"
	"atlas-configurations/configuration/task"
	"github.com/google/uuid"
)

type RestModel struct {
	Id      uuid.UUID         `json:"-"`
	Tasks   []task.RestModel  `json:"tasks"`
	Servers []login.RestModel `json:"servers"`
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
