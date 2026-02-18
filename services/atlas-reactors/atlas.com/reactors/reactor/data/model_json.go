package data

import (
	"atlas-reactors/reactor/data/point"
	"atlas-reactors/reactor/data/state"
	"encoding/json"
)

type modelJSON struct {
	Name        string                 `json:"name"`
	Tl          point.Model            `json:"tl"`
	Br          point.Model            `json:"br"`
	StateInfo   map[int8][]state.Model `json:"stateInfo"`
	TimeoutInfo map[int8]int32         `json:"timeoutInfo"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(modelJSON{
		Name:        m.name,
		Tl:          m.tl,
		Br:          m.br,
		StateInfo:   m.stateInfo,
		TimeoutInfo: m.timeoutInfo,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var j modelJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.name = j.Name
	m.tl = j.Tl
	m.br = j.Br
	m.stateInfo = j.StateInfo
	m.timeoutInfo = j.TimeoutInfo
	return nil
}
