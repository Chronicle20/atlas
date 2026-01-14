package monster

// Clone creates a ModelBuilder initialized from an existing Model.
// This centralizes field copying for immutable model mutations.
func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		uniqueId:           m.uniqueId,
		worldId:            m.worldId,
		channelId:          m.channelId,
		mapId:              m.mapId,
		maxHp:              m.maxHp,
		hp:                 m.hp,
		maxMp:              m.maxMp,
		mp:                 m.mp,
		monsterId:          m.monsterId,
		controlCharacterId: m.controlCharacterId,
		x:                  m.x,
		y:                  m.y,
		fh:                 m.fh,
		stance:             m.stance,
		team:               m.team,
		damageEntries:      m.damageEntries,
	}
}

// ModelBuilder provides a fluent interface for creating Model instances.
type ModelBuilder struct {
	uniqueId           uint32
	worldId            byte
	channelId          byte
	mapId              uint32
	maxHp              uint32
	hp                 uint32
	maxMp              uint32
	mp                 uint32
	monsterId          uint32
	controlCharacterId uint32
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	team               int8
	damageEntries      []entry
}

// SetX sets the X coordinate.
func (b *ModelBuilder) SetX(x int16) *ModelBuilder {
	b.x = x
	return b
}

// SetY sets the Y coordinate.
func (b *ModelBuilder) SetY(y int16) *ModelBuilder {
	b.y = y
	return b
}

// SetStance sets the stance/animation state.
func (b *ModelBuilder) SetStance(stance byte) *ModelBuilder {
	b.stance = stance
	return b
}

// SetHp sets the current hit points.
func (b *ModelBuilder) SetHp(hp uint32) *ModelBuilder {
	b.hp = hp
	return b
}

// SetControlCharacterId sets the controlling character ID.
func (b *ModelBuilder) SetControlCharacterId(id uint32) *ModelBuilder {
	b.controlCharacterId = id
	return b
}

// AddDamageEntry appends a damage entry to the damage tracking list.
func (b *ModelBuilder) AddDamageEntry(characterId uint32, damage uint32) *ModelBuilder {
	b.damageEntries = append(b.damageEntries, entry{
		CharacterId: characterId,
		Damage:      damage,
	})
	return b
}

// Build creates an immutable Model from the builder state.
func (b *ModelBuilder) Build() Model {
	return Model{
		uniqueId:           b.uniqueId,
		worldId:            b.worldId,
		channelId:          b.channelId,
		mapId:              b.mapId,
		maxHp:              b.maxHp,
		hp:                 b.hp,
		maxMp:              b.maxMp,
		mp:                 b.mp,
		monsterId:          b.monsterId,
		controlCharacterId: b.controlCharacterId,
		x:                  b.x,
		y:                  b.y,
		fh:                 b.fh,
		stance:             b.stance,
		team:               b.team,
		damageEntries:      b.damageEntries,
	}
}
