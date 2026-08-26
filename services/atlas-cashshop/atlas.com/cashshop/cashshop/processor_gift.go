package cashshop

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
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

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// errGiftRejected is the internal sentinel used to abort the GiftAndEmit
// transaction closure on a handled rejection whose event must fire on the
// DIRECT producer path rather than the outbox -- mirrors errPurchaseRejected
// (cashshop/processor.go) and errRebateRejected (cashshop/rebate.go), for
// the same reason: message.Emit only flushes its buffer when the wrapped
// closure returns nil, so a rejection event enqueued through mb on a
// failing branch would be silently dropped.
var errGiftRejected = errors.New("gift rejected")

// GiftAndEmit implements REQUEST_GIFT_PURCHASE (task-240 task 13): charges
// the SENDER's wallet and creates the commodity in the RECIPIENT's locker,
// atomically and idempotently.
//
// Currency: a gift is always charged to the sender's credit/NX bucket
// (walletCurrencyCredit) -- RequestGiftPurchaseCommandBody carries no
// currency field at all, unlike RequestPurchaseCommandBody, because a gift
// is never paid for with Maple Points or prepaid NX. The created asset's
// Currency column is likewise recorded as walletCurrencyCredit, so a later
// rebate of a gifted item (if the recipient ever rebates it) credits the
// same bucket a genuine credit-funded purchase would.
//
// Every rejection on this path reports cashshop.ErrorOperationGift as
// ErrorEventBody.Operation and uses the reason strings the test table
// documents: NOT_ENOUGH_CASH, CANNOT_GIFT_RECIPIENT_INVENTORY_FULL (a
// tenant-template errors code with deliberately no Go constant -- see
// task-13-brief.md C3), and UNKNOWN_ERROR for every infra failure.
func (p *ProcessorImpl) GiftAndEmit(characterId uint32, transactionId uuid.UUID, serialNumber uint32, recipientCharacterId uint32, senderName string, giftMessage string) error {
	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			reject := func(reason string) error {
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, cashshop.ErrorOperationGift, reason, transactionId))
				}
				return errGiftRejected
			}

			// Step 1: claim the transaction id FIRST, before any read or
			// write, so a Kafka redelivery aborts before touching state.
			// ErrAlreadyProcessed is success-without-effect (the original
			// gift already told the client) -- no event, no error.
			if err := ledger.Claim(p.ctx, tx, transactionId, cashshop.CommandTypeRequestGiftPurchase, characterId); err != nil {
				if errors.Is(err, ledger.ErrAlreadyProcessed) {
					return nil
				}
				p.l.WithError(err).Errorf("Unable to claim gift transaction [%s] for character [%d].", transactionId, characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 2: resolve the commodity by serial number.
			ci, err := p.comP.GetById(serialNumber)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve commodity [%d] for gift from character [%d].", serialNumber, characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 3: resolve the sender's and the recipient's account.
			sender, err := p.chaP.GetById()(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve sender character [%d] for gift.", characterId)
				return reject("UNKNOWN_ERROR")
			}
			recipient, err := p.chaP.GetById()(recipientCharacterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve recipient character [%d] for gift.", recipientCharacterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 4: resolve the RECIPIENT's compartment for their job type
			// (the same explorer/cygnus/legend selection Purchase makes at
			// cashshop/processor.go:139-146) and check ITS capacity -- not
			// the sender's. This is the check that proves a gift delivers
			// into the recipient's own locker.
			var compartmentType compartment.CompartmentType
			if job.GetType(recipient.JobId()) == job.TypeExplorer {
				compartmentType = compartment.TypeExplorer
			} else if job.GetType(recipient.JobId()) == job.TypeCygnus {
				compartmentType = compartment.TypeCygnus
			} else {
				compartmentType = compartment.TypeLegend
			}

			cicP := compartment.NewProcessor(p.l, p.ctx, tx)
			ccm, err := cicP.GetByAccountIdAndType(recipient.AccountId(), compartmentType)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve recipient [%d] compartment for gift.", recipientCharacterId)
				return reject("UNKNOWN_ERROR")
			}
			if ccm.Capacity() <= uint32(len(ccm.Assets())) {
				p.l.Debugf("Recipient [%d] has no room for gift. Compartment [%s] capacity [%d].", recipientCharacterId, ccm.Id(), ccm.Capacity())
				return reject("CANNOT_GIFT_RECIPIENT_INVENTORY_FULL")
			}

			// Step 5: check and debit the SENDER's wallet.
			walP := wallet.NewProcessor(p.l, p.ctx, tx)
			w, err := walP.GetByAccountId(sender.AccountId())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve sender [%d] wallet for gift.", characterId)
				return reject("UNKNOWN_ERROR")
			}
			balance := w.Balance(walletCurrencyCredit)
			if balance < ci.Price() {
				p.l.Debugf("Character [%d] has insufficient balance for gift. Cost [%d]. Balance [%d].", characterId, ci.Price(), balance)
				return reject("NOT_ENOUGH_CASH")
			}
			w = w.Purchase(walletCurrencyCredit, ci.Price())
			_, err = walP.UpdateWithTransaction(buf)(transactionId)(sender.AccountId())(w.Credit())(w.Points())(w.Prepaid())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to debit sender [%d] wallet for gift.", characterId)
				return err
			}

			// Step 6: create the asset in the RECIPIENT's compartment,
			// carrying GiftFrom/GiftMessage, with PurchasedBy set to the
			// SENDER's character id.
			astP := asset.NewProcessor(p.l, p.ctx, tx)
			am, err := astP.CreateGift(buf)(ccm.Id(), ci.ItemId(), serialNumber, walletCurrencyCredit, ci.Count(), 0, characterId, senderName, giftMessage)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create gift asset for recipient [%d].", recipientCharacterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 7: record the purchase against the SENDER's account --
			// the sender bought it, not the recipient.
			if err := purchaserecord.Record(tx, p.t.Id(), sender.AccountId(), serialNumber); err != nil {
				p.l.WithError(err).Errorf("Unable to record gift purchase for sender [%d].", characterId)
				return reject("UNKNOWN_ERROR")
			}

			p.l.Debugf("Character [%d] gifted item [%d] (asset [%d]) to character [%d] for [%d] currency.", characterId, ci.ItemId(), am.Id(), recipientCharacterId, ci.Price())
			return buf.Put(cashshop.EnvEventTopicStatus, cashshop2.GiftPurchasedStatusEventProvider(characterId, transactionId, recipient.Name(), ci.ItemId(), uint16(ci.Count()), ci.Price(), recipientCharacterId))
		})
	})
	if rejectEmit != nil {
		_ = rejectEmit()
		return nil
	}
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to complete gift for character [%d].", characterId)
		return txErr
	}
	return nil
}
