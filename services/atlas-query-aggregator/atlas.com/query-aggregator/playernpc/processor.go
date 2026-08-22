package playernpc

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Processor provides operations for querying player-NPC eligibility.
type Processor interface {
	GetEligibility(characterId uint32, mapId _map.Id, worldId world.Id) (EligibilityModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new player-npc eligibility processor.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetEligibility calls atlas-player-npcs' eligibility predicate
// (design §9.1) for the given character/map/world and returns the result.
func (p *ProcessorImpl) GetEligibility(characterId uint32, mapId _map.Id, worldId world.Id) (EligibilityModel, error) {
	resp, err := requestEligibility(p.l, p.ctx, characterId, uint32(mapId), byte(worldId))
	if err != nil {
		return EligibilityModel{}, err
	}
	return NewEligibilityModel(resp.Eligible, resp.Reason), nil
}
