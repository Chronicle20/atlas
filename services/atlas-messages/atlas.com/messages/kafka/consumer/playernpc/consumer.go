// Package playernpc consumes EVENT_TOPIC_PLAYER_NPC_STATUS's
// COMMAND_SUCCEEDED/COMMAND_FAILED outcome events and reports the real
// result of a "@playernpc add/remove" GM command back to the invoking GM
// (FR-8.3), replacing the accept-only pink text the command emits
// synchronously in command/playernpc.
//
// This is atlas-messages's first non-message.InitConsumers consumer, so it
// follows two existing patterns rather than inventing a third:
// consumer/npcconversation in atlas-saga-orchestrator for the two-handler
// status shape and the declining-by-default posture, and this module's own
// kafka/consumer/message package for the local InitConsumers/InitHandlers
// signatures and consumer2.NewConfig usage.
package playernpc

import (
	consumer2 "atlas-messages/kafka/consumer"
	msg "atlas-messages/kafka/message/playernpc"
	message2 "atlas-messages/message"
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("player_npc_status_event")(msg.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(msg.EnvEventTopicStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandSucceeded))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCommandFailed))); err != nil {
			return err
		}
		return nil
	}
}

// requesterField builds the field.Model an outcome's pink text is issued
// against from the Requester the invoking GM command set. It carries no
// instance -- the GM command never sets one -- matching field.Model's zero
// Instance for a non-instanced field.
func requesterField(r msg.Requester) field.Model {
	return field.NewBuilder(world.Id(r.WorldId), channel.Id(r.ChannelId), _map.Id(r.MapId)).Build()
}

// pinkTextSender reports a result string to the requester's field. Production
// wires message.NewProcessor(l, ctx).IssuePinkText (the same call
// command/playernpc's pink closure makes); tests capture the text instead.
type pinkTextSender func(f field.Model, text string, recipients []uint32) error

// handleCommandSucceeded reports a completed DEPLOY/REMOVE back to the
// invoking GM. e.Body.Requester == nil means the outcome belongs to the
// saga-driven auto-deploy path (atlas-npc-conversations), not a GM command
// -- decline it, the same declining-by-default posture the npcconversation
// status consumer uses.
func handleCommandSucceeded(l logrus.FieldLogger, ctx context.Context, e msg.StatusEvent[msg.StatusCommandOutcomeBody]) {
	pink := func(f field.Model, text string, recipients []uint32) error {
		return message2.NewProcessor(l, ctx).IssuePinkText(f, 0, text, recipients)
	}
	handleCommandSucceededWithDeps(l, e, pink)
}

// handleCommandSucceededWithDeps is handleCommandSucceeded's logic, factored
// out so a test can inject a stub pink-text sender instead of a real
// IssuePinkText call.
func handleCommandSucceededWithDeps(l logrus.FieldLogger, e msg.StatusEvent[msg.StatusCommandOutcomeBody], pink pinkTextSender) {
	if e.Type != msg.EventTypeCommandSucceeded {
		return
	}
	if e.Body.Requester == nil {
		return
	}

	f := requesterField(*e.Body.Requester)
	text := fmt.Sprintf("Player NPC %s succeeded for character %d.", commandTypeLabel(e.Body.CommandType), e.Body.CharacterId)
	if err := pink(f, text, []uint32{e.Body.Requester.CharacterId}); err != nil {
		l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": e.Body.TransactionId.String(),
			"character_id":   e.Body.CharacterId,
			"command_type":   e.Body.CommandType,
		}).Warn("Unable to report Player NPC command outcome to requester.")
	}
}

// handleCommandFailed reports a failed DEPLOY/REMOVE back to the invoking
// GM, naming the specific reason (FR-8.3) rather than the bare code.
func handleCommandFailed(l logrus.FieldLogger, ctx context.Context, e msg.StatusEvent[msg.StatusCommandOutcomeBody]) {
	pink := func(f field.Model, text string, recipients []uint32) error {
		return message2.NewProcessor(l, ctx).IssuePinkText(f, 0, text, recipients)
	}
	handleCommandFailedWithDeps(l, e, pink)
}

// handleCommandFailedWithDeps is handleCommandFailed's logic, factored out
// for the same testability reason as handleCommandSucceededWithDeps.
func handleCommandFailedWithDeps(l logrus.FieldLogger, e msg.StatusEvent[msg.StatusCommandOutcomeBody], pink pinkTextSender) {
	if e.Type != msg.EventTypeCommandFailed {
		return
	}
	if e.Body.Requester == nil {
		return
	}

	f := requesterField(*e.Body.Requester)
	text := fmt.Sprintf("Player NPC %s failed for character %d: %s.", commandTypeLabel(e.Body.CommandType), e.Body.CharacterId, failureReason(e.Body.Code))
	if err := pink(f, text, []uint32{e.Body.Requester.CharacterId}); err != nil {
		l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": e.Body.TransactionId.String(),
			"character_id":   e.Body.CharacterId,
			"command_type":   e.Body.CommandType,
			"code":           e.Body.Code,
		}).Warn("Unable to report Player NPC command outcome to requester.")
	}
}

// commandTypeLabel renders a Command.Type for the pink text; an
// unrecognised value (there should be none -- CommandType only ever mirrors
// what this service itself published) still names itself rather than
// producing an empty label.
func commandTypeLabel(commandType string) string {
	switch commandType {
	case msg.CommandTypeDeploy:
		return "deployment"
	case msg.CommandTypeRemove:
		return "removal"
	default:
		return commandType
	}
}

// failureReason maps each design §8.3 code to a distinct, human-readable
// sentence fragment -- FR-8.3 requires the GM to learn the specific reason,
// not the raw code string.
func failureReason(code string) string {
	switch code {
	case "pool_exhausted":
		return "no usable script id remains in the branch"
	case "map_full":
		return "the map has no free position at the maximum step"
	case "duplicate":
		return "the character already has a Player NPC deployed on that map"
	case "ineligible":
		return "the character does not meet the level/GM eligibility check"
	case "unresolvable":
		return "the character or map could not be resolved"
	case "internal":
		return "an internal error occurred"
	default:
		return fmt.Sprintf("unrecognised failure code %q", code)
	}
}
