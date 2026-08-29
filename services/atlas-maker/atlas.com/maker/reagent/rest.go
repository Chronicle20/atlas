package reagent

import (
	"strconv"
)

type RestModel struct {
	// Id is the reagent's item id; it is the JSON:API resource id rather than
	// an attribute.
	Id    string `json:"-"`
	Stat  string `json:"stat"`
	Value int16  `json:"value"`
}

func (r RestModel) GetName() string {
	return "reagents"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:    strconv.FormatUint(uint64(m.ReagentItemId()), 10),
		Stat:  m.Stat(),
		Value: m.Value(),
	}, nil
}
