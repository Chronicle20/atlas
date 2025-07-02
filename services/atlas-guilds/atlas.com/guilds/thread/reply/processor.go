package reply

import (
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Add(threadId uint32, posterId uint32, message string) (Model, error)
	Delete(threadId uint32, replyId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) Add(threadId uint32, posterId uint32, message string) (Model, error) {
	return create(p.db, p.t.Id(), threadId, posterId, message)
}

func (p *ProcessorImpl) Delete(threadId uint32, replyId uint32) error {
	return remove(p.db, p.t.Id(), threadId, replyId)
}
