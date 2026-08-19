package equipslot

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	Extend(characterId uint32, slotIndex int16, period time.Duration) (time.Time, error)
	GetActive(characterId uint32) ([]Model, error)
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

func (p *ProcessorImpl) Extend(characterId uint32, slotIndex int16, period time.Duration) (time.Time, error) {
	expiresAt, err := Extend(p.db.WithContext(p.ctx), p.t.Id(), characterId, slotIndex, period)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to extend equip slot [%d] for character [%d].", slotIndex, characterId)
		return time.Time{}, err
	}
	p.l.Debugf("Extended equip slot [%d] for character [%d] to [%s].", slotIndex, characterId, expiresAt)
	return expiresAt, nil
}

func (p *ProcessorImpl) GetActive(characterId uint32) ([]Model, error) {
	ms, err := GetActive(p.db.WithContext(p.ctx), p.t.Id(), characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get active equip slot extensions for character [%d].", characterId)
		return nil, err
	}
	return ms, nil
}
