package character

import (
	"atlas-rates/buffs"
	"atlas-rates/data/cash"
	"atlas-rates/data/equipment"
	"atlas-rates/inventory"
	"atlas-rates/kafka/message/buff"
	"atlas-rates/rate"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// initializedCharacters tracks which characters have been initialized to avoid re-processing
var initializedCharacters = struct {
	sync.RWMutex
	m map[tenant.Model]map[uint32]bool
}{m: make(map[tenant.Model]map[uint32]bool)}

// IsInitialized checks if a character has been initialized
func IsInitialized(t tenant.Model, characterId uint32) bool {
	initializedCharacters.RLock()
	defer initializedCharacters.RUnlock()

	if chars, ok := initializedCharacters.m[t]; ok {
		return chars[characterId]
	}
	return false
}

// MarkInitialized marks a character as initialized
func MarkInitialized(t tenant.Model, characterId uint32) {
	initializedCharacters.Lock()
	defer initializedCharacters.Unlock()

	if _, ok := initializedCharacters.m[t]; !ok {
		initializedCharacters.m[t] = make(map[uint32]bool)
	}
	initializedCharacters.m[t][characterId] = true
}

// ClearInitialized clears the initialization state for a character (e.g., on logout)
func ClearInitialized(t tenant.Model, characterId uint32) {
	initializedCharacters.Lock()
	defer initializedCharacters.Unlock()

	if chars, ok := initializedCharacters.m[t]; ok {
		delete(chars, characterId)
	}
}

// ResetInitializedForTesting clears all initialization state (testing only)
func ResetInitializedForTesting() {
	initializedCharacters.Lock()
	defer initializedCharacters.Unlock()
	initializedCharacters.m = make(map[tenant.Model]map[uint32]bool)
}

// InitializeCharacterRates queries inventory and buffs to initialize rate tracking for a character
// This is called lazily when rates are queried or on map change events
// worldId and channelId are optional (pass 0 if not available) - used for buff factor registration
func InitializeCharacterRates(l logrus.FieldLogger, ctx context.Context, characterId uint32, worldId, channelId byte) {
	t := tenant.MustFromContext(ctx)

	// Check if already initialized
	if IsInitialized(t, characterId) {
		return
	}

	l.Debugf("Initializing rate tracking for character [%d].", characterId)

	// Mark as initialized before processing to prevent duplicate processing
	MarkInitialized(t, characterId)

	p := NewProcessor(l, ctx)

	// Initialize equipped items with bonusExp
	initializeEquippedItems(l, ctx, p, characterId)

	// Initialize cash coupons with rate properties
	initializeCashCoupons(l, ctx, p, characterId)

	// Initialize active buffs with rate effects
	initializeActiveBuffs(l, ctx, characterId, worldId, channelId)
}

// initializeEquippedItems queries equipped items and tracks those with bonusExp
func initializeEquippedItems(l logrus.FieldLogger, ctx context.Context, p Processor, characterId uint32) {
	equipped, err := inventory.GetEquippedAssets(l)(ctx)(characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to get equipped assets for character [%d]. Skipping equipment initialization.", characterId)
		return
	}

	for _, asset := range equipped {
		// Check if this equipment has bonusExp
		equipData, err := equipment.GetById(l)(ctx)(asset.TemplateId)
		if err != nil {
			l.Debugf("Failed to get equipment data for template [%d]: %v", asset.TemplateId, err)
			continue
		}

		if !equipData.HasBonusExp() {
			continue
		}

		// Get equippedSince from the asset's reference data (from atlas-equipables)
		equippedSince := asset.GetEquippedSince()
		if equippedSince == nil {
			// Item is in equipped slot but equippedSince is not set
			// This can happen if the equipment was equipped before this feature was added
			// Fall back to current time
			l.Warnf("Equipment [%d] for character [%d] has no equippedSince, using current time.", asset.TemplateId, characterId)
			now := time.Now()
			equippedSince = &now
		}

		// Convert atlas-data BonusExpTier to character.BonusExpTier
		tiers := make([]BonusExpTier, len(equipData.BonusExp))
		for i, t := range equipData.BonusExp {
			tiers[i] = BonusExpTier{
				IncExpR:   t.IncExpR,
				TermStart: t.TermStart,
			}
		}

		l.Infof("Initializing bonusExp tracking for equipment [%d] with %d tiers, equippedSince [%v] for character [%d].",
			asset.TemplateId, len(tiers), equippedSince, characterId)

		if err := p.TrackBonusExpItem(characterId, asset.TemplateId, tiers, equippedSince); err != nil {
			l.WithError(err).Errorf("Unable to track bonusExp equipment for character [%d].", characterId)
		}
	}
}

// initializeCashCoupons queries cash items and tracks those with rate properties
func initializeCashCoupons(l logrus.FieldLogger, ctx context.Context, p Processor, characterId uint32) {
	cashAssets, err := inventory.GetCashAssets(l)(ctx)(characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to get cash assets for character [%d]. Skipping coupon initialization.", characterId)
		return
	}

	for _, asset := range cashAssets {
		// Check if this is a cash item with rate properties
		cashData, err := cash.GetById(l)(ctx)(asset.TemplateId)
		if err != nil {
			l.Debugf("Failed to get cash data for template [%d]: %v", asset.TemplateId, err)
			continue
		}

		if !cashData.HasRateProperties() {
			continue
		}

		// Determine rate type based on item ID range
		rateType := GetRateTypeFromTemplateId(asset.TemplateId)
		if rateType == "" {
			continue
		}

		// Get createdAt from the asset's reference data
		createdAt := asset.GetCreatedAt()
		if createdAt.IsZero() {
			l.Warnf("Cash item [%d] for character [%d] has no createdAt, using current time.", asset.TemplateId, characterId)
			createdAt = time.Now()
		}

		rateMultiplier := float64(cashData.GetRate())
		durationMins := cashData.GetTime()

		l.Infof("Initializing coupon tracking for item [%d], rate type [%s], multiplier [%.2f], duration [%d mins], createdAt [%v] for character [%d].",
			asset.TemplateId, rateType, rateMultiplier, durationMins, createdAt, characterId)

		if err := p.TrackCouponItem(characterId, asset.TemplateId, rateType, rateMultiplier, durationMins, createdAt); err != nil {
			l.WithError(err).Errorf("Unable to track coupon for character [%d].", characterId)
		}
	}
}

// GetRateTypeFromTemplateId determines the rate type based on the item's template ID
func GetRateTypeFromTemplateId(templateId uint32) rate.Type {
	// Item IDs are structured as: category (05) + subcategory (21 or 36) + specific item
	// 5210000-5219999 = EXP coupons
	// 5360000-5369999 = Drop coupons
	if templateId >= 5210000 && templateId <= 5219999 {
		return rate.TypeExp
	}
	if templateId >= 5360000 && templateId <= 5369999 {
		return rate.TypeItemDrop
	}
	return ""
}

// initializeActiveBuffs queries atlas-buffs for active buffs and initializes rate factors
func initializeActiveBuffs(l logrus.FieldLogger, ctx context.Context, characterId uint32, worldId, channelId byte) {
	activeBuffs, err := buffs.GetActiveBuffs(l)(ctx)(characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to get active buffs for character [%d]. Skipping buff initialization.", characterId)
		return
	}

	if len(activeBuffs) == 0 {
		l.Debugf("No active buffs for character [%d].", characterId)
		return
	}

	t := tenant.MustFromContext(ctx)

	for _, b := range activeBuffs {
		// Check if buff has expired (belt-and-suspenders check)
		if !b.ExpiresAt.IsZero() && time.Now().After(b.ExpiresAt) {
			continue
		}

		// Process each stat change for rate-affecting buffs
		for _, change := range b.Changes {
			mapping, exists := buff.GetRateMapping(change.Type)
			if !exists {
				continue
			}

			rateType := rate.Type(mapping.RateType)
			if rateType == "" {
				continue
			}

			// Calculate multiplier using the appropriate conversion method
			multiplier := buff.CalculateMultiplier(change.Amount, mapping.Conversion)

			source := fmt.Sprintf("buff:%d", b.SourceId)

			l.Infof("Initializing buff rate factor: source [%s], stat type [%s] -> rate type [%s], amount [%d] -> multiplier [%.2f] for character [%d].",
				source, change.Type, rateType, change.Amount, multiplier, characterId)

			// Add the factor to the registry
			f := rate.NewFactor(source, rateType, multiplier)
			GetRegistry().AddFactor(t, worldId, channelId, characterId, f)
		}
	}
}
