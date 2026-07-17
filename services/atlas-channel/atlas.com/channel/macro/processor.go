package macro

import (
	macro2 "atlas-channel/kafka/message/macro"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	Update(characterId uint32, macros []Model) error
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

// ByCharacterIdProvider fetches every skill macro for a character. The
// upstream atlas-skills list is now paginated (task-117); callers here need
// the complete set (e.g. sending the full macro record on channel spawn),
// so this drains every page rather than fetching just the first.
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(characterMacrosUrl(characterId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) Update(characterId uint32, macros []Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(macro2.EnvCommandTopic)(UpdateCommandProvider(characterId, macros))
}
