package configuration

import (
	"atlas-drops/configuration/task"
	"errors"
	"github.com/google/uuid"
)

type RestModel struct {
	Id    uuid.UUID        `json:"-"`
	Tasks []task.RestModel `json:"tasks"`
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

func (r *RestModel) FindTask(name string) (task.RestModel, error) {
	for _, v := range r.Tasks {
		if v.Type == name {
			return v, nil
		}
	}
	return task.RestModel{}, errors.New("task not found")
}
