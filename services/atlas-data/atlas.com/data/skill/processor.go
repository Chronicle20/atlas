package skill

import (
	"atlas-data/document"
	"atlas-data/xml"
	"context"

	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Processor interface {
	Register(s *document.Storage[string, RestModel], r model.Provider[Derivation]) (Stats, error)
	RegisterSkill(path string) (Stats, error)
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
	return document.NewStorage(l, db, GetModelRegistry(), "SKILL")
}

func (p *ProcessorImpl) Register(s *document.Storage[string, RestModel], r model.Provider[Derivation]) (Stats, error) {
	d, err := r()
	if err != nil {
		return Stats{}, err
	}
	for _, m := range d.Models {
		_, err = s.Add(p.ctx)(m)()
		if err != nil {
			return Stats{}, err
		}
	}
	return d.Stats, nil
}

func (p *ProcessorImpl) RegisterSkill(path string) (Stats, error) {
	var stats Stats
	err := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		s, err := p.Register(NewStorage(p.l, tx), Read(p.l)(p.ctx)(xml.FromPathProvider(path)))
		stats = s
		return err
	})
	return stats, err
}
