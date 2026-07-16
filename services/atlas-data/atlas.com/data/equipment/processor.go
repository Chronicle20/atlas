package equipment

import (
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"atlas-data/document"
	"atlas-data/item"
	"atlas-data/xml"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Register(tx *gorm.DB, s *document.Storage[string, RestModel], r model.Provider[RestModel]) error
	RegisterEquipment(path string) error
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
	return document.NewStorage(l, db, GetModelRegistry(), "EQUIPMENT")
}

func (p *ProcessorImpl) Register(tx *gorm.DB, s *document.Storage[string, RestModel], r model.Provider[RestModel]) error {
	m, err := r()
	if err != nil {
		return err
	}
	if _, err = s.Add(p.ctx)(m)(); err != nil {
		return err
	}
	return item.UpdateEquipmentClassification(tx, p.ctx, m.Id, m.ReqJob, m.Cash)
}

func (p *ProcessorImpl) RegisterEquipment(path string) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return p.Register(tx, NewStorage(p.l, tx), Read(p.l)(xml.FromPathProvider(path)))
	})
}
