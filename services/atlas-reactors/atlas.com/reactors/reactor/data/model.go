package data

import (
	"atlas-reactors/reactor/data/point"
	"atlas-reactors/reactor/data/state"
)

type Model struct {
	name                 string
	tl                   point.Model
	br                   point.Model
	stateInfo            map[int8][]state.Model
	timeoutInfo          map[int8]int32
	timeoutNextStateInfo map[int8]int8
}

func (m Model) Name() string {
	return m.name
}

func (m Model) StateInfo() map[int8][]state.Model {
	return m.stateInfo
}

func (m Model) TL() point.Model {
	return m.tl
}

func (m Model) BR() point.Model {
	return m.br
}

// Timeout returns the per-state timeout in milliseconds, or -1 if this state
// has no timeout configured.
func (m Model) Timeout(state int8) int32 {
	if m.timeoutInfo == nil {
		return -1
	}
	v, ok := m.timeoutInfo[state]
	if !ok {
		return -1
	}
	return v
}

// TimeoutNextState returns the state to transition to when this state's timer
// fires. The bool is false if no timer transition is configured for this state
// (i.e. no type-101 event was present in the .wz).
func (m Model) TimeoutNextState(state int8) (int8, bool) {
	if m.timeoutNextStateInfo == nil {
		return 0, false
	}
	v, ok := m.timeoutNextStateInfo[state]
	return v, ok
}
