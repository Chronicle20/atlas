package state

import "atlas-reactors/reactor/data/item"

type Model struct {
	theType      int32
	reactorItem  *item.Model
	activeSkills []uint32
	nextState    int8
}

func (m Model) Type() int32 {
	return m.theType
}

func (m Model) ReactorItem() *item.Model {
	return m.reactorItem
}

func (m Model) ActiveSkills() []uint32 {
	return m.activeSkills
}

func (m Model) NextState() int8 {
	return m.nextState
}
