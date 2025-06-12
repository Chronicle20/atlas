package wallet

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	ByAccountIdProvider(accountId uint32) model.Provider[Model]
	GetByAccountId(accountId uint32) (Model, error)
	Create(accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error)
	Update(accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error)
	Delete(accountId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
	return p
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
	}
}

func (p *ProcessorImpl) ByAccountIdProvider(accountId uint32) model.Provider[Model] {
	return model.Map(Make)(byAccountIdEntityProvider(p.t.Id(), accountId)(p.db))
}

func (p *ProcessorImpl) GetByAccountId(accountId uint32) (Model, error) {
	return p.ByAccountIdProvider(accountId)()
}

func (p *ProcessorImpl) Create(accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
	p.l.Debugf("Initializing wallet information for account [%d]. Credit [%d], Points [%d], and Prepaid [%d].", accountId, credit, points, prepaid)
	c, err := createEntity(p.db, p.t, accountId, credit, points, prepaid)
	if err != nil {
		p.l.WithError(err).Errorf("Could not create wallet information for account [%d].", accountId)
		return Model{}, err
	}
	return c, err
}

func (p *ProcessorImpl) Update(accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
	p.l.Debugf("Updating wallet information for account [%d]. Credit [%d], Points [%d], and Prepaid [%d].", accountId, credit, points, prepaid)
	c, err := updateEntity(p.db, p.t, accountId, credit, points, prepaid)
	if err != nil {
		p.l.WithError(err).Errorf("Could not update wallet information for account [%d].", accountId)
		return Model{}, err
	}
	return c, err
}

func (p *ProcessorImpl) Delete(accountId uint32) error {
	p.l.Debugf("Account [%d] was deleted. Cleaning up wallet information...", accountId)
	return deleteEntity(p.ctx)(p.db, p.t.Id(), accountId)
}
