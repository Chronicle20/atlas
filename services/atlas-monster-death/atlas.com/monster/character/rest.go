package character

import "strconv"

type RestModel struct {
	Id    uint32 `json:"-"`
	Level byte   `json:"level"`
}

func (r *RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}

	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:    rm.Id,
		level: rm.Level,
	}, nil
}
