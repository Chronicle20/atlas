package shops

import (
	"atlas-npc/character"
	"atlas-npc/commodities"
	"atlas-npc/compartment"
	"atlas-npc/kafka/message"
	"atlas-npc/kafka/message/shops"
	"atlas-npc/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetByNpcId(npcId uint32) (Model, error)
	ByNpcIdProvider(npcId uint32) model.Provider[Model]
	GetAllShops() ([]Model, error)
	AllShopsProvider() model.Provider[[]Model]
	AddCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error)
	UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error)
	RemoveCommodity(id uuid.UUID) error
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

type ProcessorImpl struct {
	l             logrus.FieldLogger
	ctx           context.Context
	db            *gorm.DB
	t             tenant.Model
	GetByNpcIdFn  func(npcId uint32) (Model, error)
	GetAllShopsFn func() ([]Model, error)
	cp            commodities.Processor
	charP         *character.Processor
	compP         *compartment.Processor
	kp            producer.Provider
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
		kp:    producer.ProviderImpl(l)(ctx),
	}
	p.GetByNpcIdFn = model.CollapseProvider(p.ByNpcIdProvider)
	p.GetAllShopsFn = func() ([]Model, error) {
		return p.AllShopsProvider()()
	}
	return p
}

func (p *ProcessorImpl) GetByNpcId(npcId uint32) (Model, error) {
	return p.GetByNpcIdFn(npcId)
}

func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[Model] {
	cms, err := p.cp.GetByNpcId(npcId)
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	return model.FixedProvider(NewBuilder(npcId).SetCommodities(cms).Build())
}

func (p *ProcessorImpl) AddCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error) {
	return p.cp.CreateCommodity(npcId, templateId, mesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
}

func (p *ProcessorImpl) UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error) {
	return p.cp.UpdateCommodity(id, templateId, mesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
}

func (p *ProcessorImpl) RemoveCommodity(id uuid.UUID) error {
	return p.cp.DeleteCommodity(id)
}

func (p *ProcessorImpl) EnterAndEmit(characterId uint32, npcId uint32) error {
	return message.Emit(p.kp)(model.Flip(model.Flip(p.Enter)(characterId))(npcId))
}

func (p *ProcessorImpl) Enter(mb *message.Buffer) func(characterId uint32) func(npcId uint32) error {
	return func(characterId uint32) func(npcId uint32) error {
		return func(npcId uint32) error {
			p.l.Debugf("Character [%d] attempting to enter shop [%d].", characterId, npcId)
			_, err := p.GetByNpcId(npcId)
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
		getRegistry().RemoveCharacter(p.t.Id(), characterId)
		return mb.Put(shops.EnvStatusEventTopic, exitedEventProvider(characterId))
	}
}

func (p *ProcessorImpl) GetCharactersInShop(shopId uint32) []uint32 {
	return getRegistry().GetCharactersInShop(p.t.Id(), shopId)
}

func (p *ProcessorImpl) GetAllShops() ([]Model, error) {
	return p.GetAllShopsFn()
}

func (p *ProcessorImpl) AllShopsProvider() model.Provider[[]Model] {
	// Get all commodities for the tenant using the commodities processor
	allCommodities, err := p.cp.GetAllByTenant()
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}

	// Initialize a map to store shop builders by NPC ID
	shopBuilders := make(map[uint32]*ModelBuilder)

	// Iterate over all commodities and accumulate them in shop builders
	for _, commodity := range allCommodities {
		npcId := commodity.NpcId()

		// If we don't have a builder for this NPC ID yet, create one
		if _, exists := shopBuilders[npcId]; !exists {
			shopBuilders[npcId] = NewBuilder(npcId)
		}

		// Add the commodity to the appropriate shop builder
		shopBuilders[npcId].AddCommodity(commodity)
	}

	// Build all shops from the accumulated builders
	shops := make([]Model, 0, len(shopBuilders))
	for _, builder := range shopBuilders {
		shops = append(shops, builder.Build())
	}

	return model.FixedProvider(shops)
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
				it, ok := inventory.TypeFromItemId(itemTemplateId)
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
			it, ok := inventory.TypeFromItemId(itemTemplateId)
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

			// TODO give proper value for item being sold.
			_ = p.charP.RequestChangeMeso(c.WorldId(), c.Id(), c.Id(), "SHOP", 1000)
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

			_, inShop := getRegistry().GetShop(p.t.Id(), characterId)
			if !inShop {
				p.l.Errorf("Character [%d] is not in a shop.", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			// TODO: Implement recharge logic
			return mb.Put(shops.EnvStatusEventTopic, reasonErrorEventProvider(characterId, shops.ErrorGenericErrorWithReason, "not implemented"))
		}
	}
}
