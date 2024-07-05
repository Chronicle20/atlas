package slots

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

func TransformAll(m Model) ([]RestModel, error) {
	var results = make([]RestModel, 0)
	for _, s := range m.slot {
		rm := RestModel{
			Id:   m.itemId,
			Name: m.name,
			WZ:   m.wz,
			Slot: s,
		}
		results = append(results, rm)
	}
	return results, nil
}
