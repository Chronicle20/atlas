package character

import (
	"atlas-rates/data/cash"
	"atlas-rates/data/equipment"
	"atlas-rates/rate"
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// BonusExpTier is an alias for equipment.BonusExpTier for convenience
type BonusExpTier = equipment.BonusExpTier

type Processor interface {
	GetRates(ch channel.Model, characterId uint32) (rate.Computed, []rate.Factor, error)
	AddFactor(ch channel.Model, characterId uint32, source string, rateType rate.Type, multiplier float64) error
	RemoveFactor(characterId uint32, source string, rateType rate.Type) error
	RemoveFactorsBySource(characterId uint32, source string) error
	UpdateWorldRate(worldId world.Id, rateType rate.Type, multiplier float64)
	AddBuffFactor(ch channel.Model, characterId uint32, buffSourceId int32, rateType rate.Type, multiplier float64) error
	RemoveBuffFactor(characterId uint32, buffSourceId int32, rateType rate.Type) error
	RemoveAllBuffFactors(characterId uint32, buffSourceId int32) error
	// Item rate factor methods (simple, non-time-based)
	AddItemFactor(ch channel.Model, characterId uint32, templateId uint32, rateType rate.Type, multiplier float64) error
	RemoveItemFactor(characterId uint32, templateId uint32, rateType rate.Type) error
	RemoveAllItemFactors(characterId uint32, templateId uint32) error
	// Time-based item tracking methods
	TrackBonusExpItem(characterId uint32, templateId uint32, tiers []BonusExpTier, equippedSince *time.Time) error
	TrackCouponItem(characterId uint32, templateId uint32, rateType rate.Type, rateMultiplier float64, durationMins int32, createdAt time.Time, timeWindows []cash.TimeWindow) error
	UntrackItem(characterId uint32, templateId uint32) error
	UpdateBonusExpEquippedSince(characterId uint32, templateId uint32, equippedSince *time.Time) error
	GetItemRateFactors(characterId uint32) []rate.Factor
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetRates retrieves computed rates and factors for a character
// This includes time-based item factors calculated dynamically
// If the character hasn't been initialized yet, this will lazily initialize them
func (p *ProcessorImpl) GetRates(ch channel.Model, characterId uint32) (rate.Computed, []rate.Factor, error) {
	// Lazy initialization - ensure character's item tracking is set up
	// This handles the case where atlas-rates restarts while characters are online
	if !IsInitialized(p.ctx, characterId) {
		InitializeCharacterRates(p.l, p.ctx, characterId, ch)
	}

	m := GetRegistry().GetOrCreate(p.ctx, ch, characterId)

	// Get base factors from registry
	baseFactors := m.Factors()

	// Get dynamic item factors (calculated based on current time)
	itemFactors := p.GetItemRateFactors(characterId)

	// Combine all factors
	allFactors := append(baseFactors, itemFactors...)

	// Compute rates from combined factors
	computed := rate.ComputeFromFactors(allFactors)

	return computed, allFactors, nil
}

// AddFactor adds or updates a rate factor for a character
func (p *ProcessorImpl) AddFactor(ch channel.Model, characterId uint32, source string, rateType rate.Type, multiplier float64) error {
	f := rate.NewFactor(source, rateType, multiplier)
	m := GetRegistry().AddFactor(p.ctx, ch, characterId, f)
	p.l.Debugf("Added factor [%s] for character [%d]: %s = %.2f", source, characterId, rateType, multiplier)
	p.l.Debugf("Character [%d] now has rates: exp=%.2f, meso=%.2f, drop=%.2f, quest=%.2f",
		characterId, m.ComputedRates().ExpRate(), m.ComputedRates().MesoRate(),
		m.ComputedRates().ItemDropRate(), m.ComputedRates().QuestExpRate())
	return nil
}

// RemoveFactor removes a specific rate factor for a character
func (p *ProcessorImpl) RemoveFactor(characterId uint32, source string, rateType rate.Type) error {
	m, err := GetRegistry().RemoveFactor(p.ctx, characterId, source, rateType)
	if err != nil {
		return err
	}
	p.l.Debugf("Removed factor [%s] type [%s] for character [%d]", source, rateType, characterId)
	p.l.Debugf("Character [%d] now has rates: exp=%.2f, meso=%.2f, drop=%.2f, quest=%.2f",
		characterId, m.ComputedRates().ExpRate(), m.ComputedRates().MesoRate(),
		m.ComputedRates().ItemDropRate(), m.ComputedRates().QuestExpRate())
	return nil
}

// RemoveFactorsBySource removes all factors from a specific source for a character
func (p *ProcessorImpl) RemoveFactorsBySource(characterId uint32, source string) error {
	m, err := GetRegistry().RemoveFactorsBySource(p.ctx, characterId, source)
	if err != nil {
		return err
	}
	p.l.Debugf("Removed all factors from source [%s] for character [%d]", source, characterId)
	p.l.Debugf("Character [%d] now has rates: exp=%.2f, meso=%.2f, drop=%.2f, quest=%.2f",
		characterId, m.ComputedRates().ExpRate(), m.ComputedRates().MesoRate(),
		m.ComputedRates().ItemDropRate(), m.ComputedRates().QuestExpRate())
	return nil
}

// UpdateWorldRate updates the world rate for all characters in a world
func (p *ProcessorImpl) UpdateWorldRate(worldId world.Id, rateType rate.Type, multiplier float64) {
	GetRegistry().UpdateWorldRate(p.ctx, worldId, rateType, multiplier)
	p.l.Infof("Updated world [%d] rate [%s] to %.2f", worldId, rateType, multiplier)
}

// AddBuffFactor adds a rate factor from a buff
func (p *ProcessorImpl) AddBuffFactor(ch channel.Model, characterId uint32, buffSourceId int32, rateType rate.Type, multiplier float64) error {
	source := fmt.Sprintf("buff:%d", buffSourceId)
	return p.AddFactor(ch, characterId, source, rateType, multiplier)
}

// RemoveBuffFactor removes a specific buff rate factor
func (p *ProcessorImpl) RemoveBuffFactor(characterId uint32, buffSourceId int32, rateType rate.Type) error {
	source := fmt.Sprintf("buff:%d", buffSourceId)
	return p.RemoveFactor(characterId, source, rateType)
}

// RemoveAllBuffFactors removes all rate factors from a specific buff
func (p *ProcessorImpl) RemoveAllBuffFactors(characterId uint32, buffSourceId int32) error {
	source := fmt.Sprintf("buff:%d", buffSourceId)
	return p.RemoveFactorsBySource(characterId, source)
}

// AddItemFactor adds a rate factor from an item (equipment or cash coupon)
func (p *ProcessorImpl) AddItemFactor(ch channel.Model, characterId uint32, templateId uint32, rateType rate.Type, multiplier float64) error {
	source := fmt.Sprintf("item:%d", templateId)
	return p.AddFactor(ch, characterId, source, rateType, multiplier)
}

// RemoveItemFactor removes a specific item rate factor
func (p *ProcessorImpl) RemoveItemFactor(characterId uint32, templateId uint32, rateType rate.Type) error {
	source := fmt.Sprintf("item:%d", templateId)
	return p.RemoveFactor(characterId, source, rateType)
}

// RemoveAllItemFactors removes all rate factors from a specific item
func (p *ProcessorImpl) RemoveAllItemFactors(characterId uint32, templateId uint32) error {
	source := fmt.Sprintf("item:%d", templateId)
	return p.RemoveFactorsBySource(characterId, source)
}

// TrackBonusExpItem starts tracking an equipment item with time-based EXP bonus tiers
func (p *ProcessorImpl) TrackBonusExpItem(characterId uint32, templateId uint32, tiers []BonusExpTier, equippedSince *time.Time) error {
	item := TrackedItem{
		TemplateId:    templateId,
		ItemType:      ItemTypeBonusExp,
		RateType:      rate.TypeExp,
		BonusExpTiers: tiers,
		EquippedSince: equippedSince,
	}

	GetItemTracker().TrackItem(p.ctx, characterId, item)
	p.l.Debugf("Started tracking bonusExp item [%d] for character [%d] with %d tiers, equippedSince [%v].",
		templateId, characterId, len(tiers), equippedSince)
	return nil
}

// TrackCouponItem starts tracking a cash coupon with time-limited rate bonus
func (p *ProcessorImpl) TrackCouponItem(characterId uint32, templateId uint32, rateType rate.Type, rateMultiplier float64, durationMins int32, createdAt time.Time, timeWindows []cash.TimeWindow) error {
	item := TrackedItem{
		TemplateId:   templateId,
		ItemType:     ItemTypeCoupon,
		AcquiredAt:   createdAt,
		RateType:     rateType,
		BaseRate:     rateMultiplier,
		DurationMins: durationMins,
		TimeWindows:  timeWindows,
	}

	GetItemTracker().TrackItem(p.ctx, characterId, item)
	if len(timeWindows) > 0 {
		p.l.Debugf("Started tracking coupon [%d] for character [%d]: rate type [%s], multiplier [%.2f], duration [%d mins], createdAt [%v], time windows [%d].",
			templateId, characterId, rateType, rateMultiplier, durationMins, createdAt, len(timeWindows))
	} else {
		p.l.Debugf("Started tracking coupon [%d] for character [%d]: rate type [%s], multiplier [%.2f], duration [%d mins], createdAt [%v], always active.",
			templateId, characterId, rateType, rateMultiplier, durationMins, createdAt)
	}
	return nil
}

// UntrackItem stops tracking a time-based rate item
func (p *ProcessorImpl) UntrackItem(characterId uint32, templateId uint32) error {
	GetItemTracker().UntrackItem(p.ctx, characterId, templateId)
	p.l.Debugf("Stopped tracking item [%d] for character [%d].", templateId, characterId)
	return nil
}

// UpdateBonusExpEquippedSince updates the equippedSince timestamp for a bonusExp item
func (p *ProcessorImpl) UpdateBonusExpEquippedSince(characterId uint32, templateId uint32, equippedSince *time.Time) error {
	GetItemTracker().UpdateEquippedSince(p.ctx, characterId, templateId, equippedSince)
	if equippedSince != nil {
		p.l.Debugf("Updated bonusExp item [%d] for character [%d]: equippedSince = %v.", templateId, characterId, *equippedSince)
	} else {
		p.l.Debugf("Cleared bonusExp item [%d] equippedSince for character [%d] (unequipped).", templateId, characterId)
	}
	return nil
}

// GetItemRateFactors returns current rate factors from all tracked items
func (p *ProcessorImpl) GetItemRateFactors(characterId uint32) []rate.Factor {
	// Clean up expired items first
	expired := GetItemTracker().CleanupExpiredItems(p.ctx, characterId)
	for _, templateId := range expired {
		p.l.Debugf("Cleaned up expired coupon [%d] for character [%d].", templateId, characterId)
	}

	return GetItemTracker().ComputeItemRateFactors(p.l, p.ctx, characterId)
}
