package pet

import (
	pet2 "atlas-channel/kafka/message/pet"
	"atlas-channel/pet/exclude"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	ByIdProvider(petId uint32) model.Provider[Model]
	GetById(petId uint32) (Model, error)
	ByOwnerProvider(ownerId uint32) model.Provider[[]Model]
	GetByOwner(ownerId uint32) ([]Model, error)
	Spawn(characterId uint32, petId uint32, lead bool) error
	Despawn(characterId uint32, petId uint32) error
	AttemptCommand(petId uint32, commandId byte, byName bool, characterId uint32) error
	SetExcludeItems(characterId uint32, petId uint32, items []exclude.Model) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByIdProvider(petId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(petId), Extract)
}

func (p *ProcessorImpl) GetById(petId uint32) (Model, error) {
	return p.ByIdProvider(petId)()
}

// ByOwnerProvider fetches every pet owned by a character. The upstream
// atlas-pets list is now paginated (task-117); callers here need the
// complete set (e.g. rendering the character-info popup, sending the full
// pet list on channel spawn), so this drains every page rather than
// fetching just the first.
func (p *ProcessorImpl) ByOwnerProvider(ownerId uint32) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(byOwnerUrl(ownerId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByOwner(ownerId uint32) ([]Model, error) {
	return p.ByOwnerProvider(ownerId)()
}

func (p *ProcessorImpl) Spawn(characterId uint32, petId uint32, lead bool) error {
	p.l.Debugf("Character [%d] attempting to spawn pet [%d]", characterId, petId)
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(SpawnProvider(characterId, petId, lead))
}

func (p *ProcessorImpl) Despawn(characterId uint32, petId uint32) error {
	p.l.Debugf("Character [%d] attempting to despawn pet [%d].", characterId, petId)
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(DespawnProvider(characterId, petId))
}

func (p *ProcessorImpl) AttemptCommand(petId uint32, commandId byte, byName bool, characterId uint32) error {
	p.l.Debugf("Character [%d] triggered pet [%d] command. byName [%t], command [%d]", characterId, petId, byName, commandId)
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(AttemptCommandProvider(petId, commandId, byName, characterId))
}

func (p *ProcessorImpl) SetExcludeItems(characterId uint32, petId uint32, items []exclude.Model) error {
	p.l.Debugf("Character [%d] setting exclude items for pet [%d]. count [%d].", characterId, petId, len(items))
	itemIds := make([]uint32, len(items))
	for i, item := range items {
		itemIds[i] = item.ItemId()
	}
	return producer.ProviderImpl(p.l)(p.ctx)(pet2.EnvCommandTopic)(SetExcludesCommandProvider(characterId, petId, itemIds))
}
