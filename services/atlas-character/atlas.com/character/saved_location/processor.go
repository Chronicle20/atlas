package saved_location

import (
	"context"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Put(m Model) (Model, error)
	Get(characterId uint32, locationType string) (Model, error)
	Delete(characterId uint32, locationType string) error
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

func (p *ProcessorImpl) Put(m Model) (Model, error) {
	result, err := upsert(p.db, p.t.Id(), m)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to upsert saved location [%s] for character [%d].", m.LocationType(), m.CharacterId())
		return Model{}, err
	}
	p.l.Debugf("Saved location [%s] for character [%d] at map [%d] portal [%d].", m.LocationType(), m.CharacterId(), m.MapId(), m.PortalId())
	return result, nil
}

func (p *ProcessorImpl) Get(characterId uint32, locationType string) (Model, error) {
	m, err := getByCharacterIdAndType(p.db, p.t.Id(), characterId, locationType)
	if err != nil {
		p.l.WithError(err).Debugf("Unable to get saved location [%s] for character [%d].", locationType, characterId)
		return Model{}, err
	}
	return m, nil
}

func (p *ProcessorImpl) Delete(characterId uint32, locationType string) error {
	err := deleteByCharacterIdAndType(p.db, p.t.Id(), characterId, locationType)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to delete saved location [%s] for character [%d].", locationType, characterId)
		return err
	}
	p.l.Debugf("Deleted saved location [%s] for character [%d].", locationType, characterId)
	return nil
}
