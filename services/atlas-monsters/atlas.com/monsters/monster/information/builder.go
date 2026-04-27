package information

// ModelBuilder provides a minimal fluent interface for constructing Model
// instances in tests. Only the fields that the picker reads are settable.
type ModelBuilder struct {
	skills     []Skill
	hpRecovery uint32
	mpRecovery uint32
}

// NewModelBuilder returns a new ModelBuilder with zero values.
func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

// SetSkills sets the skill list on the builder.
func (b *ModelBuilder) SetSkills(skills []Skill) *ModelBuilder {
	b.skills = skills
	return b
}

func (b *ModelBuilder) SetHpRecovery(v uint32) *ModelBuilder {
	b.hpRecovery = v
	return b
}

func (b *ModelBuilder) SetMpRecovery(v uint32) *ModelBuilder {
	b.mpRecovery = v
	return b
}

// Build constructs an immutable Model from the builder state.
func (b *ModelBuilder) Build() Model {
	skills := b.skills
	if skills == nil {
		skills = []Skill{}
	}
	return Model{
		skills:     skills,
		hpRecovery: b.hpRecovery,
		mpRecovery: b.mpRecovery,
	}
}
