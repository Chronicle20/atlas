package party_quest

import "github.com/google/uuid"

type RestModel struct {
	Id uuid.UUID `json:"-"`
}

func (r RestModel) GetName() string {
	return "instances"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id: rm.Id,
	}, nil
}
