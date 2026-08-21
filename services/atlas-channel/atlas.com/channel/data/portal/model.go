package portal

import _map "github.com/Chronicle20/atlas/libs/atlas-constants/map"

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

func (m Model) Name() string {
	return m.name
}

func (m Model) Target() string {
	return m.target
}

func (m Model) Type() uint8 {
	return m.portalType
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) TargetMapId() _map.Id {
	return m.targetMapId
}

func (m Model) ScriptName() string {
	return m.scriptName
}
