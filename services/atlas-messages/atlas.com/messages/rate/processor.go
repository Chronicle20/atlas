package rate

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetByCharacter(ch channel.Model, characterId uint32) (Model, error)
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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetByCharacter(ch channel.Model, characterId uint32) (Model, error) {
	rp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacter(ch, characterId), Extract)
	return model.Map(model.Decorate[Model](nil))(rp)()
}
