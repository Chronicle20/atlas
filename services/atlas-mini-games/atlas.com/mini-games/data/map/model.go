package mapdata

import _map "github.com/Chronicle20/atlas/libs/atlas-constants/map"

// Model is the minimal map view the mini-game validation ladder needs: the
// fieldLimit bitmask (bit 0x80 forbids starting a mini-game on the map).
type Model struct {
	id         _map.Id
	fieldLimit uint32
}

func (m Model) Id() _map.Id {
	return m.id
}

func (m Model) FieldLimit() uint32 {
	return m.fieldLimit
}
