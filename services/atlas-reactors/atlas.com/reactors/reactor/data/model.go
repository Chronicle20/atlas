package data

import (
	"atlas-reactors/reactor/data/point"
	"atlas-reactors/reactor/data/state"
)

type Model struct {
	tl          point.Model
	br          point.Model
	stateInfo   map[int8][]state.Model
	timeoutInfo map[int8]int32
}
