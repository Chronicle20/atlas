package portal

type Model struct {
	Id          uint32
	Name        string
	Target      string
	PortalType  uint8
	X           int16
	Y           int16
	TargetMapId uint32
	ScriptName  string
}
