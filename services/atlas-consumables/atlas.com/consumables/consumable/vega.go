package consumable

import (
	"atlas-consumables/asset"
	"atlas-consumables/character"
	"atlas-consumables/compartment"
	consumable3 "atlas-consumables/data/consumable"
	compartment2 "atlas-consumables/kafka/message/compartment"
	"atlas-consumables/kafka/message/consumable"
	once "atlas-consumables/kafka/once/compartment"
	"atlas-consumables/kafka/producer"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// vegaRates returns the natural scroll success rate a Vega's Spell requires
// (exact match only) and the boosted rate it applies. This is server policy
// (PRD FR-4.1), not WZ data — the Item.wz entries for 0561 carry only info
// nodes. Non-vega ids return ok=false.
func vegaRates(id item2.Id) (required uint32, boosted uint32, ok bool) {
	switch id {
	case item2.VegasSpell10:
		return 10, 30, true
	case item2.VegasSpell60:
		return 60, 90, true
	}
	return 0, 0, false
}

// VegaReservation identifies one reservation the vega chain may need to roll
// back: compartment type + slot.
type VegaReservation struct {
	inventoryType inventory2.Type
	slot          int16
}

// VegaScrollError cancels any reservations already made on the vega path and
// emits the VEGA_INVALID error event. The channel answers with the VEGA
// INVALID packet + enable-actions; the client shows its own "This item cannot
// be used." notice and closes the dialog (required — after sending, the
// client is excl-request-blocked and a silent rejection would wedge it).
func (p *ProcessorImpl) VegaScrollError(characterId uint32, transactionId uuid.UUID, reservations []VegaReservation, err error) error {
	p.l.Debugf("Character [%d] unable to vega scroll due to error: [%v]", characterId, err)
	for _, r := range reservations {
		if cErr := p.cpp.CancelItemReservation(characterId, r.inventoryType, transactionId, r.slot); cErr != nil {
			p.l.WithError(cErr).Errorf("Unable to cancel item reservation at inventory [%d] slot [%d] for character [%d] as part of transaction [%s].", r.inventoryType, r.slot, characterId, transactionId)
		}
	}
	if cErr := producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(ErrorEventProvider(ts.Id(characterId), consumable.ErrorTypeVegaInvalid)); cErr != nil {
		p.l.WithError(cErr).Errorf("Unable to issue vega error event. Character [%d] likely going to be stuck.", characterId)
	}
	return err
}

// resolveVegaEquip resolves the vega dialog's equip target. The dialog's
// equips come from the equip INVENTORY (positive slots — the drop handler
// stores the drag-source position; design §2.7), while the classic scroll
// path only addresses EQUIPPED items (negative positions). Negative slots are
// honored defensively with the classic resolution.
func resolveVegaEquip(c character.Model, equipSlot int16) (*asset.Model, error) {
	if equipSlot < 0 {
		s, err := slot.GetSlotByPosition(slot.Position(equipSlot))
		if err != nil {
			return nil, errors.New("failed to locate equipment being scrolled")
		}
		sm, ok := c.Equipment().Get(s.Type)
		if !ok || sm.Equipable == nil {
			return nil, errors.New("failed to locate equipment being scrolled")
		}
		return sm.Equipable, nil
	}
	a, ok := c.Inventory().Equipable().FindBySlot(equipSlot)
	if !ok {
		return nil, errors.New("failed to locate equipment being scrolled")
	}
	return a, nil
}

// RequestVegaScroll validates everything up front (FR-2: every rejection here
// consumes nothing), then starts the chained single-item reservation flow
// (design §3.2): CASH vega first; its RESERVED confirmation triggers the USE
// scroll reservation; the scroll's confirmation triggers ConsumeVegaScroll.
// NEVER batch the two reserves — the inventory-side batch path only processes
// the first entry (design §2.8).
func (p *ProcessorImpl) RequestVegaScroll(characterId uint32, vegaSlot int16, vegaItemId item2.Id, scrollSlot int16, equipSlot int16) error {
	cp := character.NewProcessor(p.l, p.ctx)
	cpp := compartment.NewProcessor(p.l, p.ctx)
	transactionId := uuid.New()

	required, boosted, ok := vegaRates(vegaItemId)
	if !ok || !item2.IsVegasSpell(vegaItemId) {
		return p.VegaScrollError(characterId, transactionId, nil, errors.New("not a vega scroll item"))
	}

	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}

	vegaItem, ok := c.Inventory().Cash().FindBySlot(vegaSlot)
	if !ok || item2.Id(vegaItem.TemplateId()) != vegaItemId {
		return p.VegaScrollError(characterId, transactionId, nil, errors.New("vega item not found"))
	}

	scrollItem, ok := c.Inventory().Consumable().FindBySlot(scrollSlot)
	if !ok {
		return p.VegaScrollError(characterId, transactionId, nil, errors.New("scroll item not found"))
	}

	ci, err := p.cdp.GetById(scrollItem.TemplateId())
	if err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}
	if ci.SuccessRate() != required {
		p.l.Debugf("Character [%d] vega [%d] rejected: scroll [%d] rate [%d] does not match required [%d].", characterId, vegaItemId, scrollItem.TemplateId(), ci.SuccessRate(), required)
		return p.VegaScrollError(characterId, transactionId, nil, errors.New("scroll rate mismatch"))
	}

	equip, err := resolveVegaEquip(c, equipSlot)
	if err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}
	if !p.ValidateScrollUse(*scrollItem, *equip) {
		return p.VegaScrollError(characterId, transactionId, nil, errors.New("failed slot validation"))
	}

	p.l.Debugf("Character [%d] using vega [%d]: scroll [%d] (slot [%d], rate [%d] boosted to [%d]) onto equip slot [%d] (transaction [%s]).",
		characterId, vegaItemId, scrollItem.TemplateId(), scrollSlot, required, boosted, equipSlot, transactionId.String())

	t, _ := topic.EnvProvider(p.l)(compartment2.EnvEventTopicStatus)()
	scrollValidator := once.ReservationValidator(transactionId, scrollItem.TemplateId())
	scrollHandler := compartment.Consume(ConsumeVegaScroll(transactionId, characterId, vegaItem, scrollItem, equipSlot, boosted))
	if _, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(scrollValidator, scrollHandler))); err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}
	vegaValidator := once.ReservationValidator(transactionId, vegaItem.TemplateId())
	vegaHandler := compartment.Consume(ReserveVegaScrollStage(transactionId, characterId, vegaItem, scrollItem))
	if _, err = consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(vegaValidator, vegaHandler))); err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}

	err = cpp.RequestReserve(transactionId, characterId, inventory2.TypeValueCash, []compartment.Reserves{{
		Slot:     vegaSlot,
		ItemId:   vegaItem.TemplateId(),
		Quantity: 1,
	}})
	if err != nil {
		return p.VegaScrollError(characterId, transactionId, nil, err)
	}
	return nil
}

// ReserveVegaScrollStage fires when the vega CASH reservation confirms; it
// issues the second (USE scroll) reservation of the chain. A synchronous
// producer failure cancels the vega reservation. An asynchronous inventory-
// side rejection emits nothing — the vega reservation TTL-expires (~30s) and
// the player keeps everything (design §2.9).
func ReserveVegaScrollStage(transactionId uuid.UUID, characterId uint32, vegaItem *asset.Model, scrollItem *asset.Model) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)
			l.Debugf("Character [%d] vega reservation confirmed (transaction [%s]); reserving scroll in slot [%d].", characterId, transactionId.String(), scrollItem.Slot())
			err := cpp.RequestReserve(transactionId, characterId, inventory2.TypeValueUse, []compartment.Reserves{{
				Slot:     scrollItem.Slot(),
				ItemId:   scrollItem.TemplateId(),
				Quantity: 1,
			}})
			if err != nil {
				return p.VegaScrollError(characterId, transactionId, []VegaReservation{{inventory2.TypeValueCash, vegaItem.Slot()}}, err)
			}
			return nil
		}
	}
}

// ConsumeVegaScroll fires when both reservations are confirmed: re-validates
// (state may have moved between request and confirmation), applies the scroll
// at the boosted rate via the shared core (whiteScroll=false), commits both
// reservations, handles curse destruction, and emits the VEGA_SCROLL event.
func ConsumeVegaScroll(transactionId uuid.UUID, characterId uint32, vegaItem *asset.Model, scrollItem *asset.Model, equipSlot int16, boostedProb uint32) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			cp := character.NewProcessor(l, ctx)
			cpp := compartment.NewProcessor(l, ctx)
			cdp := consumable3.NewProcessor(l, ctx)
			both := []VegaReservation{
				{inventory2.TypeValueUse, scrollItem.Slot()},
				{inventory2.TypeValueCash, vegaItem.Slot()},
			}

			l.Debugf("Character [%d] has reserved vega [%d] and scroll [%d]. Applying scroll at boosted rate [%d] (transaction [%s]).", characterId, vegaItem.TemplateId(), scrollItem.TemplateId(), boostedProb, transactionId.String())
			c, err := cp.GetById(cp.InventoryDecorator)(characterId)
			if err != nil {
				return p.VegaScrollError(characterId, transactionId, both, err)
			}

			required, _, _ := vegaRates(item2.Id(vegaItem.TemplateId()))
			ci, err := cdp.GetById(scrollItem.TemplateId())
			if err != nil {
				return p.VegaScrollError(characterId, transactionId, both, err)
			}
			if ci.SuccessRate() != required {
				return p.VegaScrollError(characterId, transactionId, both, errors.New("scroll rate mismatch"))
			}
			equip, err := resolveVegaEquip(c, equipSlot)
			if err != nil {
				return p.VegaScrollError(characterId, transactionId, both, err)
			}
			if !p.ValidateScrollUse(*scrollItem, *equip) {
				return p.VegaScrollError(characterId, transactionId, both, errors.New("failed slot validation"))
			}

			// whiteScroll=false and legendarySpirit=false throughout (FR-4.2).
			outcome, err := applyScrollCore(l, ctx, transactionId, characterId, ci, scrollItem, equip, boostedProb, false)
			if err != nil {
				return p.VegaScrollError(characterId, transactionId, both, err)
			}

			if err = cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, scrollItem.Slot()); err != nil {
				l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", scrollItem.TemplateId(), characterId)
			}
			if err = cpp.ConsumeItem(characterId, inventory2.TypeValueCash, transactionId, vegaItem.Slot()); err != nil {
				l.WithError(err).Errorf("Unable to consume item [%d] for character [%d] used during scrolling.", vegaItem.TemplateId(), characterId)
			}
			if outcome.cursed {
				if err = cpp.DestroyItem(characterId, inventory2.TypeValueEquip, equipSlot); err != nil {
					l.WithError(err).Errorf("Unable to destroy item in slot [%d] for character [%d] during scrolling.", equipSlot, characterId)
				}
			}
			return producer.ProviderImpl(l)(ctx)(consumable.EnvEventTopic)(VegaScrollEventProvider(ts.Id(characterId))(outcome.success, outcome.cursed))
		}
	}
}
