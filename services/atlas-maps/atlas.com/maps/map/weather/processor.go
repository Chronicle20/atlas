package weather

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Start(f field.Model, itemId uint32, message string, duration time.Duration)
	GetActive(f field.Model) (WeatherEntry, bool)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) Start(f field.Model, itemId uint32, message string, duration time.Duration) {
	t := tenant.MustFromContext(p.ctx)
	key := FieldKey{Tenant: t, Field: f}
	entry := WeatherEntry{
		ItemId:    itemId,
		Message:   message,
		ExpiresAt: time.Now().Add(duration),
	}
	getRegistry().Set(key, entry)
	p.l.Debugf("Weather started in map [%d] instance [%s] with item [%d] for [%s].", f.MapId(), f.Instance(), itemId, duration)
}

func (p *ProcessorImpl) GetActive(f field.Model) (WeatherEntry, bool) {
	t := tenant.MustFromContext(p.ctx)
	key := FieldKey{Tenant: t, Field: f}
	return getRegistry().Get(key)
}
