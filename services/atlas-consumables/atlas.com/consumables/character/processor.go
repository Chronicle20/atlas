package character

import (
	"atlas-consumables/inventory"
	character2 "atlas-consumables/kafka/message/character"
	"atlas-consumables/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	ip  *inventory.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		ip:  inventory.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		cp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
		return model.Map(model.Decorate(decorators))(cp)()
	}
}

func (p *Processor) InventoryDecorator(m Model) Model {
	i, err := p.ip.GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return m.SetInventory(i)
}

func (p *Processor) ChangeMap(f field.Model, characterId uint32, portalId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(changeMapProvider(f, characterId, portalId))
}

func (p *Processor) ChangeHP(f field.Model, characterId uint32, amount int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(changeHPCommandProvider(f, characterId, amount))
}

func (p *Processor) ChangeMP(f field.Model, characterId uint32, amount int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(changeMPCommandProvider(f, characterId, amount))
}
