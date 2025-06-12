package inventory

import (
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/inventory"
	inventory2 "atlas-cashshop/kafka/producer/inventory"
	"context"
	inventory3 "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[Model]
	GetByCharacterId(characterId uint32) (Model, error)
	IncreaseCapacity(mb *message.Buffer) func(characterId uint32, inventoryType inventory3.Type, amount uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
	return p
}

func (p *ProcessorImpl) IncreaseCapacity(mb *message.Buffer) func(characterId uint32, inventoryType inventory3.Type, amount uint32) error {
	return func(characterId uint32, inventoryType inventory3.Type, amount uint32) error {
		return mb.Put(inventory.EnvCommandTopic, inventory2.IncreaseCapacityCommandProvider(characterId, byte(inventoryType), amount))
	}
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}
