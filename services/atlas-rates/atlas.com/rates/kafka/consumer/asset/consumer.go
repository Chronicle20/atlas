package asset

import (
	"atlas-rates/character"
	"atlas-rates/data/cash"
	"atlas-rates/data/equipment"
	consumer2 "atlas-rates/kafka/consumer"
	"atlas-rates/kafka/message/asset"
	"context"
	"time"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_status")(asset.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(asset.EnvEventTopicStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetCreated)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetDeleted)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetMoved)))
	}
}

// handleAssetCreated handles CREATED events for cash coupons
func handleAssetCreated(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.CreatedStatusEventBody]) {
	if e.Type != asset.StatusEventTypeCreated {
		return
	}

	l.Debugf("Processing asset created event for character [%d], template [%d], slot [%d].",
		e.CharacterId, e.TemplateId, e.Slot)

	// Check if this is a cash item with rate properties
	cashData, err := cash.GetById(l)(ctx)(e.TemplateId)
	if err != nil {
		l.Debugf("Item [%d] is not a cash item or failed to fetch: %v", e.TemplateId, err)
		return
	}

	if !cashData.HasRateProperties() {
		l.Debugf("Cash item [%d] does not have rate properties.", e.TemplateId)
		return
	}

	// Determine rate type based on item ID range
	// 0521xxxx = EXP coupons, 0536xxxx = Drop coupons
	rateType := character.GetRateTypeFromTemplateId(e.TemplateId)
	if rateType == "" {
		l.Debugf("Cash item [%d] is not a recognized rate coupon.", e.TemplateId)
		return
	}

	// Get rate multiplier and duration
	rateMultiplier := float64(cashData.GetRate())
	durationMins := cashData.GetTime()

	// Get createdAt from event's reference data
	createdAt := e.Body.GetCreatedAt()

	l.Infof("Tracking cash coupon: item [%d], rate type [%s], multiplier [%.2f], duration [%d mins], createdAt [%v] for character [%d].",
		e.TemplateId, rateType, rateMultiplier, durationMins, createdAt, e.CharacterId)

	p := character.NewProcessor(l, ctx)
	if err := p.TrackCouponItem(e.CharacterId, e.TemplateId, rateType, rateMultiplier, durationMins, createdAt); err != nil {
		l.WithError(err).Errorf("Unable to track coupon for character [%d].", e.CharacterId)
	}
}

// handleAssetDeleted handles DELETED events to remove item rate factors
func handleAssetDeleted(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.DeletedStatusEventBody]) {
	if e.Type != asset.StatusEventTypeDeleted {
		return
	}

	l.Debugf("Processing asset deleted event for character [%d], template [%d].",
		e.CharacterId, e.TemplateId)

	p := character.NewProcessor(l, ctx)

	// Untrack time-based item (coupons, bonusExp equipment)
	if err := p.UntrackItem(e.CharacterId, e.TemplateId); err != nil {
		l.WithError(err).Debugf("Unable to untrack item for character [%d], item [%d].",
			e.CharacterId, e.TemplateId)
	}

	// Also remove any static factors (for backwards compatibility)
	if err := p.RemoveAllItemFactors(e.CharacterId, e.TemplateId); err != nil {
		l.WithError(err).Debugf("Unable to remove item factors for character [%d], item [%d] (may not have had any).",
			e.CharacterId, e.TemplateId)
	}
}

// handleAssetMoved handles MOVED events for equip/unequip of bonusExp items
func handleAssetMoved(l logrus.FieldLogger, ctx context.Context, e asset.StatusEvent[asset.MovedStatusEventBody]) {
	if e.Type != asset.StatusEventTypeMoved {
		return
	}

	oldSlot := e.Body.OldSlot
	newSlot := e.Slot
	createdAt := e.Body.CreatedAt

	l.Debugf("Processing asset moved event for character [%d], template [%d], from slot [%d] to slot [%d], createdAt [%v].",
		e.CharacterId, e.TemplateId, oldSlot, newSlot, createdAt)

	p := character.NewProcessor(l, ctx)

	if asset.IsEquipAction(oldSlot, newSlot) {
		// Item was equipped - check for bonusExp
		handleItemEquipped(l, ctx, p, e.CharacterId, e.TemplateId, createdAt)
	} else if asset.IsUnequipAction(oldSlot, newSlot) {
		// Item was unequipped - remove any rate factors
		handleItemUnequipped(l, ctx, p, e.CharacterId, e.TemplateId)
	}
}

func handleItemEquipped(l logrus.FieldLogger, ctx context.Context, p character.Processor, characterId uint32, templateId uint32, _ time.Time) {
	// Check if this equipment has bonusExp
	equipData, err := equipment.GetById(l)(ctx)(templateId)
	if err != nil {
		l.Debugf("Item [%d] is not equipment or failed to fetch: %v", templateId, err)
		return
	}

	if !equipData.HasBonusExp() {
		l.Debugf("Equipment [%d] does not have bonusExp.", templateId)
		return
	}

	// Convert atlas-data BonusExpTier to character.BonusExpTier
	tiers := make([]character.BonusExpTier, len(equipData.BonusExp))
	for i, t := range equipData.BonusExp {
		tiers[i] = character.BonusExpTier{
			IncExpR:   t.IncExpR,
			TermStart: t.TermStart,
		}
	}

	// equippedSince = now (the item is being equipped right now)
	equippedSince := time.Now()

	l.Infof("Tracking bonusExp equipment: item [%d] with %d tiers, equippedSince [%v] for character [%d].",
		templateId, len(tiers), equippedSince, characterId)

	if err := p.TrackBonusExpItem(characterId, templateId, tiers, &equippedSince); err != nil {
		l.WithError(err).Errorf("Unable to track bonusExp equipment for character [%d].", characterId)
	}
}

func handleItemUnequipped(l logrus.FieldLogger, ctx context.Context, p character.Processor, characterId uint32, templateId uint32) {
	l.Debugf("Removing any rate factors for unequipped item [%d] from character [%d].", templateId, characterId)

	// Untrack time-based item (bonusExp equipment)
	if err := p.UntrackItem(characterId, templateId); err != nil {
		l.WithError(err).Debugf("Unable to untrack item for character [%d], item [%d].",
			characterId, templateId)
	}

	// Also remove any static factors (for backwards compatibility)
	if err := p.RemoveAllItemFactors(characterId, templateId); err != nil {
		l.WithError(err).Debugf("Unable to remove item factors for character [%d], item [%d] (may not have had any).",
			characterId, templateId)
	}
}

