// Package playernpc consumes COMMAND_TOPIC_PLAYER_NPC (Task 17): DEPLOY,
// REDEPLOY and REMOVE, each mapped onto playernpc.Processor (Task 15).
package playernpc

import (
	consumer2 "atlas-player-npcs/kafka/consumer"
	msg "atlas-player-npcs/kafka/message/playernpc"
	"atlas-player-npcs/playernpc"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// redeployLookupPageSize bounds the GetByMap page handleRedeploy scans to
// resolve the row id REDEPLOY's (characterId, worldId, mapId) addresses
// (see kafka/message/playernpc/kafka.go's CommandRedeployBody doc). A Hall
// of Fame map's occupancy is bounded by the allocation pool per branch
// (design §4), far below this.
const redeployLookupPageSize = 1000

// ProcessorProvider builds the real, request-scoped playernpc.Processor
// for (l, ctx). main.go wires the HTTP/DB-backed processor (mirroring
// rest.go's processorFor); tests inject a stub so a handler test never
// makes a real HTTP or DB call. Shared with
// kafka/consumer/character, whose LEVEL_CHANGED handler also needs
// Processor.GetByMap for its own-deployed check.
type ProcessorProvider func(l logrus.FieldLogger, ctx context.Context) playernpc.Processor

// OutcomeEmitter publishes one StatusCommandOutcomeBody event (Task 23a)
// reporting a consumed COMMAND_TOPIC_PLAYER_NPC message's outcome. main.go
// wires the Kafka-backed one (producer.go's NewEmitter posture: log at warn
// and swallow a publish failure); consumer_test.go injects a capturing stub
// so a handler test never makes a real broker call. This is deliberately
// not playernpc.NewEmitter / statusEventProvider -- those map *domain*
// events and have no access to the command's correlation fields
// (TransactionId, Requester), which only this consumer holds alongside the
// Processor's error.
type OutcomeEmitter func(e msg.StatusEvent[msg.StatusCommandOutcomeBody])

// NewOutcomeEmitter is the Kafka-backed OutcomeEmitter: it publishes to
// EVENT_TOPIC_PLAYER_NPC_STATUS, keyed by CharacterId
// (producer.CreateKey), same as the domain events playernpc.NewEmitter
// publishes on that topic. A publish failure is logged at warn and
// otherwise swallowed, matching producer.go's documented posture -- the
// command already ran to completion (success or failure) and must not
// block or retry on a broker hiccup.
func NewOutcomeEmitter(l logrus.FieldLogger, ctx context.Context) OutcomeEmitter {
	return func(e msg.StatusEvent[msg.StatusCommandOutcomeBody]) {
		key := producer.CreateKey(int(e.Body.CharacterId))
		provider := producer.SingleMessageProvider(key, &e)
		if err := producer.ProviderImpl(l)(ctx)(msg.EnvEventTopicStatus)(provider); err != nil {
			l.WithError(err).Warnf("Unable to emit Player NPC command outcome event [%s].", e.Type)
		}
	}
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("player_npc_command")(msg.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(pp ProcessorProvider) func(oe OutcomeEmitter) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(pp ProcessorProvider) func(oe OutcomeEmitter) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(oe OutcomeEmitter) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				t, err := topic.EnvProvider(l)(msg.EnvCommandTopic)()
				if err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDeploy(pp, oe)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRedeploy(pp, oe)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRemove(pp, oe)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

// handleDeploy handles DEPLOY, always the eligibility-checked path
// (enforceEligibility is carried on the command, not hardcoded, so the GM
// command (design §9.2, Task 21) can route through the same topic with
// enforceEligibility: false).
func handleDeploy(pp ProcessorProvider, oe OutcomeEmitter) message.Handler[msg.Command[msg.CommandDeployBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c msg.Command[msg.CommandDeployBody]) {
		if c.Type != msg.CommandTypeDeploy {
			return
		}
		var explicit *playernpc.Position
		if c.Body.Position != nil {
			explicit = &playernpc.Position{X: c.Body.Position.X, Y: c.Body.Position.Y}
		}
		_, err := pp(l, ctx).Deploy(c.CharacterId, c.Body.WorldId, c.Body.MapId, c.Body.EnforceEligibility, explicit)
		if err != nil {
			l.WithError(err).Warnf("Unable to deploy Player NPC for character [%d] on map [%d].", c.CharacterId, c.Body.MapId)
		}
		emitOutcome(oe, c.TransactionId, c.CharacterId, c.Requester, msg.CommandTypeDeploy, err)
	}
}

// handleRedeploy resolves the row REDEPLOY addresses via GetByMap (see
// CommandRedeployBody's doc) and calls Processor.Redeploy on it.
func handleRedeploy(pp ProcessorProvider, oe OutcomeEmitter) message.Handler[msg.Command[msg.CommandRedeployBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c msg.Command[msg.CommandRedeployBody]) {
		if c.Type != msg.CommandTypeRedeploy {
			return
		}
		p := pp(l, ctx)
		ms, err := p.GetByMap(c.Body.WorldId, c.Body.MapId, model.Page{Number: 1, Size: redeployLookupPageSize})
		if err != nil {
			l.WithError(err).Warnf("Unable to resolve Player NPC to redeploy for character [%d] on map [%d].", c.CharacterId, c.Body.MapId)
			emitOutcome(oe, c.TransactionId, c.CharacterId, c.Requester, msg.CommandTypeRedeploy, err)
			return
		}
		for _, m := range ms {
			if m.CharacterId() == c.CharacterId {
				_, err := p.Redeploy(m.Id())
				if err != nil {
					l.WithError(err).Warnf("Unable to redeploy Player NPC [%s].", m.Id())
				}
				emitOutcome(oe, c.TransactionId, c.CharacterId, c.Requester, msg.CommandTypeRedeploy, err)
				return
			}
		}
		l.Warnf("No deployed Player NPC found for character [%d] on map [%d] to redeploy.", c.CharacterId, c.Body.MapId)
		emitOutcomeCode(oe, c.TransactionId, c.CharacterId, c.Requester, msg.CommandTypeRedeploy, playernpc.CodeUnresolvable,
			fmt.Sprintf("no deployed Player NPC found for character [%d] on map [%d]", c.CharacterId, c.Body.MapId))
	}
}

func handleRemove(pp ProcessorProvider, oe OutcomeEmitter) message.Handler[msg.Command[msg.CommandRemoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c msg.Command[msg.CommandRemoveBody]) {
		if c.Type != msg.CommandTypeRemove {
			return
		}
		_, err := pp(l, ctx).Remove(c.CharacterId, c.Body.MapId)
		if err != nil {
			l.WithError(err).Warnf("Unable to remove Player NPC(s) for character [%d].", c.CharacterId)
		}
		emitOutcome(oe, c.TransactionId, c.CharacterId, c.Requester, msg.CommandTypeRemove, err)
	}
}

// emitOutcome emits a COMMAND_SUCCEEDED (err == nil) or COMMAND_FAILED
// (Code = playernpc.CodeFor(err), Message = err.Error()) outcome, unless
// nobody is listening (TransactionId == uuid.Nil && Requester == nil), in
// which case it is skipped so the topic does not carry dead traffic.
func emitOutcome(oe OutcomeEmitter, transactionId uuid.UUID, characterId uint32, requester *msg.Requester, commandType string, err error) {
	if err == nil {
		emit(oe, transactionId, characterId, requester, commandType, "", "")
		return
	}
	emit(oe, transactionId, characterId, requester, commandType, playernpc.CodeFor(err), err.Error())
}

// emitOutcomeCode emits a COMMAND_FAILED outcome for a failure that has no
// error value of its own (handleRedeploy's "no deployed Player NPC found"
// fall-through), with an explicitly chosen §8.3 code and message.
func emitOutcomeCode(oe OutcomeEmitter, transactionId uuid.UUID, characterId uint32, requester *msg.Requester, commandType string, code string, message string) {
	emit(oe, transactionId, characterId, requester, commandType, code, message)
}

func emit(oe OutcomeEmitter, transactionId uuid.UUID, characterId uint32, requester *msg.Requester, commandType string, code string, message string) {
	if transactionId == uuid.Nil && requester == nil {
		return
	}
	eventType := msg.EventTypeCommandSucceeded
	if code != "" {
		eventType = msg.EventTypeCommandFailed
	}
	oe(msg.StatusEvent[msg.StatusCommandOutcomeBody]{
		Type: eventType,
		Body: msg.StatusCommandOutcomeBody{
			TransactionId: transactionId,
			CharacterId:   characterId,
			CommandType:   commandType,
			Code:          code,
			Message:       message,
			Requester:     requester,
		},
	})
}
