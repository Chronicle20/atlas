// Package playernpc consumes EVENT_TOPIC_PLAYER_NPC_STATUS's
// COMMAND_SUCCEEDED/COMMAND_FAILED outcome events (Task 23a) and completes
// the deploy_player_npc step they correlate to (Task 23b, FR-6.2/FR-6.3).
package playernpc

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	playernpcmsg "atlas-saga-orchestrator/kafka/message/playernpc"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/google/uuid"
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
			rf(consumer2.NewConfig(l)("player_npc_status_event")(playernpcmsg.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(playernpcmsg.EnvEventTopicStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandSucceededEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandFailedEvent))); err != nil {
			return err
		}
		return nil
	}
}

// handleCommandSucceededEvent completes a pending deploy_player_npc step. The
// ordinary GM deploy path (atlas-messages) produces commands with uuid.Nil
// (not saga-driven), and DEPLOYED/UPDATED/REMOVED/REPOSITIONED share this
// topic too, so the type check and the nil check both decline any event that
// does not belong to this handler -- the same declining-by-default posture
// the npc conversation status consumer uses.
//
// A redelivered COMMAND_SUCCEEDED for an already-completed step is declined
// by AcceptEvent, which is the idempotency guarantee the at-least-once topic
// needs.
func handleCommandSucceededEvent(l logrus.FieldLogger, ctx context.Context, e playernpcmsg.StatusEvent[playernpcmsg.StatusCommandOutcomeBody]) {
	if e.Type != playernpcmsg.EventTypeCommandSucceeded {
		return
	}
	if e.Body.TransactionId == uuid.Nil {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.Body.TransactionId, saga.EventKindPlayerNpcCommandSucceeded); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"character_id":   e.Body.CharacterId,
		"command_type":   e.Body.CommandType,
	}).Debug("Player npc command succeeded; completing deploy_player_npc step.")

	_ = p.StepCompleted(e.Body.TransactionId, true)
}

// handleCommandFailedEvent fails the step, driving the saga to compensate.
// Code carries the design §8.3 failure code (FR-6.3) -- e.g.
// "pool_exhausted", "map_full", "ineligible" -- onto the step's result via
// the existing errorCode convention (kafka/consumer/character/consumer.go,
// kafka/consumer/skill/consumer.go) so a conversation script can branch on
// it without a new saga plumbing mechanism.
func handleCommandFailedEvent(l logrus.FieldLogger, ctx context.Context, e playernpcmsg.StatusEvent[playernpcmsg.StatusCommandOutcomeBody]) {
	if e.Type != playernpcmsg.EventTypeCommandFailed {
		return
	}
	if e.Body.TransactionId == uuid.Nil {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.Body.TransactionId, saga.EventKindPlayerNpcCommandFailed); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"character_id":   e.Body.CharacterId,
		"command_type":   e.Body.CommandType,
		"code":           e.Body.Code,
		"message":        e.Body.Message,
	}).Warn("Player npc command failed; failing deploy_player_npc step.")

	result := map[string]any{"errorCode": e.Body.Code}
	if e.Body.Message != "" {
		result["errorDetail"] = e.Body.Message
	}
	_ = p.StepCompletedWithResult(e.Body.TransactionId, false, result)
}
