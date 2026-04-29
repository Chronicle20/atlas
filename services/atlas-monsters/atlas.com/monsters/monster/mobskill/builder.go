package mobskill

// ModelBuilder provides a minimal fluent interface for constructing Model
// instances in tests. Only the fields that the picker and stat-buff path read
// are settable.
type ModelBuilder struct {
	skillId  uint16
	level    uint16
	prop     uint32
	mpCon    uint32
	hp       uint32
	interval uint32
	duration uint32
	x        int32
	ltX      int32
	ltY      int32
	rbX      int32
	rbY      int32
}

// NewModelBuilder returns a new ModelBuilder with zero values.
func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

// SetSkillId sets the skill ID.
func (b *ModelBuilder) SetSkillId(skillId uint16) *ModelBuilder {
	b.skillId = skillId
	return b
}

// SetLevel sets the skill level.
func (b *ModelBuilder) SetLevel(level uint16) *ModelBuilder {
	b.level = level
	return b
}

// SetProp sets the activation probability (0-100).
func (b *ModelBuilder) SetProp(prop uint32) *ModelBuilder {
	b.prop = prop
	return b
}

// SetMpCon sets the MP cost required to activate the skill.
func (b *ModelBuilder) SetMpCon(mpCon uint32) *ModelBuilder {
	b.mpCon = mpCon
	return b
}

// SetHp sets the HP threshold (maximum HP% at which the skill is eligible).
func (b *ModelBuilder) SetHp(hp uint32) *ModelBuilder {
	b.hp = hp
	return b
}

// SetInterval sets the cooldown interval in seconds.
func (b *ModelBuilder) SetInterval(interval uint32) *ModelBuilder {
	b.interval = interval
	return b
}

// SetDuration sets the buff/debuff duration in seconds.
func (b *ModelBuilder) SetDuration(duration uint32) *ModelBuilder {
	b.duration = duration
	return b
}

// SetX sets the primary numeric parameter (e.g. reflect percent, heal amount,
// stat magnitude) used by the executor for this skill.
func (b *ModelBuilder) SetX(x int32) *ModelBuilder {
	b.x = x
	return b
}

// SetBoundingBox sets the AoE bounding-box offsets (top-left, bottom-right)
// relative to the casting monster's position.
func (b *ModelBuilder) SetBoundingBox(ltX, ltY, rbX, rbY int32) *ModelBuilder {
	b.ltX = ltX
	b.ltY = ltY
	b.rbX = rbX
	b.rbY = rbY
	return b
}

// Build constructs an immutable Model from the builder state.
func (b *ModelBuilder) Build() Model {
	return Model{
		skillId:  b.skillId,
		level:    b.level,
		prop:     b.prop,
		mpCon:    b.mpCon,
		hp:       b.hp,
		interval: b.interval,
		duration: b.duration,
		x:        b.x,
		ltX:      b.ltX,
		ltY:      b.ltY,
		rbX:      b.rbX,
		rbY:      b.rbY,
	}
}
