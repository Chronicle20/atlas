package progress

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// ErrNotFound is returned when the character has no progress record for the quest
var ErrNotFound = errors.New("quest progress not found")

// Processor provides operations for querying quest progress data
type Processor interface {
	// GetByCharacterAndQuest returns every progress entry for a character's
	// quest. Returns ErrNotFound if the character has no record for the quest.
	GetByCharacterAndQuest(characterId uint32, questId uint32) model.Provider[[]Model]
}

type processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new quest progress processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &processor{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*processor)(nil)

// GetByCharacterAndQuest returns every progress entry for a character's quest
func (p *processor) GetByCharacterAndQuest(characterId uint32, questId uint32) model.Provider[[]Model] {
	return func() ([]Model, error) {
		url, err := ByCharacterAndQuestUrl(p.ctx, characterId, questId)
		if err != nil {
			return nil, err
		}
		result, err := requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())()
		if err != nil {
			if errors.Is(err, requests.ErrNotFound) {
				return nil, ErrNotFound
			}
			p.l.WithError(err).Errorf("Failed to get quest progress for character %d quest %d", characterId, questId)
			return nil, err
		}
		return result, nil
	}
}
