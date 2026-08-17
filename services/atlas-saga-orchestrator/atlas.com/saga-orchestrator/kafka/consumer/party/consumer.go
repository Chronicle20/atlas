package party

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	party2 "atlas-saga-orchestrator/kafka/message/party"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("party_status_event")(party2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(party2.EnvStatusEventTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handlePartyLeftEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handlePartyDisbandEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handlePartyLeftEvent(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.StatusEventLeftBody]) {
	if e.Type != party2.StatusEventTypeLeft {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindPartyMemberLeft, saga.ForCharacter(e.ActorId)); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}

// handlePartyDisbandEvent handles the alternate outcome of a LEAVE command:
// atlas-parties disbands the party (rather than emitting LEFT) when the
// leaving character is the party leader. Without this, a world transfer of a
// party leader would hang to SAGA_TIMEOUT even though the severance
// succeeded (see kafka/message/party/kafka.go doc comment on
// StatusEventDisbandBody).
func handlePartyDisbandEvent(l logrus.FieldLogger, ctx context.Context, e party2.StatusEvent[party2.StatusEventDisbandBody]) {
	if e.Type != party2.StatusEventTypeDisband {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindPartyDisbanded, saga.ForCharacter(e.ActorId)); !ok {
		return
	}
	_ = p.StepCompleted(e.TransactionId, true)
}
