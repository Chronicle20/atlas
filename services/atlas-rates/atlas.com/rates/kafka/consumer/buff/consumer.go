package buff

import (
	"atlas-rates/character"
	consumer2 "atlas-rates/kafka/consumer"
	"atlas-rates/kafka/message/buff"
	"atlas-rates/rate"
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

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(buff.EnvEventStatusTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBuffApplied))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBuffExpired))); err != nil {
			return err
		}
		return nil
	}
}

func handleBuffApplied(l logrus.FieldLogger, ctx context.Context, e buff.StatusEvent[buff.AppliedStatusEventBody]) {
	if e.Type != buff.EventStatusTypeBuffApplied {
		return
	}

	l.Debugf("Processing buff applied event for character [%d], buff source [%d].", e.CharacterId, e.Body.SourceId)

	p := character.NewProcessor(l, ctx)

	// Process each stat change and add rate factors for rate-affecting changes
	for _, change := range e.Body.Changes {
		mapping, exists := buff.GetRateMapping(change.Type)
		if !exists {
			continue
		}

		rateType := rate.Type(mapping.RateType)
		if rateType == "" {
			continue
		}

		// Convert stat amount to multiplier using the appropriate conversion method
		// HOLY_SYMBOL (additive): amount=50 -> 1.50x (50% bonus)
		// MESO_UP (direct): amount=103 -> 1.03x (103% of base)
		multiplier := buff.CalculateMultiplier(change.Amount, mapping.Conversion)

		l.Debugf("Adding buff factor: stat type [%s] -> rate type [%s], amount [%d] -> multiplier [%.2f].",
			change.Type, rateType, change.Amount, multiplier)

		ch := channel.NewModel(e.WorldId, e.ChannelId)
		if err := p.AddBuffFactor(ch, e.CharacterId, e.Body.SourceId, rateType, multiplier); err != nil {
			l.WithError(err).Errorf("Unable to add buff factor for character [%d].", e.CharacterId)
		}
	}
}

func handleBuffExpired(l logrus.FieldLogger, ctx context.Context, e buff.StatusEvent[buff.ExpiredStatusEventBody]) {
	if e.Type != buff.EventStatusTypeBuffExpired {
		return
	}

	l.Debugf("Processing buff expired event for character [%d], buff source [%d].", e.CharacterId, e.Body.SourceId)

	p := character.NewProcessor(l, ctx)

	// Remove all rate factors from this buff
	if err := p.RemoveAllBuffFactors(e.CharacterId, e.Body.SourceId); err != nil {
		l.WithError(err).Errorf("Unable to remove buff factors for character [%d].", e.CharacterId)
	}
}
