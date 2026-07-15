package commodity

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
	RegisterCommodity(path string) error
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
	return document.NewStorage(l, db, GetModelRegistry(), "COMMODITY")
}

// Register adds each item via the storage's per-call commit. No outer
// transaction wraps the loop — a connection drop or single-row failure now
// preserves successfully-committed rows so a retry can converge.
//
// See task-076 F2: the prior outer ExecuteTransaction wrapped the entire
// Etc.wz import in one long-lived transaction and was fatal to any conn
// blip across a multi-thousand-row register.
func (p *ProcessorImpl) Register(s *document.Storage[string, RestModel], r model.Provider[[]RestModel]) error {
	ms, err := r()
	if err != nil {
		return err
	}
	for _, m := range ms {
		if _, err := s.Add(p.ctx)(m)(); err != nil {
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) RegisterCommodity(path string) error {
	return p.Register(NewStorage(p.l, p.db), Read(p.l)(xml.FromPathProvider(path)))
}
