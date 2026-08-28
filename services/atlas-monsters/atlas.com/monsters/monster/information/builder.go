package information

// Builder provides a minimal fluent interface for constructing Model
// instances in tests. Only the fields tests need are settable.
type Builder struct {
	skills          []Skill
	attacks         []AttackInfo
	selfDestruction SelfDestruction
	hpRecovery      uint32
	mpRecovery      uint32
	boss            bool
	firstAttack     bool
	resistances     map[string]string
	banish          Banish
}

// NewBuilder returns a new Builder with zero values.
func NewBuilder() *Builder {
	return &Builder{}
}

// SetSkills sets the skill list on the builder.
func (b *Builder) SetSkills(skills []Skill) *Builder {
	b.skills = skills
	return b
}

// SetAttacks sets the attacks list on the builder.
func (b *Builder) SetAttacks(attacks []AttackInfo) *Builder {
	b.attacks = attacks
	return b
}

func (b *Builder) SetHpRecovery(v uint32) *Builder {
	b.hpRecovery = v
	return b
}

func (b *Builder) SetMpRecovery(v uint32) *Builder {
	b.mpRecovery = v
	return b
}

// SetBoss sets the boss flag on the builder. Used by tests that drive
// boss-immunity branches in ApplyStatusEffect and the boss-skip branch
// in DrainMp.
func (b *Builder) SetBoss(boss bool) *Builder {
	b.boss = boss
	return b
}

// SetFirstAttack sets the aggressive-template flag on the builder. Used by
// tests that drive the firstAttack gate in ProcessorImpl.SetAggro.
func (b *Builder) SetFirstAttack(v bool) *Builder {
	b.firstAttack = v
	return b
}

// SetResistances sets the elemental resistance map on the builder. Keys are
// element letters ("P", "I", "F", "S", "L"); value "1" means immune (per
// Model.IsImmuneToElement). Used by tests that drive elemental-immunity
// branches in ApplyStatusEffect.
func (b *Builder) SetResistances(r map[string]string) *Builder {
	b.resistances = r
	return b
}

// SetBanish sets the banish node on the builder. Used by tests that drive the
// banish paths in Banish and executeBanish.
func (b *Builder) SetBanish(banish Banish) *Builder {
	b.banish = banish
	return b
}

// SetSelfDestruction sets the WZ selfDestruction block on the builder. Used by
// tests driving the HP-threshold and timer detonation paths.
func (b *Builder) SetSelfDestruction(sd SelfDestruction) *Builder {
	b.selfDestruction = sd
	return b
}

// Build constructs an immutable Model from the builder state.
func (b *Builder) Build() Model {
	skills := b.skills
	if skills == nil {
		skills = []Skill{}
	}
	attacks := b.attacks
	if attacks == nil {
		attacks = []AttackInfo{}
	}
	return Model{
		skills:          skills,
		attacks:         attacks,
		selfDestruction: b.selfDestruction,
		hpRecovery:      b.hpRecovery,
		mpRecovery:      b.mpRecovery,
		boss:            b.boss,
		firstAttack:     b.firstAttack,
		resistances:     b.resistances,
		banish:          b.banish,
	}
}
