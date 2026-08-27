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

// errPackageRejected is the internal sentinel used to abort the
// PurchasePackageAndEmit transaction closure on a handled rejection whose
// event must fire on the DIRECT producer path rather than the outbox --
// mirrors errPurchaseRejected (cashshop/processor.go), errRebateRejected
// (cashshop/rebate.go) and errGiftRejected (cashshop/gift.go), for the same
// reason: message.Emit only flushes its buffer when the wrapped closure
// returns nil, so a rejection event enqueued through mb on a failing branch
// would be silently dropped.
var errPackageRejected = errors.New("package purchase rejected")

// PurchasePackageAndEmit implements REQUEST_PACKAGE_PURCHASE (task-240 task
// 16): client modes 30 (buy-for-self) and 31 (gift) share this single
// entry point, discriminated by recipientCharacterId -- ZERO means
// buy-for-self, non-zero means the package's members are delivered into the
// named recipient's compartment instead of the buyer's own. Resolution,
// capacity, atomicity and pricing are identical between the two modes.
//
// Resolution chain (design §5.1) -- the package id is an ITEM id, not a
// serial number:
//
//	serialNumber -> commodity(serialNumber) -> commodity.ItemId == the CashPackage.img key
//	             -> cashPackage(ItemId).SerialNumbers -> member commodity serial numbers
//	             -> for each: commodity(memberSerialNumber) -> ItemId, Count, Price
//	             -> one cash_assets row per member
//
// The price charged once is the PACKAGE commodity's own Price -- resolved in
// step 1 above, so the "sum of the member commodities' prices" mistake is
// structurally impossible (FR-PKG-5): nothing downstream of that first
// resolution ever reads a member's Price.
//
// Every rejection reports ErrorOperationBuyPackage when recipientCharacterId
// is zero and ErrorOperationGiftPackage otherwise, so the channel's
// operation switch (task-240 task 3) answers on the arm the client is
// waiting on.
func (p *ProcessorImpl) PurchasePackageAndEmit(characterId uint32, transactionId uuid.UUID, currency uint32, serialNumber uint32, recipientCharacterId uint32, senderName string) error {
	operation := cashshop.ErrorOperationBuyPackage
	if recipientCharacterId != 0 {
		operation = cashshop.ErrorOperationGiftPackage
	}

	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			reject := func(reason string) error {
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, reason, transactionId))
				}
				return errPackageRejected
			}

			// Step 1: claim the transaction id FIRST, before any read or
			// write, so a Kafka redelivery aborts before touching state.
			// ErrAlreadyProcessed is success-without-effect (the original
			// purchase already told the client) -- no event, no error.
			if err := ledger.Claim(p.ctx, tx, transactionId, cashshop.CommandTypeRequestPackagePurchase, characterId); err != nil {
				if errors.Is(err, ledger.ErrAlreadyProcessed) {
					return nil
				}
				p.l.WithError(err).Errorf("Unable to claim package purchase transaction [%s] for character [%d].", transactionId, characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 2: resolve the package commodity by serial number. Its
			// Price is the amount charged once, below -- never the sum of
			// the member commodities' prices.
			pkgCommodity, err := p.comP.GetById(serialNumber)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve package commodity [%d] for character [%d].", serialNumber, characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 3: resolve the cash package by the commodity's item id
			// (the CashPackage.img key -- NOT the serial number).
			cp, err := p.dataCashPkgP.GetById(pkgCommodity.ItemId())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve cash package [%d] for character [%d].", pkgCommodity.ItemId(), characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 4: resolve EVERY member commodity up front. Any single
			// failure aborts before anything is written -- no asset for any
			// member is created on a partially-resolvable package.
			memberSerialNumbers := cp.SerialNumbers()
			memberCommodities := make([]struct {
				serialNumber uint32
				itemId       uint32
				count        uint32
			}, 0, len(memberSerialNumbers))
			for _, memberSerialNumber := range memberSerialNumbers {
				mc, mcErr := p.comP.GetById(memberSerialNumber)
				if mcErr != nil {
					p.l.WithError(mcErr).Errorf("Unable to resolve package member commodity [%d] for character [%d].", memberSerialNumber, characterId)
					return reject("UNKNOWN_ERROR")
				}
				memberCommodities = append(memberCommodities, struct {
					serialNumber uint32
					itemId       uint32
					count        uint32
				}{serialNumber: memberSerialNumber, itemId: mc.ItemId(), count: mc.Count()})
			}

			// Step 5: resolve the buyer, and the recipient when this is a
			// gift (recipientCharacterId != 0). The buyer's wallet always
			// pays; the target compartment is the recipient's on a gift and
			// the buyer's own otherwise.
			buyer, err := p.chaP.GetById()(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve buyer character [%d] for package purchase.", characterId)
				return reject("UNKNOWN_ERROR")
			}

			targetAccountId := buyer.AccountId()
			targetJobId := buyer.JobId()
			recipientName := buyer.Name()
			effectiveRecipientCharacterId := characterId
			if recipientCharacterId != 0 {
				recipient, rErr := p.chaP.GetById()(recipientCharacterId)
				if rErr != nil {
					p.l.WithError(rErr).Errorf("Unable to resolve recipient character [%d] for package purchase.", recipientCharacterId)
					return reject("UNKNOWN_ERROR")
				}
				targetAccountId = recipient.AccountId()
				targetJobId = recipient.JobId()
				recipientName = recipient.Name()
				effectiveRecipientCharacterId = recipientCharacterId
			}

			// Step 6: resolve the target compartment.
			var compartmentType compartment.CompartmentType
			if job.GetType(targetJobId) == job.TypeExplorer {
				compartmentType = compartment.TypeExplorer
			} else if job.GetType(targetJobId) == job.TypeCygnus {
				compartmentType = compartment.TypeCygnus
			} else {
				compartmentType = compartment.TypeLegend
			}

			cicP := compartment.NewProcessor(p.l, p.ctx, tx)
			ccm, err := cicP.GetByAccountIdAndType(targetAccountId, compartmentType)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve target compartment for package purchase by character [%d].", characterId)
				return reject("UNKNOWN_ERROR")
			}

			// Step 7: check capacity against the FULL member count before
			// charging anything -- the check that proves this precedes the
			// debit rather than failing partway through a partially-applied
			// purchase.
			if ccm.Capacity() < uint32(len(ccm.Assets())+len(memberCommodities)) {
				p.l.Debugf("Character [%d] target compartment [%s] has no room for [%d] package members. Capacity [%d], occupied [%d].", characterId, ccm.Id(), len(memberCommodities), ccm.Capacity(), len(ccm.Assets()))
				return reject("INVENTORY_FULL")
			}

			// Step 8: check and debit the BUYER's wallet by the package
			// commodity's own price.
			walP := wallet.NewProcessor(p.l, p.ctx, tx)
			w, err := walP.GetByAccountId(buyer.AccountId())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve buyer [%d] wallet for package purchase.", characterId)
				return reject("UNKNOWN_ERROR")
			}
			balance := w.Balance(currency)
			if balance < pkgCommodity.Price() {
				p.l.Debugf("Character [%d] has insufficient balance for package purchase. Cost [%d]. Balance [%d].", characterId, pkgCommodity.Price(), balance)
				return reject("NOT_ENOUGH_CASH")
			}
			w = w.Purchase(currency, pkgCommodity.Price())
			_, err = walP.Update(buf)(buyer.AccountId())(w.Credit())(w.Points())(w.Prepaid())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to debit buyer [%d] wallet for package purchase.", characterId)
				return err
			}

			// Step 9: create one asset per member, in resolution order.
			effectiveCurrency := effectivePurchaseCurrency(currency)
			astP := asset.NewProcessor(p.l, p.ctx, tx)
			assetIds := make([]uint32, 0, len(memberCommodities))
			for _, mc := range memberCommodities {
				am, cErr := astP.Create(buf)(ccm.Id(), mc.itemId, mc.serialNumber, effectiveCurrency, mc.count, 0, characterId)
				if cErr != nil {
					p.l.WithError(cErr).Errorf("Unable to create package member asset [%d] for character [%d].", mc.itemId, characterId)
					return reject("UNKNOWN_ERROR")
				}
				assetIds = append(assetIds, am.Id())
			}

			// Step 10: record the purchase against the BUYER's account for
			// the package serial number itself AND every member serial
			// number -- the client can ask about either.
			if err := purchaserecord.Record(tx, p.t.Id(), buyer.AccountId(), serialNumber); err != nil {
				p.l.WithError(err).Errorf("Unable to record package purchase for character [%d].", characterId)
				return reject("UNKNOWN_ERROR")
			}
			for _, mc := range memberCommodities {
				if err := purchaserecord.Record(tx, p.t.Id(), buyer.AccountId(), mc.serialNumber); err != nil {
					p.l.WithError(err).Errorf("Unable to record package member purchase for character [%d].", characterId)
					return reject("UNKNOWN_ERROR")
				}
			}

			p.l.Debugf("Character [%d] (%s) purchased package [%d] (item [%d]) for [%d] currency, creating [%d] assets for character [%d].", characterId, senderName, serialNumber, pkgCommodity.ItemId(), pkgCommodity.Price(), len(assetIds), effectiveRecipientCharacterId)
			return buf.Put(cashshop.EnvEventTopicStatus, cashshop2.PackagePurchasedStatusEventProvider(characterId, transactionId, ccm.Id(), assetIds, pkgCommodity.ItemId(), pkgCommodity.Price(), effectiveRecipientCharacterId, recipientName))
		})
	})
	if rejectEmit != nil {
		_ = rejectEmit()
		return nil
	}
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to complete package purchase for character [%d].", characterId)
		return txErr
	}
	return nil
}
