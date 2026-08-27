package marriage

// Builder provides a builder pattern for creating marriage models
type Builder struct {
	characterId         uint32
	hasUnclaimedGifts   bool
	unclaimedGiftCount  int
	lastGiftClaimedTime int64
}

// NewBuilder creates a new marriage model builder
func NewBuilder() *Builder {
	return &Builder{
		hasUnclaimedGifts:   false,
		unclaimedGiftCount:  0,
		lastGiftClaimedTime: 0,
	}
}

// SetCharacterId sets the character ID
func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

// SetHasUnclaimedGifts sets whether the character has unclaimed gifts
func (b *Builder) SetHasUnclaimedGifts(hasGifts bool) *Builder {
	b.hasUnclaimedGifts = hasGifts
	return b
}

// SetUnclaimedGiftCount sets the number of unclaimed gifts
func (b *Builder) SetUnclaimedGiftCount(count int) *Builder {
	b.unclaimedGiftCount = count
	b.hasUnclaimedGifts = count > 0
	return b
}

// SetLastGiftClaimedTime sets the timestamp of the last gift claimed
func (b *Builder) SetLastGiftClaimedTime(timestamp int64) *Builder {
	b.lastGiftClaimedTime = timestamp
	return b
}

// Build creates a marriage model from the builder
func (b *Builder) Build() Model {
	return Model{
		characterId:         b.characterId,
		hasUnclaimedGifts:   b.hasUnclaimedGifts,
		unclaimedGiftCount:  b.unclaimedGiftCount,
		lastGiftClaimedTime: b.lastGiftClaimedTime,
	}
}
