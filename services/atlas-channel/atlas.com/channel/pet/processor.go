package pet

import (
	pet2 "atlas-channel/kafka/message/pet"
	"atlas-channel/kafka/producer"
	pet3 "atlas-channel/kafka/producer/pet"
	"atlas-channel/pet/exclude"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) ByIdProvider(petId uint64) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(petId), Extract)
}

func (p *Processor) GetById(petId uint64) (Model, error) {
	return p.ByIdProvider(petId)()
}

func (p *Processor) ByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByOwnerId(ownerId), Extract, model.Filters[Model]())
}

func (p *Processor) GetByOwner(ownerId uint32) ([]Model, error) {
	return p.ByOwnerProvider(ownerId)()
}

func (p *Processor) GetByOwnerItem(ownerId uint32, inventoryItemId uint32) (Model, error) {
	return model.FirstProvider(p.ByOwnerProvider(ownerId), model.Filters(InventoryItemIdFilter(inventoryItemId)))()
}

func InventoryItemIdFilter(inventoryItemId uint32) model.Filter[Model] {
	return func(m Model) bool {
		return m.InventoryItemId() == inventoryItemId
	}
}

func (p *Processor) Spawn(characterId uint32, petId uint64, lead bool) error {
	p.l.Debugf("Character [%d] attempting to spawn pet [%d]", characterId, petId)
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(pet3.SpawnProvider(characterId, petId, lead))
}

func (p *Processor) Despawn(characterId uint32, petId uint64) error {
	p.l.Debugf("Character [%d] attempting to despawn pet [%d].", characterId, petId)
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(pet3.DespawnProvider(characterId, petId))
}

func (p *Processor) AttemptCommand(petId uint64, commandId byte, byName bool, characterId uint32) error {
	p.l.Debugf("Character [%d] triggered pet [%d] command. byName [%t], command [%d]", characterId, petId, byName, commandId)
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(pet3.AttemptCommandProvider(petId, commandId, byName, characterId))
}

func (p *Processor) SetExcludeItems(characterId uint32, petId uint64, items []exclude.Model) error {
	p.l.Debugf("Character [%d] setting exclude items for pet [%d]. count [%d].", characterId, petId, len(items))
	itemIds := make([]uint32, len(items))
	for i, item := range items {
		itemIds[i] = item.ItemId()
	}
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(pet3.SetExcludesCommandProvider(characterId, petId, itemIds))
}
