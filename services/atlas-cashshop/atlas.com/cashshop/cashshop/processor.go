package cashshop

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/character"
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
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Purchase(characterId uint32, currency uint32, serialNumber uint32) error
	PurchaseInventoryIncreaseByItemAndEmit(characterId uint32, currency uint32, serialNumber uint32) error
	PurchaseInventoryIncreaseByTypeAndEmit(characterId uint32, currency uint32, inventoryType inventory.Type) error
	PurchaseInventoryIncrease(mb *message.Buffer) func(characterId uint32, currency uint32, inventoryType inventory.Type, cost uint32, amount uint32) error
}

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	db   *gorm.DB
	t    tenant.Model
	chaP character.Processor
	comP commodity.Processor
	invP inventory2.Processor
	walP wallet.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:    l,
		ctx:  ctx,
		db:   db,
		t:    tenant.MustFromContext(ctx),
		chaP: character.NewProcessor(l, ctx),
		comP: commodity.NewProcessor(l, ctx),
		invP: inventory2.NewProcessor(l, ctx),
		walP: wallet.NewProcessor(l, ctx, db),
	}
	return p
}

func (p *ProcessorImpl) Purchase(characterId uint32, currency uint32, serialNumber uint32) error {
	ci, err := p.comP.GetById(serialNumber)
	if err != nil {
		return err
	}
	p.l.Debugf("Character [%d] attempting to purchase [%d] using currency [%d]. Cost is [%d].", characterId, serialNumber, currency, ci.Price())
	_ = producer.ProviderImpl(p.l)(p.ctx)(cashshop.EnvEventTopicStatus)(cashshop2.ErrorStatusEventProvider(characterId, "INVENTORY_FULL"))
	return nil
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

		ErrInsufficientFunds := errors.New("insufficient funds")
		ErrMaxSlots := errors.New("max slots")
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
			credit := w.Credit()
			points := w.Points()
			prepaid := w.Prepaid()
			balance := uint32(0)
			if currency == 1 {
				balance = w.Credit()
				credit -= cost
			} else if currency == 2 {
				balance = w.Points()
				points -= cost
			} else {
				balance = w.Prepaid()
				points -= cost
			}
			if balance < cost {
				return ErrInsufficientFunds
			}

			slots := c.Inventory().CompartmentByType(inventoryType).Capacity()
			if slots+amount > 96 {
				return ErrMaxSlots
			}
			newCapacity = slots + amount

			_, err = p.walP.WithTransaction(tx).Update(c.AccountId(), credit, points, prepaid)
			if err != nil {
				return err
			}
			err = p.invP.IncreaseCapacity(mb)(characterId, inventoryType, amount)
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
