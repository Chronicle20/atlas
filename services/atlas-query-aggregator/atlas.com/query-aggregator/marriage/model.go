package marriage

// Model represents marriage gift data for a character
type Model struct {
	characterId            uint32
	hasUnclaimedGifts      bool
	unclaimedGiftCount     int
	lastGiftClaimedTime    int64
}

// NewModel creates a new marriage model
func NewModel(characterId uint32, hasUnclaimedGifts bool) Model {
	return Model{
		characterId:         characterId,
		hasUnclaimedGifts:   hasUnclaimedGifts,
		unclaimedGiftCount:  0,
		lastGiftClaimedTime: 0,
	}
}

// CharacterId returns the character ID
func (m Model) CharacterId() uint32 {
	return m.characterId
}

// HasUnclaimedGifts returns whether the character has unclaimed marriage gifts
func (m Model) HasUnclaimedGifts() bool {
	return m.hasUnclaimedGifts
}

// UnclaimedGiftCount returns the number of unclaimed gifts
func (m Model) UnclaimedGiftCount() int {
	return m.unclaimedGiftCount
}

// LastGiftClaimedTime returns the timestamp of the last gift claimed
func (m Model) LastGiftClaimedTime() int64 {
	return m.lastGiftClaimedTime
}

// SetUnclaimedGiftCount sets the number of unclaimed gifts
func (m Model) SetUnclaimedGiftCount(count int) Model {
	return Model{
		characterId:         m.characterId,
		hasUnclaimedGifts:   count > 0,
		unclaimedGiftCount:  count,
		lastGiftClaimedTime: m.lastGiftClaimedTime,
	}
}

// SetLastGiftClaimedTime sets the timestamp of the last gift claimed
func (m Model) SetLastGiftClaimedTime(timestamp int64) Model {
	return Model{
		characterId:         m.characterId,
		hasUnclaimedGifts:   m.hasUnclaimedGifts,
		unclaimedGiftCount:  m.unclaimedGiftCount,
		lastGiftClaimedTime: timestamp,
	}
}

// ModelBuilder provides a builder pattern for creating marriage models
type ModelBuilder struct {
	characterId         uint32
	hasUnclaimedGifts   bool
	unclaimedGiftCount  int
	lastGiftClaimedTime int64
}

// NewModelBuilder creates a new marriage model builder
func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{
		hasUnclaimedGifts:   false,
		unclaimedGiftCount:  0,
		lastGiftClaimedTime: 0,
	}
}

// SetCharacterId sets the character ID
func (b *ModelBuilder) SetCharacterId(characterId uint32) *ModelBuilder {
	b.characterId = characterId
	return b
}

// SetHasUnclaimedGifts sets whether the character has unclaimed gifts
func (b *ModelBuilder) SetHasUnclaimedGifts(hasGifts bool) *ModelBuilder {
	b.hasUnclaimedGifts = hasGifts
	return b
}

// SetUnclaimedGiftCount sets the number of unclaimed gifts
func (b *ModelBuilder) SetUnclaimedGiftCount(count int) *ModelBuilder {
	b.unclaimedGiftCount = count
	b.hasUnclaimedGifts = count > 0
	return b
}

// SetLastGiftClaimedTime sets the timestamp of the last gift claimed
func (b *ModelBuilder) SetLastGiftClaimedTime(timestamp int64) *ModelBuilder {
	b.lastGiftClaimedTime = timestamp
	return b
}

// Build creates a marriage model from the builder
func (b *ModelBuilder) Build() Model {
	return Model{
		characterId:         b.characterId,
		hasUnclaimedGifts:   b.hasUnclaimedGifts,
		unclaimedGiftCount:  b.unclaimedGiftCount,
		lastGiftClaimedTime: b.lastGiftClaimedTime,
	}
}

// RestModel represents the REST representation of marriage gift data
type RestModel struct {
	CharacterId         uint32 `json:"characterId"`
	HasUnclaimedGifts   bool   `json:"hasUnclaimedGifts"`
	UnclaimedGiftCount  int    `json:"unclaimedGiftCount"`
	LastGiftClaimedTime int64  `json:"lastGiftClaimedTime"`
}

// Extract transforms a RestModel into a domain Model
func Extract(r RestModel) (Model, error) {
	return NewModelBuilder().
		SetCharacterId(r.CharacterId).
		SetHasUnclaimedGifts(r.HasUnclaimedGifts).
		SetUnclaimedGiftCount(r.UnclaimedGiftCount).
		SetLastGiftClaimedTime(r.LastGiftClaimedTime).
		Build(), nil
}