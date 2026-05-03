package information

// ModelBuilder provides a minimal fluent interface for constructing Model
// instances in tests. Only the fields tests need are settable.
type ModelBuilder struct {
	skills      []Skill
	attacks     []AttackInfo
	hpRecovery  uint32
	mpRecovery  uint32
	boss        bool
	resistances map[string]string
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

// SetAttacks sets the attacks list on the builder.
func (b *ModelBuilder) SetAttacks(attacks []AttackInfo) *ModelBuilder {
	b.attacks = attacks
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

// SetBoss sets the boss flag on the builder. Used by tests that drive
// boss-immunity branches in ApplyStatusEffect.
func (b *ModelBuilder) SetBoss(boss bool) *ModelBuilder {
	b.boss = boss
	return b
}

// SetResistances sets the elemental resistance map on the builder. Keys are
// element letters ("P", "I", "F", "S", "L"); value "1" means immune (per
// Model.IsImmuneToElement). Used by tests that drive elemental-immunity
// branches in ApplyStatusEffect.
func (b *ModelBuilder) SetResistances(r map[string]string) *ModelBuilder {
	b.resistances = r
	return b
}

// Build constructs an immutable Model from the builder state.
func (b *ModelBuilder) Build() Model {
	skills := b.skills
	if skills == nil {
		skills = []Skill{}
	}
	attacks := b.attacks
	if attacks == nil {
		attacks = []AttackInfo{}
	}
	return Model{
		skills:      skills,
		attacks:     attacks,
		hpRecovery:  b.hpRecovery,
		mpRecovery:  b.mpRecovery,
		boss:        b.boss,
		resistances: b.resistances,
	}
}
