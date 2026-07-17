package cashshop

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/character"
	compartment2 "atlas-cashshop/character/compartment"
	inventory2 "atlas-cashshop/character/inventory"
	dataPet "atlas-cashshop/data/pet"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/cashshop"
	cashshop2 "atlas-cashshop/kafka/producer/cashshop"
	"atlas-cashshop/pet"
	"atlas-cashshop/wallet"
	"context"
	"errors"

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
)

// errPurchaseRejected is an internal sentinel used to abort the Purchase
// transaction closure on a handled rejection (e.g. inventory full) whose
// event must fire on the direct producer path rather than the outbox. It
// never escapes Purchase(): the rejectEmit != nil check short-circuits
// before txErr is inspected.
var errPurchaseRejected = errors.New("purchase rejected")

type Processor interface {
	PurchaseAndEmit(characterId uint32, currency uint32, serialNumber uint32) error
	Purchase(mb *message.Buffer) func(characterId uint32, currency uint32, serialNumber uint32) error
	PurchaseInventoryIncreaseByItemAndEmit(characterId uint32, currency uint32, serialNumber uint32) error
	PurchaseInventoryIncreaseByTypeAndEmit(characterId uint32, currency uint32, inventoryType inventory.Type) error
	PurchaseInventoryIncrease(mb *message.Buffer) func(characterId uint32, currency uint32, inventoryType inventory.Type, cost uint32, amount uint32) error
}

type ProcessorImpl struct {
	l        logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	t        tenant.Model
	chaP     character.Processor
	comP     commodity.Processor
	cicP     compartment.Processor
	chaInvP  inventory2.Processor
	chaComP  compartment2.Processor
	walP     wallet.Processor
	astP     asset.Processor
	petP     pet.Processor
	dataPetP dataPet.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:        l,
		ctx:      ctx,
		db:       db,
		t:        tenant.MustFromContext(ctx),
		chaP:     character.NewProcessor(l, ctx),
		comP:     commodity.NewProcessor(l, ctx),
		cicP:     compartment.NewProcessor(l, ctx, db),
		chaInvP:  inventory2.NewProcessor(l, ctx),
		chaComP:  compartment2.NewProcessor(l, ctx),
		walP:     wallet.NewProcessor(l, ctx, db),
		astP:     asset.NewProcessor(l, ctx, db),
		petP:     pet.NewProcessor(l, ctx),
		dataPetP: dataPet.NewProcessor(l, ctx),
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) PurchaseAndEmit(characterId uint32, currency uint32, serialNumber uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return NewProcessor(p.l, p.ctx, tx).Purchase(buf)(characterId, currency, serialNumber)
		})
	})
}

func (p *ProcessorImpl) Purchase(mb *message.Buffer) func(characterId uint32, currency uint32, serialNumber uint32) error {
	return func(characterId uint32, currency uint32, serialNumber uint32) error {
		// rejectEmit captures the INVENTORY_FULL rejection (no state change
		// committed) so it can be fired on the DIRECT producer path, outside
		// the tx closure below, instead of leaking into the outbox as if it
		// were part of the committed transaction (recipe failure-path pitfall #1).
		var rejectEmit func() error
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			ci, err := p.comP.GetById(serialNumber)
			if err != nil {
				_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
				return err
			}
			p.l.Debugf("Character [%d] attempting to purchase [%d] using currency [%d]. Cost is [%d].", characterId, serialNumber, currency, ci.Price())
			c, err := p.chaP.GetById(p.chaP.InventoryDecorator)(characterId)
			if err != nil {
				_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
				return err
			}
			w, err := p.walP.GetByAccountId(c.AccountId())
			if err != nil {
				_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
				return err
			}
			balance := w.Balance(currency)
			if balance < ci.Price() {
				p.l.Debugf("Character [%d] has insufficient balance for purchase. Cost [%d]. Balance [%d].", characterId, ci.Price(), balance)
				_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "NOT_ENOUGH_CASH"))
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
				_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
				return err
			}
			if ccm.Capacity() <= uint32(len(ccm.Assets())) {
				p.l.Debugf("Character [%d] has no room for purchase. Compartment [%s] capacity [%d].", characterId, ccm.Id(), ccm.Capacity())
				rejectEmit = func() error {
					return producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(characterId, "INVENTORY_FULL"))
				}
				return errPurchaseRejected
			}

			w = w.Purchase(currency, ci.Price())
			w, err = p.walP.WithTransaction(tx).Update(mb)(c.AccountId())(w.Credit())(w.Points())(w.Prepaid())
			if err != nil {
				return err
			}

			var petId uint32
			if item.GetClassification(item.Id(ci.ItemId())) == item.ClassificationPet {
				petData, pdErr := p.dataPetP.GetById(ci.ItemId())
				petName := "Pet"
				if pdErr == nil {
					petName = petData.Name()
				} else {
					p.l.WithError(pdErr).Warnf("Unable to retrieve pet data for template [%d], using default name.", ci.ItemId())
				}

				pe, peErr := p.petP.Create(characterId, ci.ItemId(), petName)
				if peErr != nil {
					p.l.WithError(peErr).Errorf("Unable to create pet for character [%d] template [%d].", characterId, ci.ItemId())
					_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
					return peErr
				}
				petId = pe.Id()
				p.l.Debugf("Created pet [%d] for character [%d] with name [%s].", petId, characterId, petName)
			}

			// Create the flattened asset directly (no separate item creation)
			am, err := p.astP.Create(mb)(ccm.Id(), ci.ItemId(), serialNumber, ci.Count(), petId, characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create asset for character [%d].", characterId)
				_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
				return err
			}

			p.l.Debugf("Character [%d] successfully purchased item [%d] for [%d] currency.", characterId, ci.ItemId(), ci.Price())
			_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.PurchaseStatusEventProvider(characterId, ci.ItemId(), ci.Price(), ccm.Id(), am.Id()))

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
	inventoryType := inventory.Type(ci.ItemId() - 9110000/1000)
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return NewProcessor(p.l, p.ctx, tx).PurchaseInventoryIncrease(buf)(characterId, currency, inventoryType, ci.Price(), 4)
		})
	})
}

func (p *ProcessorImpl) PurchaseInventoryIncreaseByTypeAndEmit(characterId uint32, currency uint32, inventoryType inventory.Type) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return NewProcessor(p.l, p.ctx, tx).PurchaseInventoryIncrease(buf)(characterId, currency, inventoryType, 4000, 8)
		})
	})
}

func (p *ProcessorImpl) PurchaseInventoryIncrease(mb *message.Buffer) func(characterId uint32, currency uint32, inventoryType inventory.Type, cost uint32, amount uint32) error {
	return func(characterId uint32, currency uint32, inventoryType inventory.Type, cost uint32, amount uint32) error {
		newCapacity := uint32(0)

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
			_ = producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
			return txErr
		}

		p.l.Debugf("Character [%d] purchased inventory [%d] increase. New capacity will be [%d].", characterId, inventoryType, newCapacity)
		return nil
	}
}
