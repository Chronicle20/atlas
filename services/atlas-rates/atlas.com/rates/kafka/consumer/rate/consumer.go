package rate

import (
	"atlas-rates/character"
	consumer2 "atlas-rates/kafka/consumer"
	rateMsg "atlas-rates/kafka/message/rate"
	"atlas-rates/rate"
	"context"

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
			rf(consumer2.NewConfig(l)("world_rate")(rateMsg.EnvEventTopicWorldRate)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string

		t, _ = topic.EnvProvider(l)(rateMsg.EnvEventTopicWorldRate)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleWorldRateChanged)))
	}
}

func handleWorldRateChanged(l logrus.FieldLogger, ctx context.Context, e rateMsg.WorldRateEvent) {
	if e.Type != rateMsg.TypeRateChanged {
		return
	}

	l.Debugf("Processing world rate change event for world [%d], rate type [%s], multiplier [%.2f].", e.WorldId, e.RateType, e.Multiplier)

	rateType := msgRateTypeToRateType(e.RateType)
	if rateType == "" {
		l.Warnf("Unknown rate type [%s].", e.RateType)
		return
	}

	p := character.NewProcessor(l, ctx)
	p.UpdateWorldRate(e.WorldId, rateType, e.Multiplier)
}

func msgRateTypeToRateType(rt rateMsg.RateType) rate.Type {
	switch rt {
	case rateMsg.RateTypeExp:
		return rate.TypeExp
	case rateMsg.RateTypeMeso:
		return rate.TypeMeso
	case rateMsg.RateTypeItemDrop:
		return rate.TypeItemDrop
	case rateMsg.RateTypeQuestExp:
		return rate.TypeQuestExp
	default:
		return ""
	}
}
