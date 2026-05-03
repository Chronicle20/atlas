package mock

import (
	"time"

	"atlas-rates/character"
	"atlas-rates/data/cash"
	"atlas-rates/rate"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// ProcessorMock is a nil-safe stub of character.Processor. Each *Func field can
// be set in a test to capture or override behavior; otherwise the method
// returns the type's zero value (or nil error).
type ProcessorMock struct {
	GetRatesFunc                    func(ch channel.Model, characterId uint32) (rate.Computed, []rate.Factor, error)
	AddFactorFunc                   func(ch channel.Model, characterId uint32, source string, rateType rate.Type, multiplier float64) error
	RemoveFactorFunc                func(characterId uint32, source string, rateType rate.Type) error
	RemoveFactorsBySourceFunc       func(characterId uint32, source string) error
	UpdateWorldRateFunc             func(worldId world.Id, rateType rate.Type, multiplier float64)
	AddBuffFactorFunc               func(ch channel.Model, characterId uint32, buffSourceId int32, rateType rate.Type, multiplier float64) error
	RemoveBuffFactorFunc            func(characterId uint32, buffSourceId int32, rateType rate.Type) error
	RemoveAllBuffFactorsFunc        func(characterId uint32, buffSourceId int32) error
	AddItemFactorFunc               func(ch channel.Model, characterId uint32, templateId uint32, rateType rate.Type, multiplier float64) error
	RemoveItemFactorFunc            func(characterId uint32, templateId uint32, rateType rate.Type) error
	RemoveAllItemFactorsFunc        func(characterId uint32, templateId uint32) error
	TrackBonusExpItemFunc           func(characterId uint32, templateId uint32, tiers []character.BonusExpTier, equippedSince *time.Time) error
	TrackCouponItemFunc             func(characterId uint32, templateId uint32, rateType rate.Type, rateMultiplier float64, durationMins int32, createdAt time.Time, timeWindows []cash.TimeWindow) error
	UntrackItemFunc                 func(characterId uint32, templateId uint32) error
	UpdateBonusExpEquippedSinceFunc func(characterId uint32, templateId uint32, equippedSince *time.Time) error
	GetItemRateFactorsFunc          func(characterId uint32) []rate.Factor
}

func (m *ProcessorMock) GetRates(ch channel.Model, characterId uint32) (rate.Computed, []rate.Factor, error) {
	if m.GetRatesFunc != nil {
		return m.GetRatesFunc(ch, characterId)
	}
	return rate.Computed{}, nil, nil
}

func (m *ProcessorMock) AddFactor(ch channel.Model, characterId uint32, source string, rateType rate.Type, multiplier float64) error {
	if m.AddFactorFunc != nil {
		return m.AddFactorFunc(ch, characterId, source, rateType, multiplier)
	}
	return nil
}

func (m *ProcessorMock) RemoveFactor(characterId uint32, source string, rateType rate.Type) error {
	if m.RemoveFactorFunc != nil {
		return m.RemoveFactorFunc(characterId, source, rateType)
	}
	return nil
}

func (m *ProcessorMock) RemoveFactorsBySource(characterId uint32, source string) error {
	if m.RemoveFactorsBySourceFunc != nil {
		return m.RemoveFactorsBySourceFunc(characterId, source)
	}
	return nil
}

func (m *ProcessorMock) UpdateWorldRate(worldId world.Id, rateType rate.Type, multiplier float64) {
	if m.UpdateWorldRateFunc != nil {
		m.UpdateWorldRateFunc(worldId, rateType, multiplier)
	}
}

func (m *ProcessorMock) AddBuffFactor(ch channel.Model, characterId uint32, buffSourceId int32, rateType rate.Type, multiplier float64) error {
	if m.AddBuffFactorFunc != nil {
		return m.AddBuffFactorFunc(ch, characterId, buffSourceId, rateType, multiplier)
	}
	return nil
}

func (m *ProcessorMock) RemoveBuffFactor(characterId uint32, buffSourceId int32, rateType rate.Type) error {
	if m.RemoveBuffFactorFunc != nil {
		return m.RemoveBuffFactorFunc(characterId, buffSourceId, rateType)
	}
	return nil
}

func (m *ProcessorMock) RemoveAllBuffFactors(characterId uint32, buffSourceId int32) error {
	if m.RemoveAllBuffFactorsFunc != nil {
		return m.RemoveAllBuffFactorsFunc(characterId, buffSourceId)
	}
	return nil
}

func (m *ProcessorMock) AddItemFactor(ch channel.Model, characterId uint32, templateId uint32, rateType rate.Type, multiplier float64) error {
	if m.AddItemFactorFunc != nil {
		return m.AddItemFactorFunc(ch, characterId, templateId, rateType, multiplier)
	}
	return nil
}

func (m *ProcessorMock) RemoveItemFactor(characterId uint32, templateId uint32, rateType rate.Type) error {
	if m.RemoveItemFactorFunc != nil {
		return m.RemoveItemFactorFunc(characterId, templateId, rateType)
	}
	return nil
}

func (m *ProcessorMock) RemoveAllItemFactors(characterId uint32, templateId uint32) error {
	if m.RemoveAllItemFactorsFunc != nil {
		return m.RemoveAllItemFactorsFunc(characterId, templateId)
	}
	return nil
}

func (m *ProcessorMock) TrackBonusExpItem(characterId uint32, templateId uint32, tiers []character.BonusExpTier, equippedSince *time.Time) error {
	if m.TrackBonusExpItemFunc != nil {
		return m.TrackBonusExpItemFunc(characterId, templateId, tiers, equippedSince)
	}
	return nil
}

func (m *ProcessorMock) TrackCouponItem(characterId uint32, templateId uint32, rateType rate.Type, rateMultiplier float64, durationMins int32, createdAt time.Time, timeWindows []cash.TimeWindow) error {
	if m.TrackCouponItemFunc != nil {
		return m.TrackCouponItemFunc(characterId, templateId, rateType, rateMultiplier, durationMins, createdAt, timeWindows)
	}
	return nil
}

func (m *ProcessorMock) UntrackItem(characterId uint32, templateId uint32) error {
	if m.UntrackItemFunc != nil {
		return m.UntrackItemFunc(characterId, templateId)
	}
	return nil
}

func (m *ProcessorMock) UpdateBonusExpEquippedSince(characterId uint32, templateId uint32, equippedSince *time.Time) error {
	if m.UpdateBonusExpEquippedSinceFunc != nil {
		return m.UpdateBonusExpEquippedSinceFunc(characterId, templateId, equippedSince)
	}
	return nil
}

func (m *ProcessorMock) GetItemRateFactors(characterId uint32) []rate.Factor {
	if m.GetItemRateFactorsFunc != nil {
		return m.GetItemRateFactorsFunc(characterId)
	}
	return nil
}
