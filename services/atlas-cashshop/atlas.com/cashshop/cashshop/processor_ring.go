package cashshop

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/cashshop"
	cashshop2 "atlas-cashshop/kafka/producer/cashshop"
	"atlas-cashshop/ledger"
	"atlas-cashshop/purchaserecord"
	"atlas-cashshop/ring"
	"atlas-cashshop/wallet"
	"errors"

	"github.com/google/uuid"

	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// errRingRejected is the internal sentinel used to abort the
// PurchaseRingAndEmit transaction closure on a handled rejection whose event
// must fire on the DIRECT producer path rather than the outbox -- mirrors
// errGiftRejected (cashshop/gift.go) and errPurchaseRejected
// (cashshop/processor.go), for the same reason: message.Emit only flushes
// its buffer when the wrapped closure returns nil, so a rejection event
// enqueued through mb on a failing branch would be silently dropped.
var errRingRejected = errors.New("ring purchase rejected")

// ringTypeAndOperation resolves the wire RingType string to the ring
// package's Type and the ErrorEventBody.Operation this purchase reports
// rejections under. The two are the SAME string ("COUPLE" / "FRIENDSHIP")
// by construction: ring.TypeCouple/TypeFriendship and
// cashshop.ErrorOperationCouple/ErrorOperationFriendship were both defined
// as that literal, but this keeps the mapping in one place rather than
// asserting the coincidence at every call site.
func ringTypeAndOperation(ringType string) (ring.Type, string, bool) {
	switch ringType {
	case string(ring.TypeCouple):
		return ring.TypeCouple, cashshop.ErrorOperationCouple, true
	case string(ring.TypeFriendship):
		return ring.TypeFriendship, cashshop.ErrorOperationFriendship, true
	default:
		return "", "", false
	}
}

// PurchaseRingAndEmit implements REQUEST_RING_PURCHASE (task-240 task 19):
// charges the BUYER's wallet once for one commodity, then mints TWO ring
// items -- one into the buyer's own locker, one into partnerCharacterId's --
// and records them as a single pair via ring.CreatePair, atomically and
// idempotently.
//
// Item-template selection (design.md §4.3, OQ-R1): the arm carries one
// serialNumber (one commodity), but a pair needs two ring items. Both halves
// are minted from the SAME resolved commodity ItemId -- the confirmed-correct
// case for friendship rings. There is deliberately no code path here that
// derives a distinct partner template (no +1, no gender-based rule): this
// service has no data source today that distinguishes a couple-ring
// commodity needing distinct halves from an ordinary same-template ring, so
// there is nothing to detect and branch on. See context.md's open items for
// the typed COUPLE_FAILED/FRIENDSHIP_FAILED rejection this ruling reserves
// for if/when that data becomes available.
//
// Every rejection on this path reports ringType's own operation
// (cashshop.ErrorOperationCouple or cashshop.ErrorOperationFriendship, from
// ringTypeAndOperation) as ErrorEventBody.Operation, mirroring GiftAndEmit's
// use of cashshop.ErrorOperationGift.
func (p *ProcessorImpl) PurchaseRingAndEmit(characterId uint32, transactionId uuid.UUID, currency uint32, serialNumber uint32, partnerCharacterId uint32, senderName string, ringMessage string, ringType string) error {
	rt, operation, ok := ringTypeAndOperation(ringType)
	if !ok {
		p.l.Errorf("Character [%d] requested ring purchase with unknown ringType [%s].", characterId, ringType)
		_ = producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, cashshop.ErrorOperationFriendship, "UNKNOWN_ERROR", transactionId))
		return errRingRejected
	}

	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			reject := func(reason string) error {
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, reason, transactionId))
				}
				return errRingRejected
			}

			// Step 1: claim the transaction id FIRST, before any read or
			// write, so a Kafka redelivery aborts before touching state.
			// ErrAlreadyProcessed is success-without-effect (the original
			// purchase already told the client) -- no event, no error.
			if err := ledger.Claim(p.ctx, tx, transactionId, cashshop.CommandTypeRequestRingPurchase, characterId); err != nil {
				if errors.Is(err, ledger.ErrAlreadyProcessed) {
					return nil
				}
				p.l.WithError(err).Errorf("Unable to claim ring purchase transaction [%s] for character [%d].", transactionId, characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 2: resolve the commodity by serial number.
			ci, err := p.comP.GetById(serialNumber)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve commodity [%d] for ring purchase from character [%d].", serialNumber, characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 3: resolve the buyer's and the partner's account.
			buyer, err := p.chaP.GetById()(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve buyer character [%d] for ring purchase.", characterId)
				return reject("UNKNOWN_ERROR")
			}
			partner, err := p.chaP.GetById()(partnerCharacterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve partner character [%d] for ring purchase.", partnerCharacterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 4: resolve BOTH compartments and check BOTH have room --
			// a half-created pair (one asset without its partner) is exactly
			// the failure this domain's placement exists to prevent, so
			// neither asset may be created unless both lockers can hold one.
			cicP := compartment.NewProcessor(p.l, p.ctx, tx)
			buyerCompartmentType := compartmentTypeFor(buyer.JobId())
			buyerCcm, err := cicP.GetByAccountIdAndType(buyer.AccountId(), buyerCompartmentType)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve buyer [%d] compartment for ring purchase.", characterId)
				return reject("UNKNOWN_ERROR")
			}
			if buyerCcm.Capacity() <= uint32(len(buyerCcm.Assets())) {
				p.l.Debugf("Buyer [%d] has no room for ring purchase. Compartment [%s] capacity [%d].", characterId, buyerCcm.Id(), buyerCcm.Capacity())
				return reject("INVENTORY_FULL")
			}

			partnerCompartmentType := compartmentTypeFor(partner.JobId())
			partnerCcm, err := cicP.GetByAccountIdAndType(partner.AccountId(), partnerCompartmentType)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve partner [%d] compartment for ring purchase.", partnerCharacterId)
				return reject("UNKNOWN_ERROR")
			}
			if partnerCcm.Capacity() <= uint32(len(partnerCcm.Assets())) {
				p.l.Debugf("Partner [%d] has no room for ring purchase. Compartment [%s] capacity [%d].", partnerCharacterId, partnerCcm.Id(), partnerCcm.Capacity())
				return reject("PARTNER_INVENTORY_FULL")
			}

			// Step 5: check and debit the BUYER's wallet. Only the buyer
			// pays -- the partner receives their half for free, mirroring a
			// gift.
			walP := wallet.NewProcessor(p.l, p.ctx, tx)
			w, err := walP.GetByAccountId(buyer.AccountId())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve buyer [%d] wallet for ring purchase.", characterId)
				return reject("UNKNOWN_ERROR")
			}
			balance := w.Balance(currency)
			if balance < ci.Price() {
				p.l.Debugf("Character [%d] has insufficient balance for ring purchase. Cost [%d]. Balance [%d].", characterId, ci.Price(), balance)
				return reject("NOT_ENOUGH_CASH")
			}
			w = w.Purchase(currency, ci.Price())
			_, err = walP.Update(buf)(buyer.AccountId())(w.Credit())(w.Points())(w.Prepaid())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to debit buyer [%d] wallet for ring purchase.", characterId)
				return err
			}

			// Step 6: create the buyer's own half, then the partner's half
			// carrying GiftFrom/GiftMessage (FR-RING-3 -- from the partner's
			// locker this looks exactly like a gift from the buyer).
			effectiveCurrency := effectivePurchaseCurrency(currency)
			astP := asset.NewProcessor(p.l, p.ctx, tx)
			buyerAsset, err := astP.Create(buf)(buyerCcm.Id(), ci.ItemId(), serialNumber, effectiveCurrency, ci.Count(), 0, characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create buyer [%d] ring asset.", characterId)
				return reject("UNKNOWN_ERROR")
			}
			partnerAsset, err := astP.CreateGift(buf)(partnerCcm.Id(), ci.ItemId(), serialNumber, effectiveCurrency, ci.Count(), 0, characterId, senderName, ringMessage)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create partner [%d] ring asset.", partnerCharacterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 7: record BOTH halves as one pair. ring.CreatePair inserts
			// both rows in a single batched db.Create -- the statement lands
			// or neither half does, so this cannot leave a pair row for only
			// one of the two assets created above.
			pairId, err := ring.NewProcessor(p.l, p.ctx, tx, p.chaP).CreatePair(tx, rt,
				ring.Half{CharacterId: characterId, AssetId: buyerAsset.Id(), ItemTemplateId: ci.ItemId()},
				ring.Half{CharacterId: partnerCharacterId, AssetId: partnerAsset.Id(), ItemTemplateId: ci.ItemId()},
			)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to record ring pair for character [%d] and partner [%d].", characterId, partnerCharacterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 8: record the purchase against the BUYER's account.
			if err := purchaserecord.Record(tx, p.t.Id(), buyer.AccountId(), serialNumber); err != nil {
				p.l.WithError(err).Errorf("Unable to record ring purchase for character [%d].", characterId)
				return reject("UNKNOWN_ERROR")
			}

			p.l.Debugf("Character [%d] purchased ring pair [%s] (type [%s]) with partner [%d] for [%d] currency.", characterId, pairId, rt, partnerCharacterId, ci.Price())
			return buf.Put(cashshop.EnvEventTopicStatus, cashshop2.RingPurchasedStatusEventProvider(characterId, transactionId, buyerCcm.Id(), buyerAsset.Id(), partner.Name(), ci.ItemId(), uint16(ci.Count()), string(rt), pairId))
		})
	})
	if rejectEmit != nil {
		_ = rejectEmit()
		return nil
	}
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to complete ring purchase for character [%d].", characterId)
		return txErr
	}
	return nil
}

// compartmentTypeFor mirrors the explorer/cygnus/legend selection Purchase
// and GiftAndEmit make (cashshop/processor.go, cashshop/gift.go).
func compartmentTypeFor(jobId job.Id) compartment.CompartmentType {
	if job.GetType(jobId) == job.TypeExplorer {
		return compartment.TypeExplorer
	} else if job.GetType(jobId) == job.TypeCygnus {
		return compartment.TypeCygnus
	}
	return compartment.TypeLegend
}
