package cashshop

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/cashshop/item"
	"atlas-cashshop/character"
	compartment2 "atlas-cashshop/character/compartment"
	inventory2 "atlas-cashshop/character/inventory"
	"atlas-cashshop/database"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/cashshop"
	"atlas-cashshop/kafka/producer"
	cashshop2 "atlas-cashshop/kafka/producer/cashshop"
	"atlas-cashshop/wallet"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var ErrInsufficientFunds = errors.New("insufficient funds")
var ErrMaxSlots = errors.New("max slots")
var ErrAssetAlreadyReserved = errors.New("asset already reserved")

type Processor interface {
	PurchaseAndEmit(characterId uint32, currency uint32, serialNumber uint32) error
	Purchase(mb *message.Buffer) func(characterId uint32, currency uint32, serialNumber uint32) error
	PurchaseInventoryIncreaseByItemAndEmit(characterId uint32, currency uint32, serialNumber uint32) error
	PurchaseInventoryIncreaseByTypeAndEmit(characterId uint32, currency uint32, inventoryType inventory.Type) error
	PurchaseInventoryIncrease(mb *message.Buffer) func(characterId uint32, currency uint32, inventoryType inventory.Type, cost uint32, amount uint32) error
}

type ProcessorImpl struct {
	l       logrus.FieldLogger
	ctx     context.Context
	db      *gorm.DB
	t       tenant.Model
	p       producer.Provider
	chaP    character.Processor
	comP    commodity.Processor
	cicP    compartment.Processor
	chaInvP inventory2.Processor
	chaComP compartment2.Processor
	walP    wallet.Processor
	itmP    item.Processor
	astP    asset.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:       l,
		ctx:     ctx,
		db:      db,
		t:       tenant.MustFromContext(ctx),
		p:       producer.ProviderImpl(l)(ctx),
		chaP:    character.NewProcessor(l, ctx),
		comP:    commodity.NewProcessor(l, ctx),
		cicP:    compartment.NewProcessor(l, ctx, db),
		chaInvP: inventory2.NewProcessor(l, ctx),
		chaComP: compartment2.NewProcessor(l, ctx),
		walP:    wallet.NewProcessor(l, ctx, db),
		itmP:    item.NewProcessor(l, ctx, db),
		astP:    asset.NewProcessor(l, ctx, db),
	}
	return p
}

func (p *ProcessorImpl) PurchaseAndEmit(characterId uint32, currency uint32, serialNumber uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Purchase(buf)(characterId, currency, serialNumber)
	})
}

func (p *ProcessorImpl) Purchase(mb *message.Buffer) func(characterId uint32, currency uint32, serialNumber uint32) error {
	return func(characterId uint32, currency uint32, serialNumber uint32) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {

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
			if job.GetType(job.Id(c.JobId())) == job.TypeExplorer {
				compartmentType = compartment.TypeExplorer
			} else if job.GetType(job.Id(c.JobId())) == job.TypeCygnus {
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
				_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "INVENTORY_FULL"))
				return nil
			}

			w = w.Purchase(currency, ci.Price())
			w, err = p.walP.WithTransaction(tx).Update(mb)(c.AccountId())(w.Credit())(w.Points())(w.Prepaid())
			if err != nil {
				return err
			}

			// Create the cash item
			im, err := p.itmP.Create(mb)(ci.ItemId())(ci.Count())(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create cash item for character [%d].", characterId)
				_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
				return err
			}

			// Create the asset entity in the database
			am, err := p.astP.Create(mb)(ccm.Id())(im.Id())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create asset for character [%d].", characterId)
				_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
				return err
			}

			p.l.Debugf("Character [%d] successfully purchased item [%d] for [%d] currency.", characterId, ci.ItemId(), ci.Price())
			_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.PurchaseStatusEventProvider(characterId, ci.ItemId(), ci.Price(), ccm.Id(), am.Id(), im.Id()))

			return nil
		})
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
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.PurchaseInventoryIncrease(buf)(characterId, currency, inventoryType, ci.Price(), 4)
	})
}

func (p *ProcessorImpl) PurchaseInventoryIncreaseByTypeAndEmit(characterId uint32, currency uint32, inventoryType inventory.Type) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.PurchaseInventoryIncrease(buf)(characterId, currency, inventoryType, 4000, 8)
	})
}

func (p *ProcessorImpl) PurchaseInventoryIncrease(mb *message.Buffer) func(characterId uint32, currency uint32, inventoryType inventory.Type, cost uint32, amount uint32) error {
	return func(characterId uint32, currency uint32, inventoryType inventory.Type, cost uint32, amount uint32) error {
		newCapacity := uint32(0)

		p.l.Debugf("Character [%d] attempting to purchase inventory [%d] increase using currency [%d]. Cost is [%d].", characterId, inventoryType, currency, cost)
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
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
			return nil
		})
		if txErr != nil {
			_ = producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
			return txErr
		}

		p.l.Debugf("Character [%d] purchased inventory [%d] increase. New capacity will be [%d].", characterId, inventoryType, newCapacity)
		_ = producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.InventoryCapacityIncreasedStatusEventProvider(characterId, byte(inventoryType), newCapacity, amount))
		return nil
	}
}
