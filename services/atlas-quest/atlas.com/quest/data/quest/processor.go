package quest

import (
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

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

func (p *ProcessorImpl) GetQuestDefinition(questId uint32) (RestModel, error) {
	result, err := requestQuestById(questId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get quest definition for quest %d", questId)
		return RestModel{}, err
	}
	return result, nil
}

func (p *ProcessorImpl) GetAutoStartQuests(mapId _map.Id) ([]RestModel, error) {
	allAutoStart, err := requestAutoStartQuests()(p.l, p.ctx)
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
