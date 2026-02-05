package buff

import (
	"atlas-effective-stats/character"
	consumer2 "atlas-effective-stats/kafka/consumer"
	"atlas-effective-stats/kafka/message/buff"
	"atlas-effective-stats/stat"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("buff_status")(buff.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(buff.EnvEventStatusTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleBuffApplied)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleBuffExpired)))
	}
}

func handleBuffApplied(l logrus.FieldLogger, ctx context.Context, e buff.StatusEvent[buff.AppliedStatusEventBody]) {
	if e.Type != buff.EventStatusTypeBuffApplied {
		return
	}

	l.Debugf("Processing buff applied event for character [%d], buff source [%d].", e.CharacterId, e.Body.SourceId)

	p := character.NewProcessor(l, ctx)

	// Convert buff stat changes to stat bonuses
	bonuses := make([]stat.Bonus, 0)
	for _, change := range e.Body.Changes {
		statType, isMultiplier := stat.MapBuffStatType(change.Type)
		if statType == "" {
			l.Debugf("Unknown buff stat type: %s", change.Type)
			continue
		}

		if isMultiplier {
			// Percentage buff (e.g., HYPER_BODY_HP gives 60% = 0.60 multiplier)
			multiplier := float64(change.Amount) / 100.0
			bonuses = append(bonuses, stat.NewMultiplierBonus("", statType, multiplier))
		} else {
			// Flat buff
			bonuses = append(bonuses, stat.NewBonus("", statType, change.Amount))
		}
	}

	if len(bonuses) > 0 {
		ch := channel.NewModel(e.WorldId, e.ChannelId)
		if err := p.AddBuffBonuses(ch, e.CharacterId, e.Body.SourceId, bonuses); err != nil {
			l.WithError(err).Errorf("Unable to add buff bonuses for character [%d].", e.CharacterId)
		}
	}
}

func handleBuffExpired(l logrus.FieldLogger, ctx context.Context, e buff.StatusEvent[buff.ExpiredStatusEventBody]) {
	if e.Type != buff.EventStatusTypeBuffExpired {
		return
	}

	l.Debugf("Processing buff expired event for character [%d], buff source [%d].", e.CharacterId, e.Body.SourceId)

	p := character.NewProcessor(l, ctx)

	// Remove all bonuses from this buff
	if err := p.RemoveBuffBonuses(e.CharacterId, e.Body.SourceId); err != nil {
		l.WithError(err).Errorf("Unable to remove buff bonuses for character [%d].", e.CharacterId)
	}
}
