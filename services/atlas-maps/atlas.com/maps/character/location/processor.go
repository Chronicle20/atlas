package location

import (
	"context"

	"atlas-maps/data/map/info"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
	Set(characterId uint32, f field.Model) (Model, error)
	Resolve(currentField field.Model) (field.Model, ResolutionReason, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	ip  info.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return newProcessorWithInfo(l, ctx, db, info.NewProcessor(l, ctx))
}

func newProcessorWithInfo(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, ip info.Processor) *ProcessorImpl {
	return &ProcessorImpl{l: l, ctx: ctx, db: db, ip: ip}
}

func (p *ProcessorImpl) Resolve(cur field.Model) (field.Model, ResolutionReason, error) {
	md, err := p.ip.GetById(cur.MapId())
	if err != nil {
		p.l.WithError(err).Warnf("location.Resolve: map info unavailable for [%d]; staying put.", cur.MapId())
		return cur, ReasonStayPut, nil
	}
	if md.ForcedReturnMapId().IsSentinel() {
		return cur, ReasonStayPut, nil
	}
	resolved := field.NewBuilder(cur.WorldId(), cur.ChannelId(), md.ForcedReturnMapId()).SetInstance(uuid.Nil).Build()
	return resolved, ReasonForcedReturn, nil
}

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	panic("not yet implemented")
}

func (p *ProcessorImpl) Set(characterId uint32, f field.Model) (Model, error) {
	panic("not yet implemented")
}
