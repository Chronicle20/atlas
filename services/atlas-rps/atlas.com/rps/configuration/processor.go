package configuration

import (
	"atlas-rps/game"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// ErrNoRewardConfig indicates a tenant has no rps-rewards configuration. The
// game cannot run without a reward ladder, so this correctly aborts the game
// (Start fails loud on a ladder error) and the entry saga compensates/refunds.
var ErrNoRewardConfig = errors.New("no rps-rewards configuration for tenant")

// Processor defines the interface for rps-rewards configuration operations.
type Processor interface {
	// GetLadder returns the reward ladder configured for a tenant.
	GetLadder(tenantId uuid.UUID) (game.Ladder, error)
}

// ProcessorImpl implements the Processor interface.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new processor implementation.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetLadder returns the reward ladder configured for a tenant. atlas-tenants
// serves the rps-rewards resource as a JSON:API collection, so the response is
// decoded as a slice; the first (and only expected) record is returned. A tenant
// with no configured record yields ErrNoRewardConfig.
func (p *ProcessorImpl) GetLadder(tenantId uuid.UUID) (game.Ladder, error) {
	p.l.Debugf("Fetching rps-rewards configuration for tenant [%s].", tenantId)
	ladders, err := requests.SliceProvider[RpsRewardRestModel, game.Ladder](p.l, p.ctx)(requestRewards(tenantId.String()), Extract, model.Filters[game.Ladder]())()
	if err != nil {
		return game.Ladder{}, err
	}
	if len(ladders) == 0 {
		return game.Ladder{}, ErrNoRewardConfig
	}
	return ladders[0], nil
}
