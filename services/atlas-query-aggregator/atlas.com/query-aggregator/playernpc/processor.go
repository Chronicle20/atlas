package playernpc

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// EligibilityModel is the domain view of the eligibility predicate.
type EligibilityModel struct {
	eligible bool
	reason   string
}

func (m EligibilityModel) Eligible() bool { return m.eligible }
func (m EligibilityModel) Reason() string { return m.reason }

// NewEligibilityModel builds an EligibilityModel directly -- used by the
// GetEligibility implementation to wrap a decoded REST response, and by
// tests to build fakes without exporting the struct's fields.
func NewEligibilityModel(eligible bool, reason string) EligibilityModel {
	return EligibilityModel{eligible: eligible, reason: reason}
}

// NewUnavailableEligibility builds the fail-closed EligibilityModel a caller
// returns when the eligibility endpoint could not be reached or has no
// processor wired -- the graceful-degradation result GetPlayerNpcEligibility
// (validation/context.go) returns rather than propagating an error the
// evaluator contract has no channel for.
func NewUnavailableEligibility(reason string) EligibilityModel {
	return NewEligibilityModel(false, reason)
}

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
