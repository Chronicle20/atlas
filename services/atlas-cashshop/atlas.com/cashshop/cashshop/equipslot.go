package cashshop

// PurchaseEquipSlotAndEmit implements REQUEST_EQUIP_SLOT_INCREASE (task-240
// task 23, mode 9/10): charges the buyer's wallet once for one commodity,
// then queues the character's fixed equip-slot extension (the pendant2
// slot, libs/atlas-constants/inventory/slot -- see derivation-equip-slot.md
// E1 / R1) via atlas-character's write route. Unlike every other purchase
// in this file, there is no locker asset created here: the commodity's only
// effect is that extension.
//
// The atlas-character write itself does not happen inside this function
// (task-240 task 24c): it is an HTTP call with nothing local to roll back,
// so it is deferred behind the outbox (see CompleteEquipSlotExtension) and
// only performed once the wallet debit and purchase record recorded here
// have durably committed.

import (
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/cashshop"
	cashshop2 "atlas-cashshop/kafka/producer/cashshop"
	"atlas-cashshop/ledger"
	"atlas-cashshop/purchaserecord"
	"atlas-cashshop/wallet"
	"errors"

	"github.com/google/uuid"

	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// errEquipSlotRejected is the internal sentinel used to abort the
// PurchaseEquipSlotAndEmit transaction closure on a handled rejection whose
// event must fire on the DIRECT producer path rather than the outbox --
// mirrors errRingRejected (cashshop/ring.go) for the same reason.
var errEquipSlotRejected = errors.New("equip slot purchase rejected")

// PurchaseEquipSlotAndEmit charges characterId's wallet currency for
// serialNumber's commodity and, on success, extends the character's
// pendant2 equip-slot extension by the commodity's Period (in days).
func (p *ProcessorImpl) PurchaseEquipSlotAndEmit(characterId uint32, currency uint32, serialNumber uint32, transactionId uuid.UUID) error {
	pendant2, err := slot.GetSlotByType("pendant2")
	if err != nil {
		// Not reachable in production: pendant2 is a fixed entry in
		// libs/atlas-constants/inventory/slot, not caller input. Guarded
		// rather than assumed so a future rename of that table fails loudly
		// here instead of silently persisting the zero value.
		p.l.WithError(err).Errorf("Unable to resolve pendant2 slot for equip slot purchase by character [%d].", characterId)
		_ = producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, cashshop.ErrorOperationEnableEquipSlot, "UNKNOWN_ERROR", transactionId))
		return errEquipSlotRejected
	}
	slotIndex := int16(pendant2.Position)

	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			reject := func(reason string) error {
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, cashshop.ErrorOperationEnableEquipSlot, reason, transactionId))
				}
				return errEquipSlotRejected
			}

			// Step 1: claim the transaction id FIRST, before any read or
			// write, so a Kafka redelivery aborts before touching state.
			// ErrAlreadyProcessed is success-without-effect (the original
			// purchase already told the client) -- no event, no error.
			if err := ledger.Claim(p.ctx, tx, transactionId, cashshop.CommandTypeRequestEquipSlotIncrease, characterId); err != nil {
				if errors.Is(err, ledger.ErrAlreadyProcessed) {
					return nil
				}
				p.l.WithError(err).Errorf("Unable to claim equip slot purchase transaction [%s] for character [%d].", transactionId, characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 2: resolve the commodity by serial number.
			ci, err := p.comP.GetById(serialNumber)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve commodity [%d] for equip slot purchase from character [%d].", serialNumber, characterId)
				return reject("UNKNOWN_ERROR")
			}
			days := uint16(ci.Period())

			// Step 3: resolve the buyer's account.
			buyer, err := p.chaP.GetById()(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve character [%d] for equip slot purchase.", characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 4: check and debit the buyer's wallet.
			walP := wallet.NewProcessor(p.l, p.ctx, tx)
			w, err := walP.GetByAccountId(buyer.AccountId())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve wallet for character [%d] equip slot purchase.", characterId)
				return reject("UNKNOWN_ERROR")
			}
			balance := w.Balance(currency)
			if balance < ci.Price() {
				p.l.Debugf("Character [%d] has insufficient balance for equip slot purchase. Cost [%d]. Balance [%d].", characterId, ci.Price(), balance)
				return reject("NOT_ENOUGH_CASH")
			}
			w = w.Purchase(currency, ci.Price())
			_, err = walP.Update(buf)(buyer.AccountId())(w.Credit())(w.Points())(w.Prepaid())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to debit wallet for character [%d] equip slot purchase.", characterId)
				return err
			}

			// Step 5: record the purchase.
			if err := purchaserecord.Record(tx, p.t.Id(), buyer.AccountId(), serialNumber); err != nil {
				p.l.WithError(err).Errorf("Unable to record equip slot purchase for character [%d].", characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 6: mint the EXTEND_EQUIP_SLOT follow-up command via the
			// outbox rather than calling atlas-character's write route here
			// (task-240 task 24c). ExtendEquipSlot is an HTTP WRITE with
			// nothing local to roll back -- calling it inside this closure
			// meant a step 6 failure rolled back the wallet debit and the
			// ledger claim while the extension it had already granted stood,
			// permanently. Deferring it behind the outbox means the command
			// can only be published once THIS transaction (debit + record)
			// has durably committed, so that ordering defect cannot recur.
			// CompleteEquipSlotExtension (below) performs the call and emits
			// EQUIP_SLOT_INCREASED once atlas-character confirms it.
			p.l.Debugf("Character [%d] charged for equip slot [%d] extension of [%d] days; queuing atlas-character write.", characterId, slotIndex, days)
			return buf.Put(cashshop.EnvCommandTopic, cashshop2.ExtendEquipSlotCommandProvider(characterId, transactionId, slotIndex, days))
		})
	})
	if rejectEmit != nil {
		_ = rejectEmit()
		return nil
	}
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to complete equip slot purchase for character [%d].", characterId)
		return txErr
	}
	return nil
}

// CompleteEquipSlotExtension performs the atlas-character write that
// PurchaseEquipSlotAndEmit deferred behind the outbox (task-240 task 24c):
// invoked by the EXTEND_EQUIP_SLOT command consumer once that command is
// durably delivered, i.e. only after the wallet debit and purchase record
// have already committed. transactionId is the SAME idempotency key the
// purchase claimed; atlas-character's write route dedupes on it, so a
// redelivered command (the outbox is at-least-once) does not double-extend.
//
// A failure here is logged, not retried and not reported to the player as a
// purchase failure: the wallet was already charged and the purchase record
// already written, so telling the caller EQUIP_SLOT_INCREASE failed would be
// false, and reversing the charge from here would reintroduce the very
// cross-service rollback problem this task exists to remove. It leaves a
// charged-but-not-yet-extended character for manual reconciliation, which is
// the strictly better failure direction: recoverable, versus the original
// defect's unrecoverable free extension.
func (p *ProcessorImpl) CompleteEquipSlotExtension(characterId uint32, slotIndex int16, days uint16, transactionId uuid.UUID) error {
	if _, err := p.chaP.ExtendEquipSlot(characterId, slotIndex, days, transactionId); err != nil {
		p.l.WithError(err).Errorf("Unable to extend equip slot [%d] for character [%d] (transaction [%s]) after the purchase already committed; the charge stands and this requires reconciliation.", slotIndex, characterId, transactionId)
		return err
	}
	p.l.Debugf("Character [%d] extended equip slot [%d] by [%d] days (transaction [%s]).", characterId, slotIndex, days, transactionId)
	return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.EquipSlotIncreasedStatusEventProvider(characterId, transactionId, slotIndex, days))
}
