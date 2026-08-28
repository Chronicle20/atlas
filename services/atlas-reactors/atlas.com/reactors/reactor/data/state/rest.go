package state

import "atlas-reactors/reactor/data/item"

type RestModel struct {
	Type         int32           `json:"type"`
	ReactorItem  *item.RestModel `json:"reactorItem"`
	ActiveSkills []uint32        `json:"activeSkills"`
	NextState    int8            `json:"nextState"`
}

// Transform converts a Model into a RestModel. It is the inverse of Extract.
func Transform(m Model) (RestModel, error) {
	rm := RestModel{
		Type:         m.theType,
		ActiveSkills: m.activeSkills,
		NextState:    m.nextState,
	}
	if m.reactorItem != nil {
		rim, err := item.Transform(*m.reactorItem)
		if err != nil {
			return RestModel{}, err
		}
		rm.ReactorItem = &rim
	}
	return rm, nil
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
