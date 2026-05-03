package information

type AttackInfo struct {
	Pos         uint8
	ConMP       int32
	AttackAfter int32
}

type Model struct {
	monsterId uint32
	attacks   []AttackInfo
}

func (m Model) MonsterId() uint32 {
	return m.monsterId
}

func (m Model) Attacks() []AttackInfo {
	return m.attacks
}
