package visit

import (
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	RecordVisit(characterId uint32, mapId _map.Id) error
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Visit]
	ByCharacterIdAndMapIdProvider(characterId uint32, mapId _map.Id) model.Provider[Visit]
	DeleteByCharacterId(characterId uint32) (int64, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

func (p *ProcessorImpl) RecordVisit(characterId uint32, mapId _map.Id) error {
	t := tenant.MustFromContext(p.ctx)
	return recordVisit(p.db.WithContext(p.ctx))(t.Id())(characterId)(mapId)
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Visit] {
	return model.SliceMap[Entity, Visit](Make)(getByCharacterIdProvider(characterId)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

func (p *ProcessorImpl) ByCharacterIdAndMapIdProvider(characterId uint32, mapId _map.Id) model.Provider[Visit] {
	return model.Map[Entity, Visit](Make)(getByCharacterIdAndMapIdProvider(characterId)(mapId)(p.db.WithContext(p.ctx)))
}

func (p *ProcessorImpl) DeleteByCharacterId(characterId uint32) (int64, error) {
	return deleteByCharacterId(p.db.WithContext(p.ctx))(characterId)
}
