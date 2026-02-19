package state

import (
	"atlas-reactors/reactor/data/item"
	"encoding/json"
)

type modelJSON struct {
	TheType      int32        `json:"type"`
	ReactorItem  *item.Model  `json:"reactorItem,omitempty"`
	ActiveSkills []uint32     `json:"activeSkills"`
	NextState    int8         `json:"nextState"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(modelJSON{
		TheType:      m.theType,
		ReactorItem:  m.reactorItem,
		ActiveSkills: m.activeSkills,
		NextState:    m.nextState,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var j modelJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	m.theType = j.TheType
	m.reactorItem = j.ReactorItem
	m.activeSkills = j.ActiveSkills
	m.nextState = j.NextState
	return nil
}
