package configuration

import (
	"atlas-rps/game"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

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

// GetLadder returns the reward ladder configured for a tenant.
func (p *ProcessorImpl) GetLadder(tenantId uuid.UUID) (game.Ladder, error) {
	p.l.Debugf("Fetching rps-rewards configuration for tenant [%s].", tenantId)
	return requests.Provider[RpsRewardRestModel, game.Ladder](p.l, p.ctx)(requestRewards(tenantId.String()), Extract)()
}
