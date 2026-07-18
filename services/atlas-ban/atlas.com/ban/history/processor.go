package history

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	Record(accountId uint32, accountName string, ipAddress string, hwid string, success bool, failureReason string) (Model, error)
	ByAccountIdProvider(accountId uint32, page model.Page) model.Provider[model.Paged[Model]]
	ByIPPagedProvider(ip string, page model.Page) model.Provider[model.Paged[Model]]
	ByHWIDPagedProvider(hwid string, page model.Page) model.Provider[model.Paged[Model]]
	AllProvider(page model.Page) model.Provider[model.Paged[Model]]
	PurgeOlderThan(days int) error
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) Record(accountId uint32, accountName string, ipAddress string, hwid string, success bool, failureReason string) (Model, error) {
	p.l.Debugf("Recording login attempt for account [%d] ip [%s] hwid [%s] success [%t].", accountId, ipAddress, hwid, success)
	m, err := create(p.db.WithContext(p.ctx))(p.t.Id(), accountId, accountName, ipAddress, hwid, success, failureReason)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to record login attempt for account [%d].", accountId)
		return Model{}, err
	}
	return m, nil
}

func (p *ProcessorImpl) ByAccountIdProvider(accountId uint32, page model.Page) model.Provider[model.Paged[Model]] {
	ep := entitiesByAccountId(accountId, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) ByIPPagedProvider(ip string, page model.Page) model.Provider[model.Paged[Model]] {
	ep := entitiesByIP(ip, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) ByHWIDPagedProvider(hwid string, page model.Page) model.Provider[model.Paged[Model]] {
	ep := entitiesByHWID(hwid, page)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) AllProvider(page model.Page) model.Provider[model.Paged[Model]] {
	ep := entitiesByTenant(page)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())
}

func (p *ProcessorImpl) PurgeOlderThan(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	p.l.Debugf("Purging login history older than %d days (before %s).", days, cutoff.Format(time.RFC3339))
	return deleteOlderThan(p.db.WithContext(p.ctx))(cutoff)
}
