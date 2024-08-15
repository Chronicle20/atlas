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
