package mobskill

type Model struct {
	skillId      uint16
	level        uint16
	mpCon        uint32
	duration     uint32
	hp           uint32
	x            int32
	y            int32
	prop         uint32
	interval     uint32
	count        uint32
	limit        uint32
	ltX          int32
	ltY          int32
	rbX          int32
	rbY          int32
	summonEffect uint32
	summons      []uint32
}

func (m Model) SkillId() uint16 {
	return m.skillId
}

func (m Model) Level() uint16 {
	return m.level
}

func (m Model) MpCon() uint32 {
	return m.mpCon
}

func (m Model) Duration() uint32 {
	return m.duration
}

func (m Model) Hp() uint32 {
	return m.hp
}

func (m Model) X() int32 {
	return m.x
}

func (m Model) Y() int32 {
	return m.y
}

func (m Model) Prop() uint32 {
	return m.prop
}

func (m Model) Interval() uint32 {
	return m.interval
}

func (m Model) Count() uint32 {
	return m.count
}

func (m Model) Limit() uint32 {
	return m.limit
}

func (m Model) HasBoundingBox() bool {
	return m.ltX != 0 || m.ltY != 0 || m.rbX != 0 || m.rbY != 0
}

func (m Model) LtX() int32 {
	return m.ltX
}

func (m Model) LtY() int32 {
	return m.ltY
}

func (m Model) RbX() int32 {
	return m.rbX
}

func (m Model) RbY() int32 {
	return m.rbY
}

func (m Model) SummonEffect() uint32 {
	return m.summonEffect
}

func (m Model) Summons() []uint32 {
	return m.summons
}
