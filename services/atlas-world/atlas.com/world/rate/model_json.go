package rate

import "encoding/json"

type modelJSON struct {
	ExpRate      float64 `json:"expRate"`
	MesoRate     float64 `json:"mesoRate"`
	ItemDropRate float64 `json:"itemDropRate"`
	QuestExpRate float64 `json:"questExpRate"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(&modelJSON{
		ExpRate:      m.expRate,
		MesoRate:     m.mesoRate,
		ItemDropRate: m.itemDropRate,
		QuestExpRate: m.questExpRate,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux modelJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.expRate = aux.ExpRate
	m.mesoRate = aux.MesoRate
	m.itemDropRate = aux.ItemDropRate
	m.questExpRate = aux.QuestExpRate
	return nil
}
