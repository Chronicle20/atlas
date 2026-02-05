package buff

import (
	"atlas-messages/data/skill"
	"atlas-messages/kafka/message/buff"
	"atlas-messages/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Apply(f field.Model, characterId uint32, fromId uint32, skillId uint32, level byte, durationOverride int32) error
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

func (p *ProcessorImpl) Apply(f field.Model, characterId uint32, fromId uint32, skillId uint32, level byte, durationOverride int32) error {
	sdp := skill.NewProcessor(p.l, p.ctx)

	effect, err := sdp.GetEffect(skillId, level)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get skill effect for skill [%d] level [%d].", skillId, level)
		return err
	}

	duration := effect.Duration()
	if durationOverride > 0 {
		duration = durationOverride * 1000
	}

	statups := effect.StatUps()
	changes := make([]buff.StatChange, 0, len(statups))
	for _, su := range statups {
		changes = append(changes, buff.StatChange{
			Type:   su.Mask(),
			Amount: su.Amount(),
		})
	}

	return producer.ProviderImpl(p.l)(p.ctx)(buff.EnvCommandTopic)(buff.ApplyCommandProvider(f, characterId, fromId, int32(skillId), duration, changes))
}
