package shops

import (
	"atlas-npc/commodities"
	"atlas-npc/kafka/message"
	"atlas-npc/kafka/message/shops"
	"atlas-npc/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetByNpcId(npcId uint32) (Model, error)
	ByNpcIdProvider(npcId uint32) model.Provider[Model]
	AddCommodity(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error)
	UpdateCommodity(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenItemId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (commodities.Model, error)
	RemoveCommodity(id uuid.UUID) error
	EnterAndEmit(characterId uint32, npcId uint32) error
	Enter(mb *message.Buffer) func(characterId uint32) func(npcId uint32) error
	ExitAndEmit(characterId uint32) error
	Exit(mb *message.Buffer) func(characterId uint32) error
	BuyAndEmit(characterId uint32, slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) error
	Buy(mb *message.Buffer) func(characterId uint32) func(slot uint16, itemTemplateId uint32, quantity uint32, discountPrice uint32) error
	SellAndEmit(characterId uint32, slot uint16, itemTemplateId uint32, quantity uint32) error
	Sell(mb *message.Buffer) func(characterId uint32) func(slot uint16, itemTemplateId uint32, quantity uint32) error
	RechargeAndEmit(characterId uint32, slot uint16) error
	Recharge(mb *message.Buffer) func(characterId uint32) func(slot uint16) error
	GetCharactersInShop(shopId uint32) []uint32
}

type ProcessorImpl struct {
	l            logrus.FieldLogger
	ctx          context.Context
	db           *gorm.DB
	t            tenant.Model
	GetByNpcIdFn func(npcId uint32) (Model, error)
	cp           commodities.Processor
	kp           producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		cp:  commodities.NewProcessor(l, ctx, db),
		kp:  producer.ProviderImpl(l)(ctx),
	}
	p.GetByNpcIdFn = model.CollapseProvider(p.ByNpcIdProvider)
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
			for _, cm := range cms {
				if cm.TemplateId() == itemTemplateId {
					found = true
					break
				}
			}
			if !found {
				p.l.Errorf("Character [%d] is attempting to buy item [%d] from slot [%d] but it is not available.", characterId, itemTemplateId, slot)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			// TODO: Implement buy logic
			p.l.Debugf("Character [%d] is in shop [%d].", characterId, shopId)

			return nil
		}
	}
}

func (p *ProcessorImpl) SellAndEmit(characterId uint32, slot uint16, itemTemplateId uint32, quantity uint32) error {
	return message.Emit(p.kp)(func(mb *message.Buffer) error {
		return p.Sell(mb)(characterId)(slot, itemTemplateId, quantity)
	})
}

func (p *ProcessorImpl) Sell(mb *message.Buffer) func(characterId uint32) func(slot uint16, itemTemplateId uint32, quantity uint32) error {
	return func(characterId uint32) func(slot uint16, itemTemplateId uint32, quantity uint32) error {
		return func(slot uint16, itemTemplateId uint32, quantity uint32) error {
			p.l.Debugf("Character [%d] attempting to sell item [%d] from slot [%d].", characterId, itemTemplateId, slot)

			_, inShop := getRegistry().GetShop(p.t.Id(), characterId)
			if !inShop {
				p.l.Errorf("Character [%d] is not in a shop.", characterId)
				return mb.Put(shops.EnvStatusEventTopic, errorEventProvider(characterId, shops.ErrorGenericError))
			}

			// TODO: Implement sell logic
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
			return nil
		}
	}
}
