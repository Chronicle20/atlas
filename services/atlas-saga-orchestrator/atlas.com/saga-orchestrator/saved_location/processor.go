package saved_location

import (
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Put(characterId uint32, locationType string, mapId _map.Id, portalId uint32) error
	Get(characterId uint32, locationType string) (RestModel, error)
	Delete(characterId uint32, locationType string) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

func (p *ProcessorImpl) Put(characterId uint32, locationType string, mapId _map.Id, portalId uint32) error {
	_, err := PutSavedLocation(p.l, p.ctx)(characterId, locationType, mapId, portalId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to save location [%s] for character [%d].", locationType, characterId)
		return err
	}
	return nil
}

func (p *ProcessorImpl) Get(characterId uint32, locationType string) (RestModel, error) {
	rm, err := GetSavedLocation(p.l, p.ctx)(characterId, locationType)
	if err != nil {
		p.l.WithError(err).Debugf("Unable to get saved location [%s] for character [%d].", locationType, characterId)
		return RestModel{}, err
	}
	return rm, nil
}

func (p *ProcessorImpl) Delete(characterId uint32, locationType string) error {
	err := DeleteSavedLocation(p.l, p.ctx)(characterId, locationType)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to delete saved location [%s] for character [%d].", locationType, characterId)
		return err
	}
	return nil
}
