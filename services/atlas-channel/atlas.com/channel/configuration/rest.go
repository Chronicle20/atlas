package configuration

import (
	"atlas-channel/configuration/task"
	"errors"

	"github.com/google/uuid"
)

type RestModel struct {
	Id      uuid.UUID                `json:"-"`
	Tasks   []task.RestModel         `json:"tasks"`
	Tenants []ChannelTenantRestModel `json:"tenants"`
}

func (r RestModel) GetName() string {
	return "services"
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

type ChannelTenantRestModel struct {
	Id        string                  `json:"id"`
	IPAddress string                  `json:"ipAddress"`
	Worlds    []ChannelWorldRestModel `json:"worlds"`
}

type ChannelWorldRestModel struct {
	Id       byte                      `json:"id"`
	Channels []ChannelChannelRestModel `json:"channels"`
}

type ChannelChannelRestModel struct {
	Id   byte `json:"id"`
	Port int  `json:"port"`
}

func (r *RestModel) FindTask(name string) (task.RestModel, error) {
	for _, v := range r.Tasks {
		if v.Type == name {
			return v, nil
		}
	}
	return task.RestModel{}, errors.New("task not found")
}
