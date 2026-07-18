package quest

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// identity is the no-op Transformer requests.DrainProvider needs when the
// wire type (RestModel) is also the domain type this package works with.
func identity(rm RestModel) (RestModel, error) {
	return rm, nil
}

// Processor provides quest definition lookup functionality from atlas-data.
type Processor interface {
	// GetQuestDefinition fetches the full quest definition from atlas-data.
	GetQuestDefinition(questId uint32) (RestModel, error)
	// GetAutoStartQuests fetches all auto-start quests and filters by mapId.
	// If mapId is 0, returns all auto-start quests.
	GetAutoStartQuests(mapId _map.Id) ([]RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetQuestDefinition(questId uint32) (RestModel, error) {
	result, err := requestQuestById(questId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get quest definition for quest %d", questId)
		return RestModel{}, err
	}
	return result, nil
}

func (p *ProcessorImpl) GetAutoStartQuests(mapId _map.Id) ([]RestModel, error) {
	// atlas-data's GET /data/quests/auto-start is now paginated (task-117);
	// this drains every page rather than fetching one, since callers need
	// the complete auto-start set to filter by mapId below.
	allAutoStart, err := requests.DrainProvider[RestModel, RestModel](p.l, p.ctx)(autoStartQuestsUrl(), 250, identity, model.Filters[RestModel]())()
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get auto-start quests")
		return nil, err
	}

	// If no mapId filter, return all
	if mapId == 0 {
		return allAutoStart, nil
	}

	// Filter by map - quests that should auto-start on this map
	var filtered []RestModel
	for _, q := range allAutoStart {
		// NormalAutoStart means the quest can auto-start on any map
		// Area > 0 means the quest should only auto-start on that specific map
		if q.StartRequirements.NormalAutoStart || (q.Area > 0 && q.Area == mapId) {
			filtered = append(filtered, q)
		}
	}

	return filtered, nil
}
