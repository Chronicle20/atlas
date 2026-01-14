package shops

import (
	"atlas-npc/character"
	"atlas-npc/character/skill"
	"atlas-npc/commodities"
	"atlas-npc/compartment"
	"atlas-npc/data/consumable"
	"atlas-npc/data/equipable"
	"atlas-npc/data/etc"
	"atlas-npc/data/setup"
	"atlas-npc/database"
	inventory2 "atlas-npc/inventory"
	"atlas-npc/kafka/message"
	"atlas-npc/kafka/message/shops"
	"atlas-npc/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	skill2 "github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"math"
)

type Processor interface {
	CommodityDecorator(m Model) Model
	RechargeableConsumablesDecorator(m Model) Model
	GetByNpcId(decorators ...model.Decorator[Model]) func(npcId uint32) (Model, error)
	ByNpcIdProvider(decorators ...model.Decorator[Model]) func(npcId uint32) model.Provider[Model]
	GetAllShops(decorators ...model.Decorator[Model]) ([]Model, error)
	AllShopsProvider(decorators ...model.Decorator[Model]) model.Provider[[]Model]
	CreateShop(npcId uint32, recharger bool, commodities []commodities.Model) (Model, error)
	UpdateShop(npcId uint32, recharger bool, commodities []commodities.Model) (Model, error)
	AddCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error)
	UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error)
	RemoveCommodity(id uuid.UUID) error
	DeleteAllCommoditiesByNpcId(npcId uint32) error
	DeleteAllShops() error
	EnterAndEmit(characterId uint32, npcId uint32) error
	Enter(mb *message.Buffer) func(characterId uint32) func(npcId uint32) error
	ExitAndEmit(characterId uint32) error
	Exit(mb *message.Buffer) func(characterId uint32) error
	BuyAndEmit(characterId uint32, slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) error
	Buy(mb *message.Buffer) func(characterId uint32) func(slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) error
	SellAndEmit(characterId uint32, slot int16, itemTemplateId uint32, quantity uint32) error
	Sell(mb *message.Buffer) func(characterId uint32) func(slot int16, itemTemplateId uint32, quantity uint32) error
	RechargeAndEmit(characterId uint32, slot uint16) error
	Recharge(mb *message.Buffer) func(characterId uint32) func(slot uint16) error
	GetCharactersInShop(shopId uint32) []uint32
}

var ErrNotFound = errors.New("not found")

type ProcessorImpl struct {
	l                                  logrus.FieldLogger
	ctx                                context.Context
	db                                 *gorm.DB
	t                                  tenant.Model
	GetByNpcIdFn                       func(decorators ...model.Decorator[Model]) func(npcId uint32) (Model, error)
	GetAllShopsFn                      func(decorators ...model.Decorator[Model]) ([]Model, error)
	RechargeableConsumablesDecoratorFn func(m Model) Model
	cp                                 commodities.Processor
	charP                              character.Processor
	compP                              compartment.Processor
	invP                               inventory2.Processor
	kp                                 producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		db:    db,
		t:     tenant.MustFromContext(ctx),
		cp:    commodities.NewProcessor(l, ctx, db),
		charP: character.NewProcessor(l, ctx),
		compP: compartment.NewProcessor(l, ctx),
		invP:  inventory2.NewProcessor(l, ctx),
		kp:    producer.ProviderImpl(l)(ctx),
	}
	return p
}

func (p *ProcessorImpl) CommodityDecorator(m Model) Model {
	cms, err := p.cp.GetByNpcId(m.NpcId())
	if err != nil {
		return m
	}
	result, err := Clone(m).SetCommodities(cms).Build()
	if err != nil {
		return m
	}
	return result
}

func (p *ProcessorImpl) GetByNpcId(decorators ...model.Decorator[Model]) func(npcId uint32) (Model, error) {
	return func(npcId uint32) (Model, error) {
		if p.GetByNpcIdFn != nil {
			return p.GetByNpcIdFn(decorators...)(npcId)
		}
		return p.ByNpcIdProvider(decorators...)(npcId)()
	}
}

func (p *ProcessorImpl) ByNpcIdProvider(decorators ...model.Decorator[Model]) func(npcId uint32) model.Provider[Model] {
	return func(npcId uint32) model.Provider[Model] {
		// First check if the shop entity exists
		shopExists, err := existsByNpcId(p.t.Id(), npcId)(p.db)()
		if err != nil {
			return model.ErrorProvider[Model](err)
		}

		// If the shop entity doesn't exist, check if commodities exist
		if !shopExists {
			return model.ErrorProvider[Model](ErrNotFound)
		}

		// Get the shop entity
		shopEntity, err := getByNpcId(p.t.Id(), npcId)(p.db)()
		if err != nil {
			return model.ErrorProvider[Model](err)
		}

		// Convert entity to model
		m, err := Make(shopEntity)
		if err != nil {
			return model.ErrorProvider[Model](err)
		}

		return model.Map(model.Decorate(append(decorators, p.RechargeableConsumablesDecorator)))(model.FixedProvider(m))
	}
}

func (p *ProcessorImpl) AddCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error) {
	return p.cp.CreateCommodity(npcId, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
}

func (p *ProcessorImpl) UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error) {
	return p.cp.UpdateCommodity(id, templateId, mesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
}

func (p *ProcessorImpl) RemoveCommodity(id uuid.UUID) error {
	return p.cp.DeleteCommodity(id)
}

func (p *ProcessorImpl) CreateShop(npcId uint32, recharger bool, commodities []commodities.Model) (Model, error) {
	shopEntity, err := createShop(p.t.Id(), npcId, recharger)(p.db)()
	if err != nil {
		return Model{}, err
	}

	// For each commodity, create it in the database
	for _, commodity := range commodities {
		_, err := p.cp.CreateCommodity(
			npcId,
			commodity.TemplateId(),
			commodity.MesoPrice(),
			commodity.DiscountRate(),
			commodity.TokenTemplateId(),
			commodity.TokenPrice(),
			commodity.Period(),
			commodity.LevelLimit(),
		)
		if err != nil {
			return Model{}, err
		}
	}

	// Convert entity to model and add commodities
	shop, err := Make(shopEntity)
	if err != nil {
		return Model{}, err
	}
	return Clone(shop).SetCommodities(commodities).Build()
}

func (p *ProcessorImpl) UpdateShop(npcId uint32, recharger bool, commodities []commodities.Model) (Model, error) {
	p.l.Debugf("Updating shop for NPC [%d] with [%d] commodities.", npcId, len(commodities))

	var shop Model
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		// Update or create the shop entity with the provided recharger value
		var shopEntity Entity
		var err error
		shopEntity, err = updateShop(p.t.Id(), npcId, recharger)(tx)()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to update/create shop entity for NPC [%d].", npcId)
			return err
		}
		p.l.Debugf("Updated/created shop entity for NPC [%d] with recharger=[%t].", npcId, recharger)

		// Get existing commodities for the NPC ID
		existingCommodities, err := p.cp.WithTransaction(tx).GetByNpcId(npcId)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to retrieve existing commodities for NPC [%d].", npcId)
			return err
		}
		p.l.Debugf("Found [%d] existing commodities for NPC [%d].", len(existingCommodities), npcId)

		// Delete all existing commodities
		for i, commodity := range existingCommodities {
			p.l.Debugf("Deleting commodity [%d/%d] with ID [%s] for NPC [%d].", i+1, len(existingCommodities), commodity.Id(), npcId)
			err = p.cp.WithTransaction(tx).DeleteCommodity(commodity.Id())
			if err != nil {
				p.l.WithError(err).Errorf("Failed to delete commodity [%s] for NPC [%d].", commodity.Id(), npcId)
				return err
			}
		}
		p.l.Debugf("Successfully deleted all [%d] existing commodities for NPC [%d].", len(existingCommodities), npcId)

		// For each commodity, create it in the database
		for i, commodity := range commodities {
			p.l.Debugf("Creating commodity [%d/%d] with template ID [%d] for NPC [%d].", i+1, len(commodities), commodity.TemplateId(), npcId)
			_, err = p.cp.WithTransaction(tx).CreateCommodity(
				npcId,
				commodity.TemplateId(),
				commodity.MesoPrice(),
				commodity.DiscountRate(),
				commodity.TokenTemplateId(),
				commodity.TokenPrice(),
				commodity.Period(),
				commodity.LevelLimit(),
			)
			if err != nil {
				p.l.WithError(err).Errorf("Failed to create commodity with template ID [%d] for NPC [%d].", commodity.TemplateId(), npcId)
				return err
			}
		}
		p.l.Debugf("Successfully created all [%d] new commodities for NPC [%d].", len(commodities), npcId)

		// Convert entity to model and add commodities
		shopModel, err := Make(shopEntity)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to convert shop entity to model for NPC [%d].", npcId)
			return err
		}
		shop, err = Clone(shopModel).SetCommodities(commodities).Build()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to build shop model for NPC [%d].", npcId)
			return err
		}
		p.l.Debugf("Created shop model for NPC [%d].", npcId)

		return nil
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Transaction failed while updating shop for NPC [%d].", npcId)
		return Model{}, txErr
	}
	p.l.Debugf("Successfully updated shop for NPC [%d] with [%d] commodities.", npcId, len(commodities))
	return shop, nil
}

func (p *ProcessorImpl) EnterAndEmit(characterId uint32, npcId uint32) error {
	return message.Emit(p.kp)(model.Flip(model.Flip(p.Enter)(characterId))(npcId))
}

func (p *ProcessorImpl) Enter(mb *message.Buffer) func(characterId uint32) func(npcId uint32) error {
	return func(characterId uint32) func(npcId uint32) error {
		return func(npcId uint32) error {
			p.l.Debugf("Character [%d] attempting to enter shop [%d].", characterId, npcId)
			_, err := p.GetByNpcId(p.CommodityDecorator)(npcId)
			if err != nil {
				p.l.WithError(err).Errorf("Cannot locate shop [%d] character [%d] is attempting to enter.", npcId, characterId)
				return err
			}
			getRegistry().AddCharacter(p.t.Id(), characterId, npcId)
			return mb.Put(shops.EnvStatusEventTopic, enteredEventProvider(characterId, npcId))
		}
	}
}

func (p *ProcessorImpl) ExitAndEmit(characterId uint32) error {
	return message.Emit(p.kp)(model.Flip(p.Exit)(characterId))
}

func (p *ProcessorImpl) Exit(mb *message.Buffer) func(characterId uint32) error {
	return func(characterId uint32) error {
		p.l.Debugf("Character [%d] attempting to exit shop.", characterId)
		_, inShop := getRegistry().GetShop(p.t.Id(), characterId)
		getRegistry().RemoveCharacter(p.t.Id(), characterId)
		if inShop {
			return mb.Put(shops.EnvStatusEventTopic, exitedEventProvider(characterId))
		}
		return nil
	}
}

func (p *ProcessorImpl) GetCharactersInShop(shopId uint32) []uint32 {
	return getRegistry().GetCharactersInShop(p.t.Id(), shopId)
}

func (p *ProcessorImpl) DeleteAllCommoditiesByNpcId(npcId uint32) error {
	commoditiesProcessor := commodities.NewProcessor(p.l, p.ctx, p.db)
	return commoditiesProcessor.DeleteAllCommoditiesByNpcId(npcId)
}

func (p *ProcessorImpl) DeleteAllShops() error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		_, err := deleteAllShops(p.t.Id())(tx)()
		if err != nil {
			return err
		}
		return p.cp.WithTransaction(tx).DeleteAllCommodities()
	})

}

func (p *ProcessorImpl) GetAllShops(decorators ...model.Decorator[Model]) ([]Model, error) {
	if p.GetAllShopsFn != nil {
		return p.GetAllShopsFn(decorators...)
	}
	return p.AllShopsProvider(decorators...)()
}

func (p *ProcessorImpl) AllShopsProvider(decorators ...model.Decorator[Model]) model.Provider[[]Model] {
	// Get all shop entities
	shopEntities, err := getAllShops(p.t.Id())(p.db)()
	if err != nil {
		// If there's an error getting shop entities, fall back to getting NPC IDs from commodities
		npcIds, err := p.cp.GetDistinctNpcIds()
		if err != nil {
			return model.FixedProvider[[]Model](make([]Model, 0))
		}

		// Create shop entities for each NPC ID if they don't exist
		for _, npcId := range npcIds {
			shopExists, err := existsByNpcId(p.t.Id(), npcId)(p.db)()
			if err != nil {
				continue
			}
			if !shopExists {
				_, err = createShop(p.t.Id(), npcId, true)(p.db)()
				if err != nil {
					continue
				}
			}
		}

		// Try to get all shop entities again
		shopEntities, err = getAllShops(p.t.Id())(p.db)()
		if err != nil {
			return model.FixedProvider[[]Model](make([]Model, 0))
		}
	}

	// Convert entities to models
	sbp := model.SliceMap(func(entity Entity) (Model, error) {
		return Make(entity)
	})(model.FixedProvider(shopEntities))(model.ParallelMap())

	return model.SliceMap(model.Decorate(append(decorators, p.RechargeableConsumablesDecorator)))(sbp)(model.ParallelMap())
}

func (p *ProcessorImpl) BuyAndEmit(characterId uint32, slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) error {
	return message.Emit(p.kp)(func(mb *message.Buffer) error {
		return p.Buy(mb)(characterId)(slot, itemTemplateId, quantity, discountPrice)
	})
}

func (p *ProcessorImpl) Buy(mb *message.Buffer) func(characterId uint32) func(slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) error {
	return func(characterId uint32) func(slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) error {
		return func(slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) error {
			p.l.Debugf("Character [%d] attempting to buy item [%d] from slot [%d].", characterId, itemTemplateId, slot)

			shopId, inShop := getRegistry().GetShop(p.t.Id(), characterId)
			if !inShop {
				p.l.Errorf("Character [%d] is not in a shop.", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			cms, err := p.cp.GetByNpcId(shopId)
			if err != nil {
				p.l.WithError(err).Errorf("Cannot locate shop [%d] character [%d] is attempting to buy from.", shopId, characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			found := false
			var cm commodities.Model
			for _, cm = range cms {
				if cm.TemplateId() == itemTemplateId {
					found = true
					break
				}
			}
			if !found {
				p.l.Errorf("Character [%d] is attempting to buy item [%d] from slot [%d] but it is not available.", characterId, itemTemplateId, slot)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			// TODO: this needs better transaction handling.

			c, err := p.charP.GetById(p.charP.InventoryDecorator)(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Cannot locate character [%d].", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			if cm.MesoPrice() > 0 {
				totalCost := cm.MesoPrice() * quantity

				if c.Meso() < totalCost {
					p.l.Errorf("Character [%d] is attempting to buy item [%d] from slot [%d] but they do not have enough meso.", characterId, itemTemplateId, slot)
					return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorNotEnoughMoney))
				}
				it, ok := inventory.TypeFromItemId(item.Id(itemTemplateId))
				if !ok {
					p.l.Errorf("Character [%d] is attempting to buy item [%d] from slot [%d] but it is not a valid item.", characterId, itemTemplateId, slot)
					return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
				}
				_, err = c.Inventory().CompartmentByType(it).NextFreeSlot()
				if err != nil {
					p.l.WithError(err).Errorf("Cannot locate free slot for character [%d].", characterId)
					return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorInventoryFull))
				}
				_ = p.charP.RequestChangeMeso(c.WorldId(), c.Id(), c.Id(), "SHOP", -int32(totalCost))
				_ = p.compP.RequestCreateItem(c.Id(), itemTemplateId, quantity)
				p.l.Debugf("Character [%d] bought item [%d].", characterId, itemTemplateId)
				return nil
			}

			// TODO: implement TokenItem purchasing.
			return mb.Put(shops.EnvStatusEventTopic, reasonErrorEventProvider(characterId, shops.ErrorGenericErrorWithReason, "not implemented"))
		}
	}
}

func (p *ProcessorImpl) SellAndEmit(characterId uint32, slot int16, itemTemplateId uint32, quantity uint32) error {
	return message.Emit(p.kp)(func(mb *message.Buffer) error {
		return p.Sell(mb)(characterId)(slot, itemTemplateId, quantity)
	})
}

func (p *ProcessorImpl) Sell(mb *message.Buffer) func(characterId uint32) func(slot int16, itemTemplateId uint32, quantity uint32) error {
	return func(characterId uint32) func(slot int16, itemTemplateId uint32, quantity uint32) error {
		return func(slot int16, itemTemplateId uint32, quantity uint32) error {
			p.l.Debugf("Character [%d] attempting to sell [%d] item [%d] from slot [%d].", characterId, quantity, itemTemplateId, slot)

			_, inShop := getRegistry().GetShop(p.t.Id(), characterId)
			if !inShop {
				p.l.Errorf("Character [%d] is not in a shop.", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			// TODO: this needs better transaction handling.

			c, err := p.charP.GetById(p.charP.InventoryDecorator)(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Cannot locate character [%d].", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}
			it, ok := inventory.TypeFromItemId(item.Id(itemTemplateId))
			if !ok {
				p.l.Errorf("Character [%d] is attempting to sell item [%d] from slot [%d] but it is not a valid item.", characterId, itemTemplateId, slot)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}
			a, ok := c.Inventory().CompartmentByType(it).FindBySlot(slot)
			if !ok {
				p.l.Errorf("Character [%d] is attempting to sell item [%d] from slot [%d] but it is not in their inventory.", characterId, itemTemplateId, slot)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}
			if a.TemplateId() != itemTemplateId {
				p.l.Errorf("Character [%d] is attempting to sell item [%d] from slot [%d] but it is not in their inventory.", characterId, itemTemplateId, slot)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}
			if a.Quantity() < quantity {
				p.l.Errorf("Character [%d] is attempting to sell item [%d] from slot [%d] but it is not in their inventory.", characterId, itemTemplateId, slot)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorNeedMoreItems))
			}

			price := uint32(0)
			if it == inventory.TypeValueEquip {
				var em equipable.Model
				em, err = equipable.NewProcessor(p.l, p.ctx).GetById(itemTemplateId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to get item template [%d].", itemTemplateId)
					return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
				}
				price = em.Price()
			} else if it == inventory.TypeValueUse {
				var cm consumable.Model
				cm, err = consumable.NewProcessor(p.l, p.ctx).GetById(itemTemplateId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to get item template [%d].", itemTemplateId)
					return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
				}
				price = cm.Price()
			} else if it == inventory.TypeValueSetup {
				var sm setup.Model
				sm, err = setup.NewProcessor(p.l, p.ctx).GetById(itemTemplateId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to get item template [%d].", itemTemplateId)
					return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
				}
				price = sm.Price()
			} else if it == inventory.TypeValueETC {
				var em etc.Model
				em, err = etc.NewProcessor(p.l, p.ctx).GetById(itemTemplateId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to get item template [%d].", itemTemplateId)
					return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
				}
				price = em.Price()
			}
			price = price * quantity

			_ = p.charP.RequestChangeMeso(c.WorldId(), c.Id(), c.Id(), "SHOP", int32(price))
			_ = p.compP.RequestDestroyItem(characterId, it, slot, quantity)

			p.l.Debugf("Character [%d] sold [%d] item [%d] from slot [%d].", characterId, quantity, itemTemplateId, slot)
			return nil
		}
	}
}

func (p *ProcessorImpl) RechargeAndEmit(characterId uint32, slot uint16) error {
	return message.Emit(p.kp)(func(mb *message.Buffer) error {
		return p.Recharge(mb)(characterId)(slot)
	})
}

func (p *ProcessorImpl) Recharge(mb *message.Buffer) func(characterId uint32) func(slot uint16) error {
	return func(characterId uint32) func(slot uint16) error {
		return func(slot uint16) error {
			p.l.Debugf("Character [%d] attempting to recharge item from slot [%d].", characterId, slot)

			shopId, inShop := getRegistry().GetShop(p.t.Id(), characterId)
			if !inShop {
				p.l.Errorf("Character [%d] is not in a shop.", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			// Check if the shop allows recharging
			shopEntity, err := getByNpcId(p.t.Id(), shopId)(p.db)()
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve shop entity for NPC [%d].", shopId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			if !shopEntity.Recharger {
				p.l.Errorf("Character [%d] attempting to recharge item in shop [%d] that does not allow recharging.", characterId, shopId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}
			c, err := p.charP.GetById(p.charP.InventoryDecorator)(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve character [%d].", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}
			rim, ok := c.Inventory().Consumable().FindBySlot(int16(slot))
			if !ok {
				p.l.Errorf("Unable to retrieve item in slot [%d] for character [%d] being recharged.", slot, characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}
			cm, err := consumable.NewProcessor(p.l, p.ctx).GetById(rim.TemplateId())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to get item template [%d].", rim.TemplateId())
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}
			sms, err := skill.NewProcessor(p.l, p.ctx).GetByCharacterId(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to locate skills for character [%d].", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}
			addSlotMax := uint16(0)
			if item.IsThrowingStar(item.Id(rim.TemplateId())) {
				addSlotMax += uint16(skill.GetLevel(sms, skill2.NightWalkerStage2ClawMasteryId)) * 10
				addSlotMax += uint16(skill.GetLevel(sms, skill2.AssassinClawMasteryId)) * 10
			}
			if item.IsBullet(item.Id(rim.TemplateId())) {
				addSlotMax += uint16(skill.GetLevel(sms, skill2.GunslingerGunMasteryId)) * 10
			}
			slotMax := uint16(cm.SlotMax()) + addSlotMax
			if rim.Quantity() >= uint32(slotMax) {
				p.l.Warnf("Character [%d] attempting to recharge item [%d] in slot [%d] that does not need recharging.", characterId, rim.TemplateId(), slot)
				return nil
			}
			price := math.Ceil(cm.UnitPrice() * float64(uint32(slotMax)-rim.Quantity()))
			if c.Meso() < uint32(price) {
				p.l.Debugf("Character [%d] has [%d] meso. Needs [%d] meso to recharge item [%d] in slot [%d].", characterId, c.Meso(), price, rim.TemplateId(), slot)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorNotEnoughMoney2))
			}

			// Decrement character's meso
			err = p.charP.RequestChangeMeso(c.WorldId(), c.Id(), c.Id(), "SHOP", -int32(price))
			if err != nil {
				p.l.WithError(err).Errorf("Unable to decrement meso for character [%d].", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			// Recharge the item
			quantityToAdd := uint32(slotMax) - rim.Quantity()
			err = p.compP.RequestRechargeItem(characterId, inventory.TypeValueUse, int16(slot), quantityToAdd)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to recharge item for character [%d].", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			p.l.Debugf("Character [%d] recharged item [%d] in slot [%d] with [%d] quantity.", characterId, rim.TemplateId(), slot, quantityToAdd)
			return nil
		}
	}
}

func (p *ProcessorImpl) RechargeableConsumablesDecorator(m Model) Model {
	if p.RechargeableConsumablesDecoratorFn != nil {
		return p.RechargeableConsumablesDecoratorFn(m)
	}

	if !m.Recharger() {
		return m
	}

	existing := m.Commodities()
	existingByTemplateId := make(map[uint32]commodities.Model, len(existing))
	for _, ec := range existing {
		existingByTemplateId[ec.TemplateId()] = ec
	}

	// Prepare the final list
	finalCommodities := make([]commodities.Model, 0, len(existing))

	// Update existing commodities if they match a rechargeable one
	for _, ec := range existing {
		if rc := findRechargeable(ec.TemplateId(), GetConsumableCache().GetConsumables(p.l, p.ctx, p.t.Id())); rc != nil {
			updated, err := commodities.Clone(ec).
				SetSlotMax(rc.SlotMax()).
				SetUnitPrice(rc.UnitPrice()).
				Build()
			if err == nil {
				ec = updated
			}
		}
		finalCommodities = append(finalCommodities, ec)
	}

	// Add new rechargeable consumables that do not exist
	for _, rc := range GetConsumableCache().GetConsumables(p.l, p.ctx, p.t.Id()) {
		if _, found := existingByTemplateId[rc.Id()]; !found {
			cm, err := commodities.NewBuilder().
				SetId(uuid.New()).
				SetNpcId(m.NpcId()).
				SetTemplateId(rc.Id()).
				SetSlotMax(rc.SlotMax()).
				SetUnitPrice(rc.UnitPrice()).
				Build()
			if err == nil {
				finalCommodities = append(finalCommodities, cm)
			}
		}
	}

	result, err := Clone(m).SetCommodities(finalCommodities).Build()
	if err != nil {
		return m
	}
	return result
}

func findRechargeable(templateId uint32, rechargeables []consumable.Model) *consumable.Model {
	for _, rc := range rechargeables {
		if rc.Id() == templateId {
			return &rc
		}
	}
	return nil
}
