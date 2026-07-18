package monster

import (
	"atlas-data/xml"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Processor interface {
	Register(s *Storage, r model.Provider[RestModel]) error
	RegisterMonster(path string) error
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

func (p *ProcessorImpl) Register(s *Storage, r model.Provider[RestModel]) error {
	m, err := r()
	if err != nil {
		return err
	}
	_, err = s.Add(p.ctx)(m)()
	if err != nil {
		return err
	}
	return nil
}

func (p *ProcessorImpl) RegisterMonster(path string) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return p.Register(NewStorage(p.l, tx), Read(p.l)(p.ctx)(xml.FromPathProvider(path)))
	})
}
