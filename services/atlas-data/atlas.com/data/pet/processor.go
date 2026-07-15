package pet

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"atlas-data/document"
	"atlas-data/xml"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Register(s *document.Storage[string, RestModel], r model.Provider[RestModel]) error
	RegisterPet(path string) error
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

func NewStorage(l logrus.FieldLogger, db *gorm.DB) *document.Storage[string, RestModel] {
	return document.NewStorage(l, db, GetModelRegistry(), "PET")
}

func (p *ProcessorImpl) Register(s *document.Storage[string, RestModel], r model.Provider[RestModel]) error {
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

func (p *ProcessorImpl) RegisterPet(path string) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return p.Register(NewStorage(p.l, tx), Read(p.l)(p.ctx)(xml.FromPathProvider(path)))
	})
}
