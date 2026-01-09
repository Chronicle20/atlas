package setup

import "strconv"

type RestModel struct {
	Id      string `json:"-"`
	SlotMax uint32 `json:"slotMax"`
}

func (r RestModel) GetName() string {
	return "setups"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Extract(rm RestModel) (Model, error) {
	id, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:      uint32(id),
		slotMax: rm.SlotMax,
	}, nil
}
