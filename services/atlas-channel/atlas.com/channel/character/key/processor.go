package key

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor interface defines the operations for key processing
type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	Update(characterId uint32, key int32, theType int8, action int32) error
}

// ProcessorImpl implements the Processor interface
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

// ByCharacterIdProvider fetches every key binding for a character. The
// upstream atlas-keys list is now paginated (task-117); callers here need
// the complete key map (e.g. sending the full key-map record on channel
// spawn), so this drains every page rather than fetching just the first.
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	url, err := characterKeysUrl(p.ctx, characterId)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) Update(characterId uint32, key int32, theType int8, action int32) error {
	_, err := updateKey(p.ctx, characterId, key, theType, action)(p.l, p.ctx)
	return err
}
