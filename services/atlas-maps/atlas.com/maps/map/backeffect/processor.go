package backeffect

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	Set(f field.Model, entry BackEffectEntry)
	Clear(f field.Model) bool
	GetActive(f field.Model) []BackEffectEntry
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Set(f field.Model, entry BackEffectEntry) {
	t := tenant.MustFromContext(p.ctx)
	key := FieldKey{Tenant: t, Field: f}
	getRegistry().Set(key, entry)
	p.l.Debugf("Set back effect in map [%d] instance [%s] page [%d] effect [%d].", f.MapId(), f.Instance(), entry.PageId, entry.Effect)
}

func (p *ProcessorImpl) Clear(f field.Model) bool {
	t := tenant.MustFromContext(p.ctx)
	key := FieldKey{Tenant: t, Field: f}
	removed := getRegistry().Clear(key)
	p.l.Debugf("Cleared back effects in map [%d] instance [%s].", f.MapId(), f.Instance())
	return removed
}

func (p *ProcessorImpl) GetActive(f field.Model) []BackEffectEntry {
	t := tenant.MustFromContext(p.ctx)
	key := FieldKey{Tenant: t, Field: f}
	return getRegistry().Get(key)
}
