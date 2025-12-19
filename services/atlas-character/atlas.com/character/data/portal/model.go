package portal

import "fmt"

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

func (m Model) HasScript() bool {
	return m.scriptName != ""
}

func (m Model) String() string {
	return fmt.Sprintf("%d - %s", m.id, m.name)
}

func (m Model) ScriptName() string {
	return m.scriptName
}

func (m Model) HasTargetMap() bool {
	return m.targetMapId != 999999999
}

func (m Model) TargetMapId() uint32 {
	return m.targetMapId
}

func (m Model) Target() string {
	return m.target
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}
