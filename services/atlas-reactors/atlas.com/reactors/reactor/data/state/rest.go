package state

import "atlas-reactors/reactor/data/item"

type RestModel struct {
	Type         int32           `json:"type"`
	ReactorItem  *item.RestModel `json:"reactorItem"`
	ActiveSkills []uint32        `json:"activeSkills"`
	NextState    int8            `json:"nextState"`
}

func Extract(rm RestModel) (Model, error) {
	m := Model{
		theType:      rm.Type,
		activeSkills: rm.ActiveSkills,
		nextState:    rm.NextState,
	}
	if rm.ReactorItem != nil {
		rim, err := item.Extract(*rm.ReactorItem)
		if err != nil {
			return m, err
		}
		m.reactorItem = &rim
	}
	return m, nil
}
