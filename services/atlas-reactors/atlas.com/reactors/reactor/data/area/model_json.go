package area

import (
	"atlas-reactors/reactor/data/point"
	"encoding/json"
)

type modelJSON struct {
	TL point.Model `json:"tl"`
	BR point.Model `json:"br"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(modelJSON{
		TL: m.tl,
		BR: m.br,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var j modelJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.tl = j.TL
	m.br = j.BR
	return nil
}
