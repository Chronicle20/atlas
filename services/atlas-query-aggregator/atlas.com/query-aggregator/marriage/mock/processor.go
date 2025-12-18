package mock

import (
	"atlas-query-aggregator/marriage"
	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorImpl is a mock implementation of the marriage.Processor interface
type ProcessorImpl struct {
	GetMarriageGiftsFunc     func(characterId uint32) model.Provider[marriage.Model]
	HasUnclaimedGiftsFunc    func(characterId uint32) model.Provider[bool]
	GetUnclaimedGiftCountFunc func(characterId uint32) model.Provider[int]
}

// GetMarriageGifts returns the marriage gift data for a character
func (m *ProcessorImpl) GetMarriageGifts(characterId uint32) model.Provider[marriage.Model] {
	if m.GetMarriageGiftsFunc != nil {
		return m.GetMarriageGiftsFunc(characterId)
	}
	return func() (marriage.Model, error) {
		return marriage.NewModel(characterId, false), nil
	}
}

// HasUnclaimedGifts returns whether the character has unclaimed marriage gifts
func (m *ProcessorImpl) HasUnclaimedGifts(characterId uint32) model.Provider[bool] {
	if m.HasUnclaimedGiftsFunc != nil {
		return m.HasUnclaimedGiftsFunc(characterId)
	}
	return func() (bool, error) {
		return false, nil
	}
}

// GetUnclaimedGiftCount returns the number of unclaimed gifts for a character
func (m *ProcessorImpl) GetUnclaimedGiftCount(characterId uint32) model.Provider[int] {
	if m.GetUnclaimedGiftCountFunc != nil {
		return m.GetUnclaimedGiftCountFunc(characterId)
	}
	return func() (int, error) {
		return 0, nil
	}
}