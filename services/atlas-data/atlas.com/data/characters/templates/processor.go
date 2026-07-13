package templates

import (
	"atlas-data/document"
	"atlas-data/xml"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Register(s *document.Storage[string, RestModel], r model.Provider[[]RestModel]) error
	RegisterCharacterTemplate(path string) error
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
	return document.NewStorage(l, db, GetModelRegistry(), "CHARACTER_TEMPLATE")
}

func (p *ProcessorImpl) Register(s *document.Storage[string, RestModel], r model.Provider[[]RestModel]) error {
	ms, err := r()
	if err != nil {
		return err
	}
	for _, m := range ms {
		_, err = s.Add(p.ctx)(m)()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) RegisterCharacterTemplate(path string) error {
	return p.Register(NewStorage(p.l, p.db), Read(p.l)(xml.FromPathProvider(path)))
}
