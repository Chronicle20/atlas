package information

import (
	"context"
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

type Processor interface {
	GetById(monsterId uint32) (Model, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetById returns the parsed template attack info for monsterId, served
// from a tenant-scoped in-process read-through TTL cache when enabled.
func (p *ProcessorImpl) GetById(monsterId uint32) (Model, error) {
	c := getInfoCache()
	if !c.cfg.enabled {
		return upstreamFn(p.l, p.ctx, monsterId)
	}

	t := tenant.MustFromContext(p.ctx)
	now := time.Now()

	if e, ok := c.lookup(t.Id(), monsterId, now); ok {
		if e.negative {
			recordCacheHit(t, "negative")
			return Model{}, notFoundError(monsterId)
		}
		recordCacheHit(t, "positive")
		return e.model, nil
	}

	recordCacheMiss(t)
	m, err := upstreamFn(p.l, p.ctx, monsterId)
	if err == nil {
		c.put(t.Id(), monsterId, cacheEntry{model: m, expiresAt: now.Add(c.cfg.ttl)})
		return m, nil
	}
	// Negative caching only for the not-found sentinel; transient errors
	// (network, 5xx, parse) are never cached (task-060 classification).
	if errors.Is(err, requests.ErrNotFound) && c.cfg.negativeTTL > 0 {
		c.put(t.Id(), monsterId, cacheEntry{negative: true, expiresAt: now.Add(c.cfg.negativeTTL)})
	}
	return Model{}, err
}
