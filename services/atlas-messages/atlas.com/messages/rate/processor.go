package rate

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetByCharacter(worldId byte, channelId byte, characterId uint32) (Model, error)
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

func (p *ProcessorImpl) GetByCharacter(worldId byte, channelId byte, characterId uint32) (Model, error) {
	rp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacter(worldId, channelId, characterId), Extract)
	return model.Map(model.Decorate[Model](nil))(rp)()
}
