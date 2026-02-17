package data

import (
	"atlas-reactors/reactor/data/point"
	"atlas-reactors/reactor/data/state"
)

type Model struct {
	name        string
	tl          point.Model
	br          point.Model
	stateInfo   map[int8][]state.Model
	timeoutInfo map[int8]int32
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
