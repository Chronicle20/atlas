// Package playernpc consumes COMMAND_TOPIC_PLAYER_NPC (Task 17): DEPLOY,
// REDEPLOY and REMOVE, each mapped onto playernpc.Processor (Task 15).
package playernpc

import (
	consumer2 "atlas-player-npcs/kafka/consumer"
	msg "atlas-player-npcs/kafka/message/playernpc"
	"atlas-player-npcs/playernpc"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
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

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("player_npc_command")(msg.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(pp ProcessorProvider) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(pp ProcessorProvider) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			t, _ := topic.EnvProvider(l)(msg.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDeploy(pp)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRedeploy(pp)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRemove(pp)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handleDeploy handles DEPLOY, always the eligibility-checked path
// (enforceEligibility is carried on the command, not hardcoded, so the GM
// command (design §9.2, Task 21) can route through the same topic with
// enforceEligibility: false).
func handleDeploy(pp ProcessorProvider) message.Handler[msg.Command[msg.CommandDeployBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c msg.Command[msg.CommandDeployBody]) {
		if c.Type != msg.CommandTypeDeploy {
			return
		}
		var explicit *playernpc.Position
		if c.Body.Position != nil {
			explicit = &playernpc.Position{X: c.Body.Position.X, Y: c.Body.Position.Y}
		}
		if _, err := pp(l, ctx).Deploy(c.CharacterId, c.Body.WorldId, c.Body.MapId, c.Body.EnforceEligibility, explicit); err != nil {
			l.WithError(err).Warnf("Unable to deploy Player NPC for character [%d] on map [%d].", c.CharacterId, c.Body.MapId)
		}
	}
}

// handleRedeploy resolves the row REDEPLOY addresses via GetByMap (see
// CommandRedeployBody's doc) and calls Processor.Redeploy on it.
func handleRedeploy(pp ProcessorProvider) message.Handler[msg.Command[msg.CommandRedeployBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c msg.Command[msg.CommandRedeployBody]) {
		if c.Type != msg.CommandTypeRedeploy {
			return
		}
		p := pp(l, ctx)
		ms, err := p.GetByMap(c.Body.WorldId, c.Body.MapId, model.Page{Number: 1, Size: redeployLookupPageSize})
		if err != nil {
			l.WithError(err).Warnf("Unable to resolve Player NPC to redeploy for character [%d] on map [%d].", c.CharacterId, c.Body.MapId)
			return
		}
		for _, m := range ms {
			if m.CharacterId() == c.CharacterId {
				if _, err := p.Redeploy(m.Id()); err != nil {
					l.WithError(err).Warnf("Unable to redeploy Player NPC [%s].", m.Id())
				}
				return
			}
		}
		l.Warnf("No deployed Player NPC found for character [%d] on map [%d] to redeploy.", c.CharacterId, c.Body.MapId)
	}
}

func handleRemove(pp ProcessorProvider) message.Handler[msg.Command[msg.CommandRemoveBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c msg.Command[msg.CommandRemoveBody]) {
		if c.Type != msg.CommandTypeRemove {
			return
		}
		if _, err := pp(l, ctx).Remove(c.CharacterId, c.Body.MapId); err != nil {
			l.WithError(err).Warnf("Unable to remove Player NPC(s) for character [%d].", c.CharacterId)
		}
	}
}
