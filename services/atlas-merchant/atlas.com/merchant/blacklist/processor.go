package blacklist

import (
	"context"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Add(shopId uuid.UUID, name string) error
	Remove(shopId uuid.UUID, name string) error
	Names(shopId uuid.UUID) ([]string, error)
	IsBlacklisted(shopId uuid.UUID, name string) (bool, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db, t: tenant.MustFromContext(ctx)}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Add(shopId uuid.UUID, name string) error {
	_, err := create(p.t.Id(), shopId, name)(p.db.WithContext(p.ctx))()
	return err
}

func (p *ProcessorImpl) Remove(shopId uuid.UUID, name string) error {
	_, err := deleteByShopIdAndName(shopId, name)(p.db.WithContext(p.ctx))()
	return err
}

func (p *ProcessorImpl) Names(shopId uuid.UUID) ([]string, error) {
	es, err := getByShopId(shopId)(p.db.WithContext(p.ctx))()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(es))
	for _, e := range es {
		names = append(names, e.Name)
	}
	return names, nil
}

func (p *ProcessorImpl) IsBlacklisted(shopId uuid.UUID, name string) (bool, error) {
	return existsByShopIdAndName(shopId, name)(p.db.WithContext(p.ctx))()
}
