package monster

type Model struct {
	uniqueId           uint32
	hp                 uint32
	mp                 uint32
	monsterId          uint32
	controlCharacterId uint32
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	team               int8
}

func (m Model) UniqueId() uint32 {
	return m.uniqueId
}

func (m Model) Controlled() bool {
	return m.controlCharacterId != 0
}

func (m Model) MonsterId() uint32 {
	return m.monsterId
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) Stance() byte {
	return m.stance
}

func (m Model) Fh() int16 {
	return m.fh
}

func (m Model) Team() int8 {
	return m.team
}
