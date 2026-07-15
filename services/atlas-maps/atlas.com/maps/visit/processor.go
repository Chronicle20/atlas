package visit

import (
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	RecordVisit(characterId uint32, mapId _map.Id) error
	ByCharacterIdProvider(characterId uint32, page model.Page) model.Provider[model.Paged[Visit]]
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) RecordVisit(characterId uint32, mapId _map.Id) error {
	t := tenant.MustFromContext(p.ctx)
	return recordVisit(p.db.WithContext(p.ctx))(t.Id())(characterId)(mapId)
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32, page model.Page) model.Provider[model.Paged[Visit]] {
	ep := database.PagedQuery[Entity](p.db.WithContext(p.ctx).Where("character_id = ?", characterId), page)
	return model.MapPaged[Entity, Visit](Make)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) ByCharacterIdAndMapIdProvider(characterId uint32, mapId _map.Id) model.Provider[Visit] {
	return model.Map[Entity, Visit](Make)(getByCharacterIdAndMapIdProvider(characterId)(mapId)(p.db.WithContext(p.ctx)))
}

func (p *ProcessorImpl) DeleteByCharacterId(characterId uint32) (int64, error) {
	return deleteByCharacterId(p.db.WithContext(p.ctx))(characterId)
}
