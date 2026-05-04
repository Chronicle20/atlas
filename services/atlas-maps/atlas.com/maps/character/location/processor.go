package location

import (
	"context"
	"time"

	"atlas-maps/data/map/info"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	_, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(p.ctx, "Location.Resolve")
	defer span.End()

	t := tenant.MustFromContext(p.ctx)
	span.SetAttributes(
		attribute.Int("current.map.id", int(cur.MapId())),
		attribute.String("tenant.id", t.Id().String()),
	)

	md, err := p.ip.GetById(cur.MapId())
	if err != nil {
		p.l.WithError(err).Warnf("location.Resolve: map info unavailable for [%d]; staying put.", cur.MapId())
		span.SetAttributes(attribute.String("resolution.reason", string(ReasonStayPut)))
		locationResolutionsTotal.WithLabelValues(string(ReasonStayPut)).Inc()
		return cur, ReasonStayPut, nil
	}
	span.SetAttributes(attribute.Int("forced.return.map.id", int(md.ForcedReturnMapId())))
	if md.ForcedReturnMapId().IsSentinel() {
		span.SetAttributes(attribute.String("resolution.reason", string(ReasonStayPut)))
		locationResolutionsTotal.WithLabelValues(string(ReasonStayPut)).Inc()
		return cur, ReasonStayPut, nil
	}
	resolved := field.NewBuilder(cur.WorldId(), cur.ChannelId(), md.ForcedReturnMapId()).SetInstance(uuid.Nil).Build()
	span.SetAttributes(attribute.String("resolution.reason", string(ReasonForcedReturn)))
	locationResolutionsTotal.WithLabelValues(string(ReasonForcedReturn)).Inc()
	return resolved, ReasonForcedReturn, nil
}

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	t := tenant.MustFromContext(p.ctx)
	var e entity
	if err := p.db.WithContext(p.ctx).
		Where("tenant_id = ? AND character_id = ?", t.Id(), characterId).
		First(&e).Error; err != nil {
		return Model{}, err
	}
	return NewBuilder(e.CharacterId).
		SetWorldId(e.WorldId).
		SetChannelId(e.ChannelId).
		SetMapId(e.MapId).
		SetInstance(e.Instance).
		Build(), nil
}

func (p *ProcessorImpl) Set(characterId uint32, f field.Model) (Model, error) {
	t := tenant.MustFromContext(p.ctx)
	e := entity{
		TenantId:    t.Id(),
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		UpdatedAt:   time.Now(),
	}
	if err := p.db.WithContext(p.ctx).Save(&e).Error; err != nil {
		return Model{}, err
	}
	return NewBuilder(e.CharacterId).SetField(f).Build(), nil
}
