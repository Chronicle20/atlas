package skill

import "time"

// Model represents a character skill
type Model struct {
	id                uint32
	level             byte
	masterLevel       byte
	expiration        time.Time
	cooldownExpiresAt time.Time
}

// NewModel creates a new skill model
func NewModel(id uint32, level byte, masterLevel byte) Model {
	return Model{
		id:          id,
		level:       level,
		masterLevel: masterLevel,
	}
}

// Id returns the skill ID
func (m Model) Id() uint32 {
	return m.id
}

// Level returns the current skill level
func (m Model) Level() byte {
	return m.level
}

// MasterLevel returns the master level of the skill
func (m Model) MasterLevel() byte {
	return m.masterLevel
}

// Expiration returns when the skill expires
func (m Model) Expiration() time.Time {
	return m.expiration
}

// CooldownExpiresAt returns when the cooldown expires
func (m Model) CooldownExpiresAt() time.Time {
	return m.cooldownExpiresAt
}

// RestModel represents the REST representation of a skill
type RestModel struct {
	Id                uint32    `json:"-"`
	Level             byte      `json:"level"`
	MasterLevel       byte      `json:"masterLevel"`
	Expiration        time.Time `json:"expiration"`
	CooldownExpiresAt time.Time `json:"cooldownExpiresAt"`
}

// GetName returns the resource name for JSON:API
func (r RestModel) GetName() string {
	return "skills"
}

// Extract transforms a RestModel into a domain Model
func Extract(r RestModel) (Model, error) {
	return Model{
		id:                r.Id,
		level:             r.Level,
		masterLevel:       r.MasterLevel,
		expiration:        r.Expiration,
		cooldownExpiresAt: r.CooldownExpiresAt,
	}, nil
}

// ModelBuilder provides a fluent API for building skill models
type ModelBuilder struct {
	id                uint32
	level             byte
	masterLevel       byte
	expiration        time.Time
	cooldownExpiresAt time.Time
}

// NewModelBuilder creates a new ModelBuilder
func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

// SetId sets the skill ID
func (b *ModelBuilder) SetId(id uint32) *ModelBuilder {
	b.id = id
	return b
}

// SetLevel sets the skill level
func (b *ModelBuilder) SetLevel(level byte) *ModelBuilder {
	b.level = level
	return b
}

// SetMasterLevel sets the master level
func (b *ModelBuilder) SetMasterLevel(masterLevel byte) *ModelBuilder {
	b.masterLevel = masterLevel
	return b
}

// SetExpiration sets the expiration time
func (b *ModelBuilder) SetExpiration(expiration time.Time) *ModelBuilder {
	b.expiration = expiration
	return b
}

// SetCooldownExpiresAt sets the cooldown expiration time
func (b *ModelBuilder) SetCooldownExpiresAt(cooldownExpiresAt time.Time) *ModelBuilder {
	b.cooldownExpiresAt = cooldownExpiresAt
	return b
}

// Build creates the Model from the builder
func (b *ModelBuilder) Build() Model {
	return Model{
		id:                b.id,
		level:             b.level,
		masterLevel:       b.masterLevel,
		expiration:        b.expiration,
		cooldownExpiresAt: b.cooldownExpiresAt,
	}
}
