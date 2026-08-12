package consumable

import (
	"atlas-consumables/catchdelay"
	"atlas-consumables/compartment"
	consumable3 "atlas-consumables/data/consumable"
	"atlas-consumables/inventory"
	compartment2 "atlas-consumables/kafka/message/compartment"
	"atlas-consumables/kafka/message/consumable"
	monsterMsg "atlas-consumables/kafka/message/monster"
	once "atlas-consumables/kafka/once/compartment"
	"atlas-consumables/monster"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	ts "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

// validateCatchItem is the pre-reserve item gate (FR-3.2): the item must be a
// class-227 bridle consumable AND name a reward to grant. Classification comes
// from libs/atlas-constants rather than an ad-hoc itemId/10000 == 227 (DOM-21).
func validateCatchItem(itemId uint32, ci consumable3.Model) bool {
	if item2.GetClassification(item2.Id(itemId)) != item2.ClassificationConsumableCatchItem {
		return false
	}
	return ci.Create() != 0
}

func catchUseDelay(ci consumable3.Model) time.Duration {
	return time.Duration(ci.UseDelay()) * time.Millisecond
}

// RequestCatchMonster begins a bridle (catch-item) attempt: validate the item,
// arm the useDelay window, confirm the reward can be placed, then reserve the
// item and hand the monster-state decision to atlas-monsters. Modelled on
// RequestItemReward (processor.go:1079) — one transactionId spans
// reserve -> resolve -> commit.
//
// The item is NOT consumed here. Commit happens only on
// CATCH_RESOLVED(success=true), which satisfies "a failed catch does not consume
// the item" (FR-3.9) by construction rather than by compensation.
func (p *ProcessorImpl) RequestCatchMonster(f field.Model, characterId uint32, slot int16, itemId item2.Id, monsterUniqueId uint32) error {
	transactionId := uuid.New()

	ci, err := p.cdp.GetById(uint32(itemId))
	if err != nil {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, err)
	}
	if !validateCatchItem(uint32(itemId), ci) {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, errors.New("not a usable catch item"))
	}

	allowed, err := catchdelay.GetRegistry().Allow(p.ctx, characterId, uint32(itemId), catchUseDelay(ci))
	if err != nil {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, err)
	}
	if !allowed {
		return p.catchError(characterId, itemId, consumable.CatchCauseUseDelay, nil)
	}

	// atlas-inventory owns the merge-aware verdict. A full inventory fails here,
	// before anything is reserved and before a monster is removed.
	ok, err := p.ip.CanAccommodate(characterId, []inventory.AccommodationRequest{{ItemId: ci.Create(), Quantity: 1}})
	if err != nil {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, err)
	}
	if !ok {
		return p.catchError(characterId, itemId, consumable.CatchCauseInventoryFull, errors.New("inventory full"))
	}

	// Register the outcome handler BEFORE the reserve handler, and both BEFORE
	// RequestReserve, so no terminal event can race ahead of its handler.
	catchTopic, _ := topic.EnvProvider(p.l)(monsterMsg.EnvEventTopicCatch)()
	if _, err = consumer.GetManager().RegisterHandler(catchTopic, message.AdaptHandler(message.OneTimeConfig(catchResolvedValidator(characterId, itemId), catchResolutionHandler(transactionId, characterId, slot, itemId, ci.Create())))); err != nil {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, err)
	}

	compTopic, _ := topic.EnvProvider(p.l)(compartment2.EnvEventTopicStatus)()
	validator := once.ReservationValidator(transactionId, uint32(itemId))
	handler := compartment.Consume(ConsumeCatch(f, monsterUniqueId, characterId, itemId, transactionId, slot))
	if _, err = consumer.GetManager().RegisterHandler(compTopic, message.AdaptHandler(message.OneTimeConfig(validator, handler))); err != nil {
		return p.catchError(characterId, itemId, consumable.CatchCauseInvalidItem, err)
	}

	if err = p.cpp.RequestReserve(transactionId, characterId, inventory2.TypeValueUse, 30*time.Second, []compartment.Reserves{{Slot: slot, ItemId: uint32(itemId), Quantity: 1}}); err != nil {
		return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
	}
	return nil
}

// ConsumeCatch fires on RESERVED: the item is now held, so the monster-state
// decision can be handed to atlas-monsters, which is authoritative. If the
// CATCH command cannot even be produced (broker unavailable, serialization
// error), the reservation must not be left dangling: no CATCH_RESOLVED will
// ever arrive to drive catchResolutionHandler, so the once-handler registered
// in RequestCatchMonster would sit forever and the client would never unlock.
// Cancel via the same ConsumeError path every other ItemConsumer in this file
// uses (ConsumeBare, ConsumeMonsterCard, ConsumeStandard) rather than a new
// mechanism; ConsumeError's generic ERROR event is what drives the
// unrecognized-cause fallback unlock on the channel side.
//
// This is distinct from the failure path catchResolutionHandler already
// covers: this only fires when the command never made it to atlas-monsters at
// all, so there is exactly one cancel here, never both.
func ConsumeCatch(f field.Model, monsterUniqueId uint32, characterId uint32, itemId item2.Id, transactionId uuid.UUID, slot int16) ItemConsumer {
	return func(l logrus.FieldLogger) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			p := NewProcessor(l, ctx)
			if err := monster.NewProcessor(l, ctx).RequestCatch(f, monsterUniqueId, characterId, uint32(itemId)); err != nil {
				return p.ConsumeError(characterId, transactionId, inventory2.TypeValueUse, slot, err)
			}
			return nil
		}
	}
}

// catchError unsticks the client on a pre-reserve rejection: nothing is
// reserved, so this only reports the semantic cause. atlas-channel maps cause
// to the client's wire reason byte — this service never picks 0 or 1 (DOM-25).
func (p *ProcessorImpl) catchError(characterId uint32, itemId item2.Id, cause string, err error) error {
	p.l.Debugf("Character [%d] catch request with item [%d] rejected pre-reserve: cause [%s], err [%v].", characterId, itemId, cause, err)
	if cErr := producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(CatchFailedEventProvider(ts.Id(characterId), uint32(itemId), cause)); cErr != nil {
		p.l.WithError(cErr).Errorf("Unable to emit catch failure for character [%d]; client may be stuck.", characterId)
	}
	if err != nil {
		return err
	}
	return errors.New(cause)
}

// catchDecision is the pure branch the resolution handler takes. Keeping it
// pure is what makes the commit/cancel contract testable without Kafka.
type catchDecision struct {
	commit bool
	grant  bool
	cancel bool
}

func catchOutcome(success bool) catchDecision {
	if success {
		return catchDecision{commit: true, grant: true}
	}
	return catchDecision{cancel: true}
}

// catchResolvedValidator matches only this attempt's CATCH_RESOLVED event.
// Correlation is by (characterId, itemId) captured at reserve time rather than a
// transaction id on the wire, so the CATCH command body stays minimal (FR-3.7).
func catchResolvedValidator(characterId uint32, itemId item2.Id) message.Validator[monsterMsg.Event[monsterMsg.CatchResolvedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monsterMsg.Event[monsterMsg.CatchResolvedBody]) bool {
		return e.Type == monsterMsg.EventMonsterCatchResolved &&
			e.Body.CharacterId == characterId &&
			e.Body.ItemId == uint32(itemId)
	}
}

// catchResolutionHandler commits or cancels the reservation opened by
// RequestCatchMonster. On success it grants the create item through the same
// once-handler pair the reward-box flow uses, so a post-reserve creation failure
// cancels correctly. On failure it cancels and the item is untouched (FR-3.9) —
// and it emits NO consumable error event, because atlas-channel already renders
// the failure from the monster CATCH_FAILED event and two unlock packets would
// be sent otherwise.
func catchResolutionHandler(transactionId uuid.UUID, characterId uint32, slot int16, itemId item2.Id, createItemId uint32) message.Handler[monsterMsg.Event[monsterMsg.CatchResolvedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monsterMsg.Event[monsterMsg.CatchResolvedBody]) {
		p := NewProcessor(l, ctx).(*ProcessorImpl)
		d := catchOutcome(e.Body.Success)

		if d.cancel {
			if cErr := p.cpp.CancelItemReservation(characterId, inventory2.TypeValueUse, transactionId, slot); cErr != nil {
				l.WithError(cErr).Errorf("Unable to cancel catch reservation for character [%d] (transaction [%s]).", characterId, transactionId.String())
			}
			l.Debugf("Character [%d] catch failed (cause [%s]); item [%d] preserved.", characterId, e.Body.Cause, itemId)
			return
		}

		if d.commit {
			if cErr := p.cpp.ConsumeItem(characterId, inventory2.TypeValueUse, transactionId, slot); cErr != nil {
				l.WithError(cErr).Errorf("Catch succeeded but the item consume failed for character [%d] (transaction [%s]); needs ops intervention.", characterId, transactionId.String())
			}
		}
		if d.grant {
			if cErr := p.cpp.RequestCreateItem(transactionId, characterId, createItemId, 1, time.Time{}); cErr != nil {
				l.WithError(cErr).Errorf("Catch succeeded but granting reward item [%d] failed for character [%d].", createItemId, characterId)
			}
		}
	}
}
