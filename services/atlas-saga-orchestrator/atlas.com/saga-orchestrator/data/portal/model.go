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

func (p Model) Id() uint32 {
	return p.id
}

func (p Model) Name() string {
	return p.name
}

func (p Model) TargetMapId() _map.Id {
	return p.targetMapId
}

func (p Model) Type() uint8 {
	return p.portalType
}

func SpawnPoint(m Model) bool {
	return m.Type() == 0
}

func NoTarget(m Model) bool {
	return m.TargetMapId() == 999999999
}

func ByName(name string) func(Model) bool {
	return func(m Model) bool {
		return m.Name() == name
	}
}
