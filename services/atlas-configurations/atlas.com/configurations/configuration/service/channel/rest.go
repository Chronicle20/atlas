package channel

import (
	"atlas-configurations/configuration/server"
	"atlas-configurations/configuration/task"
	"github.com/google/uuid"
)

type RestModel struct {
	Id        uuid.UUID          `json:"-"`
	Tasks     []task.RestModel   `json:"tasks"`
	Channels  []server.RestModel `json:"channels"`
	IpAddress string             `json:"ipAddress"`
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
