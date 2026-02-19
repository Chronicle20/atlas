package stat

import "encoding/json"

type Model struct {
	statType string
	amount   int32
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		StatType string `json:"statType"`
		Amount   int32  `json:"amount"`
	}{
		StatType: m.statType,
		Amount:   m.amount,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux struct {
		StatType string `json:"statType"`
		Amount   int32  `json:"amount"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.statType = aux.StatType
	m.amount = aux.Amount
	return nil
}

func (m Model) Type() string {
	return m.statType
}

func (m Model) Amount() int32 {
	return m.amount
}

func NewStat(statType string, amount int32) Model {
	return Model{
		statType: statType,
		amount:   amount,
	}
}
