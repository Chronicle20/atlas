package history

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Record(accountId uint32, accountName string, ipAddress string, hwid string, success bool, failureReason string) (Model, error)
	GetByAccountId(accountId uint32) ([]Model, error)
	GetByIP(ip string) ([]Model, error)
	GetByHWID(hwid string) ([]Model, error)
	GetByTenant() ([]Model, error)
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

func (p *ProcessorImpl) Record(accountId uint32, accountName string, ipAddress string, hwid string, success bool, failureReason string) (Model, error) {
	p.l.Debugf("Recording login attempt for account [%d] ip [%s] hwid [%s] success [%t].", accountId, ipAddress, hwid, success)
	m, err := create(p.db.WithContext(p.ctx))(p.t.Id(), accountId, accountName, ipAddress, hwid, success, failureReason)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to record login attempt for account [%d].", accountId)
		return Model{}, err
	}
	return m, nil
}

func (p *ProcessorImpl) GetByAccountId(accountId uint32) ([]Model, error) {
	return model.SliceMap(Make)(entitiesByAccountId(accountId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetByIP(ip string) ([]Model, error) {
	return model.SliceMap(Make)(entitiesByIP(ip)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetByHWID(hwid string) ([]Model, error) {
	return model.SliceMap(Make)(entitiesByHWID(hwid)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetByTenant() ([]Model, error) {
	return model.SliceMap(Make)(entitiesByTenant()(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) PurgeOlderThan(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	p.l.Debugf("Purging login history older than %d days (before %s).", days, cutoff.Format(time.RFC3339))
	return deleteOlderThan(p.db.WithContext(p.ctx))(cutoff)
}
