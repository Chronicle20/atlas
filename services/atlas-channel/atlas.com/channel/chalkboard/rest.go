package chalkboard

import "strconv"

type RestModel struct {
	Id      uint32 `json:"-"`
	Message string `json:"message"`
}

func (r RestModel) GetName() string {
	return "chalkboards"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:      rm.Id,
		message: rm.Message,
	}, nil
}
