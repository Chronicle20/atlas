package area_info

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	Put(m Model) (Model, error)
	GetByArea(characterId uint32, area uint16) (Model, error)
	GetByAreaAsString(characterId uint32, area uint16) (string, error)
	GetAll(characterId uint32) ([]Model, error)
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

func (p *ProcessorImpl) Put(m Model) (Model, error) {
	result, err := upsert(p.db.WithContext(p.ctx), p.t.Id(), m)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to upsert area info [%d] for character [%d].", m.Area(), m.CharacterId())
		return Model{}, err
	}
	p.l.Debugf("Set area info [%d] for character [%d] to [%s].", m.Area(), m.CharacterId(), m.Info())
	return result, nil
}

func (p *ProcessorImpl) GetByArea(characterId uint32, area uint16) (Model, error) {
	m, err := getByCharacterIdAndArea(p.db.WithContext(p.ctx), characterId, area)
	if err != nil {
		p.l.WithError(err).Debugf("Unable to get area info [%d] for character [%d].", area, characterId)
		return Model{}, err
	}
	return m, nil
}

// GetByAreaAsString returns the stored area-info string for area, or "" when
// unset — Cosmic's area_info.get(area) on a missing key is handled as "does
// not contain" rather than an error.
func (p *ProcessorImpl) GetByAreaAsString(characterId uint32, area uint16) (string, error) {
	m, err := p.GetByArea(characterId, area)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return m.Info(), nil
}

func (p *ProcessorImpl) GetAll(characterId uint32) ([]Model, error) {
	ms, err := getAllByCharacterId(p.db.WithContext(p.ctx), characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get area info for character [%d].", characterId)
		return nil, err
	}
	return ms, nil
}
