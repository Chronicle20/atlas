package configuration

import (
	"atlas-channel/configuration/server/channel"
	"atlas-channel/configuration/task"
	"errors"
	"github.com/google/uuid"
)

func (r *RestModel) FindTask(name string) (task.RestModel, error) {
	for _, v := range r.Tasks {
		if v.Type == name {
			return v, nil
		}
	}
	return task.RestModel{}, errors.New("task not found")
}

func (r *RestModel) FindServer(tenantId uuid.UUID) (channel.RestModel, error) {
	for _, v := range r.Channels {
		if v.TenantId == tenantId {
			return v, nil
		}
	}
	return channel.RestModel{}, errors.New("server not found")
}
