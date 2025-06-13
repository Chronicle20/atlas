package cashshop

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/cashshop/inventory/asset/reservation"
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
	"github.com/google/uuid"
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
	MoveFromCashInventoryAndEmit(characterId uint32, serialNumber uint64, inventoryType byte, slot int16) error
	MoveFromCashInventory(mb *message.Buffer) func(characterId uint32, serialNumber uint64, inventoryType byte, slot int16) error
	MoveFromFailedAndEmit(characterId uint32, cashItemId uint32) error
	MoveFromFailed(mb *message.Buffer) func(characterId uint32, cashItemId uint32) error
	MovedFromAndEmit(characterId uint32, cashItemId uint32, compartmentId uuid.UUID, slot int16) error
	MovedFrom(mb *message.Buffer) func(characterId uint32, cashItemId uint32, compartmentId uuid.UUID, slot int16) error
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
			im, err := p.itmP.Create(mb)(ci.ItemId())(1)(characterId)
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

func (p *ProcessorImpl) MoveFromCashInventoryAndEmit(characterId uint32, serialNumber uint64, inventoryType byte, slot int16) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.MoveFromCashInventory(buf)(characterId, serialNumber, inventoryType, slot)
	})
}

// MoveFromCashInventory moves an item from the cash inventory to a character's inventory
func (p *ProcessorImpl) MoveFromCashInventory(mb *message.Buffer) func(characterId uint32, serialNumber uint64, inventoryType byte, slot int16) error {
	return func(characterId uint32, serialNumber uint64, inventoryType byte, slot int16) error {
		p.l.Debugf("Character [%d] attempting to move item [%d] from cash inventory to inventory type [%d] slot [%d].", characterId, serialNumber, inventoryType, slot)

		// Get the character
		c, err := p.chaP.GetById(p.chaP.InventoryDecorator)(characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to get character [%d].", characterId)
			_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
			return err
		}

		// Determine the compartment type based on the character's job
		var compartmentType compartment.CompartmentType
		if job.GetType(job.Id(c.JobId())) == job.TypeExplorer {
			compartmentType = compartment.TypeExplorer
		} else if job.GetType(job.Id(c.JobId())) == job.TypeCygnus {
			compartmentType = compartment.TypeCygnus
		} else {
			compartmentType = compartment.TypeLegend
		}

		// Get the compartment
		ccm, err := p.cicP.GetByAccountIdAndType(c.AccountId(), compartmentType)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to get compartment for account [%d] and type [%s].", c.AccountId(), compartmentType)
			_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
			return err
		}

		// Find the asset in the compartment
		var targetAsset asset.Model
		found := false
		for _, a := range ccm.Assets() {
			if a.Item().CashId() == int64(serialNumber) {
				targetAsset = a
				found = true
				break
			}
		}

		if !found {
			p.l.Errorf("Asset with cash ID [%d] not found in compartment [%s].", serialNumber, ccm.Id())
			_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "ITEM_NOT_FOUND"))
			return errors.New("asset not found")
		}

		// Check if the item is already reserved
		cache := reservation.GetInstance()
		if cache.IsReserved(targetAsset.Item().Id()) {
			p.l.Errorf("Item [%d] with cash ID [%d] is already reserved.", targetAsset.Item().Id(), serialNumber)
			_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "ITEM_ALREADY_RESERVED"))
			return ErrAssetAlreadyReserved
		}

		// Reserve the item
		if !cache.Reserve(targetAsset.Item().Id(), characterId) {
			p.l.Errorf("Failed to reserve item [%d] with cash ID [%d] for character [%d].", targetAsset.Item().Id(), serialNumber, characterId)
			_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "ITEM_ALREADY_RESERVED"))
			return ErrAssetAlreadyReserved
		}
		p.l.Debugf("Reserved item [%d] with cash ID [%d] for character [%d].", targetAsset.Item().Id(), serialNumber, characterId)

		// Send a command to the inventory service to add the item to the character's inventory
		err = p.chaComP.MoveCashItemToCompartment(mb)(characterId, inventoryType, slot, targetAsset.Item().Id())
		if err != nil {
			// Release the reservation since the operation failed
			reservation.GetInstance().Release(targetAsset.Item().Id())
			p.l.WithError(err).Errorf("Unable to take ownership of cash item [%d] for character [%d].", targetAsset.Item().Id(), characterId)
			_ = mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
			return err
		}

		p.l.Debugf("Character [%d] initiating moving item [%d] from cash inventory to inventory type [%d] slot [%d].", characterId, serialNumber, inventoryType, slot)
		return nil
	}
}

func (p *ProcessorImpl) MoveFromFailedAndEmit(characterId uint32, cashItemId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.MoveFromFailed(buf)(characterId, cashItemId)
	})

}

// MoveFromFailed handles error events with CashItemMoveFailed error code
func (p *ProcessorImpl) MoveFromFailed(mb *message.Buffer) func(characterId uint32, cashItemId uint32) error {
	return func(characterId uint32, cashItemId uint32) error {
		p.l.Debugf("Handling cash item move failed error for character [%d], cash item [%d].", characterId, cashItemId)

		// Release the reservation using the cashItemId
		p.l.Debugf("Releasing asset reservation for character [%d], cash item [%d].", characterId, cashItemId)
		reservation.GetInstance().Release(cashItemId)

		// Emit a cash shop error event
		return mb.Put(cashshop.EnvEventTopicStatus, cashshop2.ErrorStatusEventProvider(characterId, "UNKNOWN_ERROR"))
	}
}

func (p *ProcessorImpl) MovedFromAndEmit(characterId uint32, cashItemId uint32, compartmentId uuid.UUID, slot int16) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.MovedFrom(buf)(characterId, cashItemId, compartmentId, slot)
	})
}

// MovedFrom handles cash item moved events
func (p *ProcessorImpl) MovedFrom(mb *message.Buffer) func(characterId uint32, cashItemId uint32, compartmentId uuid.UUID, slot int16) error {
	return func(characterId uint32, cashItemId uint32, compartmentId uuid.UUID, slot int16) error {
		p.l.Debugf("Handling cash item moved event for character [%d], cash item [%d], slot [%d].", characterId, cashItemId, slot)

		// Release the reservation using the cashItemId
		p.l.Debugf("Releasing asset reservation for character [%d], cash item [%d].", characterId, cashItemId)
		reservation.GetInstance().Release(cashItemId)

		// Remove the cash compartment - cash item association (asset entity)
		p.l.Debugf("Removing cash compartment - cash item association for character [%d], cash item [%d].", characterId, cashItemId)

		// Delete the asset entity from the database
		err := p.astP.MovedFrom(mb)(cashItemId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to remove cash compartment - cash item association for character [%d], cash item [%d].", characterId, cashItemId)
			return err
		}

		// Emit a cash shop status event for "cash shop item moved to inventory"
		return mb.Put(cashshop.EnvEventTopicStatus, cashshop2.CashItemMovedToInventoryStatusEventProvider(characterId, compartmentId, slot))
	}
}
