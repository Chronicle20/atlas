package configuration

import (
	"atlas-login/configuration/login"
	"atlas-login/configuration/task"
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

func (r *RestModel) FindServer(tenantId uuid.UUID) (login.RestModel, error) {
	for _, v := range r.Servers {
		if v.TenantId == tenantId {
			return v, nil
		}
	}
	return login.RestModel{}, errors.New("server not found")
}
