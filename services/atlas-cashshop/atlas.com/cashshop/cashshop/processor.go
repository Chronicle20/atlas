package cashshop

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/character"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/pet"
	"atlas-cashshop/purchaserecord"
	"atlas-cashshop/wallet"
	"context"
	"errors"

	compartment2 "atlas-cashshop/character/compartment"
	inventory2 "atlas-cashshop/character/inventory"
	dataCashPkg "atlas-cashshop/data/cashpackage"
	dataPet "atlas-cashshop/data/pet"

	cashshop2 "atlas-cashshop/kafka/producer/cashshop"

	"github.com/google/uuid"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var (
	ErrInsufficientFunds    = errors.New("insufficient funds")
	ErrMaxSlots             = errors.New("max slots")
	ErrAssetAlreadyReserved = errors.New("asset already reserved")
	ErrInvalidInventoryType = errors.New("invalid inventory type")
)

// errPurchaseRejected is an internal sentinel used to abort the Purchase
// transaction closure on a handled rejection (e.g. inventory full) whose
// event must fire on the direct producer path rather than the outbox. It
// never escapes Purchase(): the rejectEmit != nil check short-circuits
// before txErr is inspected.
var errPurchaseRejected = errors.New("purchase rejected")

// Wallet bucket ids, matching wallet.Model's Balance/Purchase/Award dispatch
// (wallet/model.go): 1 routes to credit (NX), 2 to Maple Points, and EVERY
// OTHER value -- including 0 -- routes to prepaid. No entry in
// libs/atlas-constants covers wallet currency buckets, so these are defined
// here, next to the one place that normalizes a raw wire currency into a
// value worth persisting.
const (
	walletCurrencyCredit  uint32 = 1
	walletCurrencyPoints  uint32 = 2
	walletCurrencyPrepaid uint32 = 3
)

// effectivePurchaseCurrency maps a raw wire currency (what the client sent,
// and what w.Purchase/w.Balance dispatch on for the DEBIT) to the value
// worth PERSISTING on the created asset row for a later rebate to read back.
//
// This is the fix for a real bug found in review: wallet.Model routes any
// currency other than 1 or 2 to prepaid, so a BUY_NORMAL purchase (wire
// currency 0, atlas-channel's resolvePurchaseCurrency -- see
// services/atlas-channel/atlas.com/channel/cashshop/processor.go:118 --
// deliberately produces isPoints=false/currency=0 for exactly this arm,
// design.md:18) debits prepaid. But persisting the raw 0 on the asset makes
// it indistinguishable from a legacy row that predates the Currency column
// at all (asset.Entity.Currency's 0-means-legacy convention) -- so a rebate
// reading a raw-0 asset cannot tell "this was a prepaid purchase" from
// "nothing was ever recorded," and defaulted to credit, crediting the WRONG
// bucket (a free currency conversion on the most common buy path).
//
// The fix normalizes at this write site instead of guessing at the read
// site: 1 and 2 persist unchanged (they are already unambiguous), and every
// other raw value -- including 0 -- persists as walletCurrencyPrepaid (3),
// which is not 0 and therefore cannot collide with the legacy convention.
// wallet.Model's dispatch is untouched: 0 and 3 already hit the identical
// "everything else" prepaid arm, so the DEBIT (which keeps using the raw
// wire value) behaves exactly as before.
func effectivePurchaseCurrency(currency uint32) uint32 {
	switch currency {
	case walletCurrencyCredit, walletCurrencyPoints:
		return currency
	default:
		return walletCurrencyPrepaid
	}
}

// isValidInventoryType reports whether t is one of the known inventory
// compartment types (inventory.Types). Used to guard against a computed
// (rather than wire-provided) inventory type that does not correspond to a
// real compartment -- see PurchaseInventoryIncreaseByItemAndEmit.
func isValidInventoryType(t inventory.Type) bool {
	for _, v := range inventory.Types {
		if v == t {
			return true
		}
	}
	return false
}

type Processor interface {
	PurchaseAndEmit(characterId uint32, currency uint32, serialNumber uint32, transactionId uuid.UUID, operation string) error
	Purchase(mb *message.Buffer) func(characterId uint32, currency uint32, serialNumber uint32, transactionId uuid.UUID, operation string) error
	PurchaseInventoryIncreaseByItemAndEmit(characterId uint32, currency uint32, serialNumber uint32) error
	PurchaseInventoryIncreaseByTypeAndEmit(characterId uint32, currency uint32, inventoryType inventory.Type) error
	PurchaseInventoryIncrease(mb *message.Buffer) func(characterId uint32, currency uint32, inventoryType inventory.Type, cost uint32, amount uint32) error
	RebateAndEmit(characterId uint32, accountId uint32, cashId int64, transactionId uuid.UUID) error
	GiftAndEmit(characterId uint32, transactionId uuid.UUID, serialNumber uint32, recipientCharacterId uint32, senderName string, giftMessage string) error
	PurchasePackageAndEmit(characterId uint32, transactionId uuid.UUID, currency uint32, serialNumber uint32, recipientCharacterId uint32, senderName string) error
	PurchaseRingAndEmit(characterId uint32, transactionId uuid.UUID, currency uint32, serialNumber uint32, partnerCharacterId uint32, senderName string, message string, ringType string) error
	PurchaseEquipSlotAndEmit(characterId uint32, currency uint32, serialNumber uint32, transactionId uuid.UUID) error
	CompleteEquipSlotExtension(characterId uint32, slotIndex int16, days uint16, transactionId uuid.UUID) error
	AcknowledgeGiftsAndEmit(accountId uint32, cashIds []int64) error
}

type ProcessorImpl struct {
	l            logrus.FieldLogger
	ctx          context.Context
	db           *gorm.DB
	t            tenant.Model
	chaP         character.Processor
	comP         commodity.Processor
	cicP         compartment.Processor
	chaInvP      inventory2.Processor
	chaComP      compartment2.Processor
	walP         wallet.Processor
	astP         asset.Processor
	petP         pet.Processor
	dataPetP     dataPet.Processor
	dataCashPkgP dataCashPkg.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:            l,
		ctx:          ctx,
		db:           db,
		t:            tenant.MustFromContext(ctx),
		chaP:         character.NewProcessor(l, ctx),
		comP:         commodity.NewProcessor(l, ctx),
		cicP:         compartment.NewProcessor(l, ctx, db),
		chaInvP:      inventory2.NewProcessor(l, ctx),
		chaComP:      compartment2.NewProcessor(l, ctx),
		walP:         wallet.NewProcessor(l, ctx, db),
		astP:         asset.NewProcessor(l, ctx, db),
		petP:         pet.NewProcessor(l, ctx),
		dataPetP:     dataPet.NewProcessor(l, ctx),
		dataCashPkgP: dataCashPkg.NewProcessor(l, ctx),
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) PurchaseAndEmit(characterId uint32, currency uint32, serialNumber uint32, transactionId uuid.UUID, operation string) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return NewProcessor(p.l, p.ctx, tx).Purchase(buf)(characterId, currency, serialNumber, transactionId, operation)
		})
	})
}

func (p *ProcessorImpl) Purchase(mb *message.Buffer) func(characterId uint32, currency uint32, serialNumber uint32, transactionId uuid.UUID, operation string) error {
	return func(characterId uint32, currency uint32, serialNumber uint32, transactionId uuid.UUID, operation string) error {
		// rejectEmit captures every purchase-path rejection/error (no state
		// change committed on this branch) so it can be fired on the DIRECT
		// producer path, outside the tx closure below, instead of being
		// buffered on mb and silently dropped: message.Emit only flushes its
		// buffer when the wrapped closure returns nil
		// (kafka/message/message.go:49-52), and PurchaseAndEmit's closure
		// returns whatever error this tx produces -- so an mb.Put'd error
		// event on a failing purchase never reached the outbox (recipe
		// failure-path pitfall #1).
		var rejectEmit func() error
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			ci, err := p.comP.GetById(serialNumber)
			if err != nil {
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, "UNKNOWN_ERROR", transactionId))
				}
				return err
			}
			p.l.Debugf("Character [%d] attempting to purchase [%d] using currency [%d]. Cost is [%d].", characterId, serialNumber, currency, ci.Price())
			c, err := p.chaP.GetById(p.chaP.InventoryDecorator)(characterId)
			if err != nil {
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, "UNKNOWN_ERROR", transactionId))
				}
				return err
			}
			w, err := p.walP.GetByAccountId(c.AccountId())
			if err != nil {
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, "UNKNOWN_ERROR", transactionId))
				}
				return err
			}
			balance := w.Balance(currency)
			if balance < ci.Price() {
				p.l.Debugf("Character [%d] has insufficient balance for purchase. Cost [%d]. Balance [%d].", characterId, ci.Price(), balance)
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, "NOT_ENOUGH_CASH", transactionId))
				}
				return ErrInsufficientFunds
			}

			var compartmentType compartment.CompartmentType
			if job.GetType(c.JobId()) == job.TypeExplorer {
				compartmentType = compartment.TypeExplorer
			} else if job.GetType(c.JobId()) == job.TypeCygnus {
				compartmentType = compartment.TypeCygnus
			} else {
				compartmentType = compartment.TypeLegend
			}

			ccm, err := p.cicP.GetByAccountIdAndType(c.AccountId(), compartmentType)
			if err != nil {
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, "UNKNOWN_ERROR", transactionId))
				}
				return err
			}
			if ccm.Capacity() <= uint32(len(ccm.Assets())) {
				p.l.Debugf("Character [%d] has no room for purchase. Compartment [%s] capacity [%d].", characterId, ccm.Id(), ccm.Capacity())
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, "INVENTORY_FULL", transactionId))
				}
				return errPurchaseRejected
			}

			w = w.Purchase(currency, ci.Price())
			w, err = p.walP.WithTransaction(tx).Update(mb)(c.AccountId())(w.Credit())(w.Points())(w.Prepaid())
			if err != nil {
				return err
			}

			var petId uint32
			// petCashId is reserved before either row is written so the pet and
			// the cash asset share one serial. The client keys that single value
			// (GW_ItemSlotBase::liCashItemSN) for BOTH locker removal on withdraw
			// and spawned-pet-to-inventory binding, so a pet whose two rows
			// disagree gets stuck in the cash-shop locker UI forever.
			var petCashId int64
			if item.GetClassification(item.Id(ci.ItemId())) == item.ClassificationPet {
				petData, pdErr := p.dataPetP.GetById(ci.ItemId())
				petName := "Pet"
				if pdErr == nil {
					petName = petData.Name()
				} else {
					p.l.WithError(pdErr).Warnf("Unable to retrieve pet data for template [%d], using default name.", ci.ItemId())
				}

				petCashId, err = p.astP.NextCashId()
				if err != nil {
					p.l.WithError(err).Errorf("Unable to reserve a cash serial for character [%d] template [%d].", characterId, ci.ItemId())
					rejectEmit = func() error {
						return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, "UNKNOWN_ERROR", transactionId))
					}
					return err
				}

				pe, peErr := p.petP.Create(characterId, uint64(petCashId), ci.ItemId(), petName)
				if peErr != nil {
					p.l.WithError(peErr).Errorf("Unable to create pet for character [%d] template [%d].", characterId, ci.ItemId())
					rejectEmit = func() error {
						return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, "UNKNOWN_ERROR", transactionId))
					}
					return peErr
				}
				petId = pe.Id()
				p.l.Debugf("Created pet [%d] (cash serial [%d]) for character [%d] with name [%s].", petId, petCashId, characterId, petName)
			}

			// Create the flattened asset directly (no separate item creation).
			// Pets must carry the serial reserved above; everything else gets a
			// freshly generated one.
			// effectiveCurrency (NOT the raw wire currency) is recorded on the
			// asset row so a later locker rebate (task-240 task 11) knows which
			// bucket to credit back instead of guessing -- see
			// effectivePurchaseCurrency's doc comment for why the raw value
			// must not be persisted as-is.
			effectiveCurrency := effectivePurchaseCurrency(currency)
			var am asset.Model
			if petCashId != 0 {
				am, err = p.astP.CreateWithCashId(mb)(ccm.Id(), petCashId, ci.ItemId(), serialNumber, effectiveCurrency, ci.Count(), petId, characterId)
			} else {
				am, err = p.astP.Create(mb)(ccm.Id(), ci.ItemId(), serialNumber, effectiveCurrency, ci.Count(), petId, characterId)
			}
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create asset for character [%d].", characterId)
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, "UNKNOWN_ERROR", transactionId))
				}
				return err
			}

			if err = purchaserecord.Record(tx, p.t.Id(), c.AccountId(), serialNumber); err != nil {
				p.l.WithError(err).Errorf("Unable to record purchase for character [%d].", characterId)
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventForOperationProvider(characterId, operation, "UNKNOWN_ERROR", transactionId))
				}
				return err
			}

			p.l.Debugf("Character [%d] successfully purchased item [%d] for [%d] currency.", characterId, ci.ItemId(), ci.Price())
			_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.PurchaseStatusEventProvider(characterId, ci.ItemId(), ci.Price(), ccm.Id(), am.Id(), transactionId, operation))

			return nil
		})
		if rejectEmit != nil {
			_ = rejectEmit()
			return nil
		}
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to complete purchase for character [%d].", characterId)
			return txErr
		}
		return nil
	}
}

func (p *ProcessorImpl) PurchaseInventoryIncreaseByItemAndEmit(characterId uint32, currency uint32, serialNumber uint32) error {
	ci, err := p.comP.GetById(serialNumber)
	if err != nil {
		return err
	}
	// ItemId() and 9110000 are both uint32; if ItemId() < 9110000 the
	// subtraction below underflows before truncation to inventory.Type
	// (int8), and the truncated byte could coincidentally land in a valid
	// range, silently defeating isValidInventoryType. Reject here, before
	// the subtraction can wrap, rather than relying solely on the post-hoc
	// range check. Only reachable via the server's own commodity table, not
	// client input.
	if ci.ItemId() < 9110000 {
		p.l.Errorf("Character [%d] attempted to purchase inventory increase for commodity item [%d] below the inventory-type base offset. Rejecting without debiting wallet.", characterId, ci.ItemId())
		_ = producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR", uuid.Nil))
		return ErrInvalidInventoryType
	}
	inventoryType := inventory.Type((ci.ItemId() - 9110000) / 1000)
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return NewProcessor(p.l, p.ctx, tx).PurchaseInventoryIncrease(buf)(characterId, currency, inventoryType, ci.Price(), 4)
		})
	})
}

func (p *ProcessorImpl) PurchaseInventoryIncreaseByTypeAndEmit(characterId uint32, currency uint32, inventoryType inventory.Type) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return NewProcessor(p.l, p.ctx, tx).PurchaseInventoryIncrease(buf)(characterId, currency, inventoryType, 4000, 4)
		})
	})
}

func (p *ProcessorImpl) PurchaseInventoryIncrease(mb *message.Buffer) func(characterId uint32, currency uint32, inventoryType inventory.Type, cost uint32, amount uint32) error {
	return func(characterId uint32, currency uint32, inventoryType inventory.Type, cost uint32, amount uint32) error {
		newCapacity := uint32(0)

		if !isValidInventoryType(inventoryType) {
			p.l.Errorf("Character [%d] attempted to purchase inventory increase for invalid inventory type [%d]. Rejecting without debiting wallet.", characterId, inventoryType)
			_ = producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR", uuid.Nil))
			return ErrInvalidInventoryType
		}

		p.l.Debugf("Character [%d] attempting to purchase inventory [%d] increase using currency [%d]. Cost is [%d].", characterId, inventoryType, currency, cost)
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.chaP.GetById(p.chaP.InventoryDecorator)(characterId)
			if err != nil {
				return err
			}

			w, err := p.walP.WithTransaction(tx).GetByAccountId(c.AccountId())
			if err != nil {
				return err
			}

			balance := w.Balance(currency)
			w = w.Purchase(currency, cost)

			if balance < cost {
				return ErrInsufficientFunds
			}

			slots := c.Inventory().CompartmentByType(inventoryType).Capacity()
			if slots+amount > 96 {
				return ErrMaxSlots
			}
			newCapacity = slots + amount

			w, err = p.walP.WithTransaction(tx).Update(mb)(c.AccountId())(w.Credit())(w.Points())(w.Prepaid())
			if err != nil {
				return err
			}
			err = p.chaComP.IncreaseCapacity(mb)(characterId, inventoryType, amount)
			if err != nil {
				return err
			}

			// InventoryCapacityIncreasedStatusEventProvider asserts a
			// committed state change (capacity was increased in this same
			// tx), so per D7 it is enqueued through mb inside the tx rather
			// than fired directly after the fact.
			return mb.Put(cashshop.EnvEventTopicStatus, cashshop2.InventoryCapacityIncreasedStatusEventProvider(characterId, byte(inventoryType), newCapacity, amount))
		})
		if txErr != nil {
			// UNKNOWN_ERROR reflects no committed state change (the tx
			// above rolled back / never wrote), so it stays on the direct
			// producer path outside the tx rather than the outbox.
			_ = producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR", uuid.Nil))
			return txErr
		}

		p.l.Debugf("Character [%d] purchased inventory [%d] increase. New capacity will be [%d].", characterId, inventoryType, newCapacity)
		return nil
	}
}
