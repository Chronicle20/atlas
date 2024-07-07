package information

import "strconv"

type RestModel struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
	WZ   string `json:"WZ"`
	Slot int16  `json:"slot"`
}

func (r RestModel) GetName() string {
	return "slots"
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

func Extract(m RestModel) (Model, error) {
	return Model{
		itemId: m.Id,
		name:   m.Name,
		wz:     m.WZ,
		slot:   m.Slot,
	}, nil
}
