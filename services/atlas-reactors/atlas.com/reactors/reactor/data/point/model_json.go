package point

import "encoding/json"

type modelJSON struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(modelJSON{
		X: m.x,
		Y: m.y,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var j modelJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.x = j.X
	m.y = j.Y
	return nil
}
