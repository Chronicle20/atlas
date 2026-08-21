package jukebox

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	Start(f field.Model, itemId uint32, playerName string, duration time.Duration)
	GetActive(f field.Model) (JukeboxEntry, bool)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Start(f field.Model, itemId uint32, playerName string, duration time.Duration) {
	t := tenant.MustFromContext(p.ctx)
	key := FieldKey{Tenant: t, Field: f}
	entry := JukeboxEntry{
		ItemId:     itemId,
		PlayerName: playerName,
		ExpiresAt:  time.Now().Add(duration),
	}
	getRegistry().Set(key, entry)
	p.l.Debugf("Jukebox started in map [%d] instance [%s] with item [%d] by [%s] for [%s].", f.MapId(), f.Instance(), itemId, playerName, duration)
}

func (p *ProcessorImpl) GetActive(f field.Model) (JukeboxEntry, bool) {
	t := tenant.MustFromContext(p.ctx)
	key := FieldKey{Tenant: t, Field: f}
	return getRegistry().Get(key)
}
