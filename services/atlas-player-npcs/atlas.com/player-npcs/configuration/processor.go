package configuration

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	// GetByTenantId resolves the tenant's player-npcs configuration.
	// Missing config (404) is the expected unconfigured state; any other
	// error is logged at warn. Both fall back to DefaultModel so one
	// tenant's config problem never stalls deployment.
	GetByTenantId(tenantId uuid.UUID) Model
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetByTenantId(tenantId uuid.UUID) Model {
	rm, err := requestByTenantId(p.ctx, tenantId)(p.l, p.ctx)
	if err != nil {
		if !errors.Is(err, requests.ErrNotFound) {
			p.l.WithError(err).Warnf("Unable to read player-npcs configuration for tenant [%s]; using defaults.", tenantId)
		}
		return DefaultModel()
	}
	m, err := Extract(rm)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to extract player-npcs configuration for tenant [%s]; using defaults.", tenantId)
		return DefaultModel()
	}
	return m
}
