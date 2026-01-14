package stat

import "strconv"

type RestModel struct {
	Id     string `json:"-"`
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func (r RestModel) GetName() string {
	return "stats"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:     strconv.Itoa(int(m.Amount())),
		Type:   m.Type(),
		Amount: m.Amount(),
	}, nil
}
