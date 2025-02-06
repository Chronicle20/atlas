package portal

type Model struct {
	id          uint32
	name        string
	target      string
	portalType  uint8
	x           int16
	y           int16
	targetMapId uint32
	scriptName  string
}

func (p Model) Id() uint32 {
	return p.id
}

func (p Model) TargetMapId() uint32 {
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
