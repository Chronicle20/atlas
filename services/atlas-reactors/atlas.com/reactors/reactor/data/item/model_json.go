package item

import "encoding/json"

type modelJSON struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint16 `json:"quantity"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(modelJSON{
		ItemId:   m.itemId,
		Quantity: m.quantity,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var j modelJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.itemId = j.ItemId
	m.quantity = j.Quantity
	return nil
}
