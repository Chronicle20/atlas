package mobskill

// Builder provides a minimal fluent interface for constructing Model
// instances in tests. Only the fields that the picker and stat-buff path read
// are settable.
type Builder struct {
	skillId  uint16
	level    uint16
	prop     uint32
	mpCon    uint32
	hp       uint32
	interval uint32
	duration uint32
	count    uint32
	x        int32
	ltX      int32
	ltY      int32
	rbX      int32
	rbY      int32
}

// NewBuilder returns a new Builder with zero values.
func NewBuilder() *Builder {
	return &Builder{}
}

// SetSkillId sets the skill ID.
func (b *Builder) SetSkillId(skillId uint16) *Builder {
	b.skillId = skillId
	return b
}

// SetLevel sets the skill level.
func (b *Builder) SetLevel(level uint16) *Builder {
	b.level = level
	return b
}

// SetProp sets the activation probability (0-100).
func (b *Builder) SetProp(prop uint32) *Builder {
	b.prop = prop
	return b
}

// SetMpCon sets the MP cost required to activate the skill.
func (b *Builder) SetMpCon(mpCon uint32) *Builder {
	b.mpCon = mpCon
	return b
}

// SetHp sets the HP threshold (maximum HP% at which the skill is eligible).
func (b *Builder) SetHp(hp uint32) *Builder {
	b.hp = hp
	return b
}

// SetInterval sets the cooldown interval in seconds.
func (b *Builder) SetInterval(interval uint32) *Builder {
	b.interval = interval
	return b
}

// SetDuration sets the buff/debuff duration in seconds.
func (b *Builder) SetDuration(duration uint32) *Builder {
	b.duration = duration
	return b
}

// SetCount sets the maximum number of targets the skill affects. Only the
// SEDUCE disease honours this cap (task-259 FR-3.1); it is carried on every
// skill because the WZ data declares it on every skill.
func (b *Builder) SetCount(count uint32) *Builder {
	b.count = count
	return b
}

// SetX sets the primary numeric parameter (e.g. reflect percent, heal amount,
// stat magnitude) used by the executor for this skill.
func (b *Builder) SetX(x int32) *Builder {
	b.x = x
	return b
}

// SetBoundingBox sets the AoE bounding-box offsets (top-left, bottom-right)
// relative to the casting monster's position.
func (b *Builder) SetBoundingBox(ltX, ltY, rbX, rbY int32) *Builder {
	b.ltX = ltX
	b.ltY = ltY
	b.rbX = rbX
	b.rbY = rbY
	return b
}

// Build constructs an immutable Model from the builder state.
func (b *Builder) Build() Model {
	return Model{
		skillId:  b.skillId,
		level:    b.level,
		prop:     b.prop,
		mpCon:    b.mpCon,
		hp:       b.hp,
		interval: b.interval,
		duration: b.duration,
		count:    b.count,
		x:        b.x,
		ltX:      b.ltX,
		ltY:      b.ltY,
		rbX:      b.rbX,
		rbY:      b.rbY,
	}
}
