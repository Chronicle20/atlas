package character

import (
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

type Processor interface {
	GetById(characterId uint32) (Model, error)
	AwardExperience(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error
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

func (p *ProcessorImpl) AwardExperience(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(awardExperienceCommandProvider(characterId, ch, white, amount, party))
}
