package cashshop

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/cashshop"
	cashshop2 "atlas-cashshop/kafka/producer/cashshop"
	"atlas-cashshop/ledger"
	"atlas-cashshop/wallet"
	"errors"
	"time"

	"github.com/google/uuid"

	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// errRebateRejected is the internal sentinel used to abort the RebateAndEmit
// transaction closure on a handled rejection whose event must fire on the
// DIRECT producer path rather than the outbox -- mirrors errPurchaseRejected
// in cashshop/processor.go, and for the same reason: message.Emit only
// flushes its buffer when the wrapped closure returns nil, so a rejection
// event enqueued through mb on a failing branch would be silently dropped.
var errRebateRejected = errors.New("rebate rejected")

// reasonRebateUnknownError is the sole rejection reason this operation emits.
// Every failure mode here (redelivery infra error, asset not found, asset
// owned by another account, expired asset, no commodity id) is reported
// identically to the caller -- an asset in another account's compartment is
// structurally indistinguishable from a missing one (the scan is
// account-scoped), so there is no richer reason to report without leaking
// which case applied.
const reasonRebateUnknownError = "unknown_error"

// RebateAndEmit implements REQUEST_LOCKER_REBATE (task-240 task 11): refunds
// one locker asset's purchase price and removes it, atomically and
// idempotently.
//
// Currency resolution (controller corrections C1/C2/C3): the wallet bucket
// credited is the refunded asset's OWN asset.Model.Currency() -- the bucket
// Purchase recorded on the asset row when it was bought
// (cashshop/processor.go's Purchase, cashshop/inventory/asset.Entity.Currency).
// Currency == 0 is not "unknown"; it is the explicit default-bucket
// convention that covers both (a) every asset that predates the column and
// (b) every asset created on a gift/reward/surprise path that was never
// bought with currency at all -- both resolve to the ordinary credit/NX
// bucket (currency id 1 in wallet.Model's Balance/Award convention), the
// same bucket Purchase's own default arm uses.
func (p *ProcessorImpl) RebateAndEmit(characterId uint32, accountId uint32, cashId int64, transactionId uuid.UUID) error {
	var rejectEmit func() error
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			reject := func() error {
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, cashshop.ErrorOperationRebate, reasonRebateUnknownError, transactionId))
				}
				return errRebateRejected
			}

			// Step 1: claim the transaction id FIRST, before any read or
			// write, so a Kafka redelivery aborts before touching state.
			// ErrAlreadyProcessed is success-without-effect (the original
			// rebate already told the client) -- no event, no error.
			if err := ledger.Claim(p.ctx, tx, transactionId, cashshop.CommandTypeRequestLockerRebate, characterId); err != nil {
				if errors.Is(err, ledger.ErrAlreadyProcessed) {
					return nil
				}
				p.l.WithError(err).Errorf("Unable to claim rebate transaction [%s] for character [%d].", transactionId, characterId)
				return reject()
			}

			// Step 2: resolve the asset by CashId within the requesting
			// account's compartments only. An asset owned by another account
			// is simply absent from this account-scoped scan, so it reports
			// the same rejection as a missing one -- there is no separate
			// NOT_OWNED outcome (mirrors surprise.Open's box resolution).
			cicP := compartment.NewProcessor(p.l, p.ctx, tx)
			ccms, err := cicP.GetByAccountId(accountId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve compartments for account [%d].", accountId)
				return reject()
			}

			var am *asset.Model
			for i := range ccms {
				assets := ccms[i].Assets()
				for j := range assets {
					if assets[j].CashId() == cashId {
						found := assets[j]
						am = &found
						break
					}
				}
				if am != nil {
					break
				}
			}
			if am == nil {
				p.l.Warnf("Rebate requested for cashId [%d] on account [%d], but no such asset is in this account's compartments.", cashId, accountId)
				return reject()
			}

			// Step 3: reject a gift/reward/surprise asset (never bought with
			// currency, FR-REB-4), an expired asset, or one whose commodity
			// no longer resolves.
			if am.CommodityId() == 0 {
				p.l.Warnf("Rebate requested for cashId [%d], but asset [%d] carries no commodity id (never bought with currency).", cashId, am.Id())
				return reject()
			}
			if am.Expiration().Before(time.Now()) {
				p.l.Warnf("Rebate requested for cashId [%d], but asset [%d] is expired.", cashId, am.Id())
				return reject()
			}
			ci, err := commodity.NewProcessor(p.l, p.ctx).GetById(am.CommodityId())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve commodity [%d] for cashId [%d].", am.CommodityId(), cashId)
				return reject()
			}

			// Step 4: delete the asset.
			astP := asset.NewProcessor(p.l, p.ctx, tx)
			if err := astP.Delete(buf)(am.Id()); err != nil {
				p.l.WithError(err).Errorf("Unable to delete asset [%d] for rebate.", am.Id())
				return err
			}

			// Step 5: credit the wallet on the currency the asset was
			// purchased with -- see this file's package-level doc comment
			// for the 0-means-default-bucket convention.
			currency := am.Currency()
			if currency == 0 {
				currency = 1
			}
			walP := wallet.NewProcessor(p.l, p.ctx, tx)
			w, err := walP.GetByAccountId(accountId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to resolve wallet for account [%d].", accountId)
				return err
			}
			w = w.Award(currency, ci.Price())
			_, err = walP.Update(buf)(accountId)(w.Credit())(w.Points())(w.Prepaid())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to credit wallet for account [%d].", accountId)
				return err
			}

			// Step 6: LOCKER_REBATED.
			p.l.Debugf("Character [%d] rebated locker asset [%d] (cashId [%d]) for [%d] on currency [%d].", characterId, am.Id(), cashId, ci.Price(), currency)
			return buf.Put(cashshop.EnvEventTopicStatus, cashshop2.LockerRebatedStatusEventProvider(characterId, cashId, int32(ci.Price()), currency, transactionId))
		})
	})
	if rejectEmit != nil {
		_ = rejectEmit()
		return nil
	}
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to complete locker rebate for character [%d].", characterId)
		return txErr
	}
	return nil
}
