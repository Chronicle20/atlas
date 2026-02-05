package portal

import _map "github.com/Chronicle20/atlas-constants/map"

type Model struct {
	id          uint32
	name        string
	target      string
	portalType  uint8
	x           int16
	y           int16
	targetMapId _map.Id
	scriptName  string
}

func (m Model) Id() uint32 {
	return m.id
}
