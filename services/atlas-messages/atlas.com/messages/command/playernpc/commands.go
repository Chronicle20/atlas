// Package playernpc implements the design §9.2 GM commands: "@playernpc
// add <name>" deploys a named online character's Player NPC at the
// invoking GM's current position and map, bypassing the level and
// auto-deploy checks; "@playernpc remove <name> [here]" removes every (or,
// with "here", the current-map) Player NPC belonging to a named character.
// Both commands emit onto COMMAND_TOPIC_PLAYER_NPC (Task 17's shipped
// contract, kafka/message/playernpc/kafka.go) and report the outcome back
// to the invoking GM via pink text.
package playernpc

import (
	"atlas-messages/character"
	"atlas-messages/command"
	msg "atlas-messages/kafka/message/playernpc"
	"atlas-messages/message"
	"context"
	"fmt"
	"regexp"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

var (
	deployRe = regexp.MustCompile(`^@playernpc\s+add\s+(\S+)$`)
	removeRe = regexp.MustCompile(`^@playernpc\s+remove\s+(\S+)(?:\s+(here))?$`)
)

// characterLookup resolves a named character to its model. Production
// wires character.NewProcessor(l, ctx).GetByName(); tests inject a stub so
// a handler test never makes a real HTTP call.
type characterLookup func(name string) (character.Model, error)

// commandPublisher emits one COMMAND_TOPIC_PLAYER_NPC message. Production
// wires producer.ProviderImpl(l)(ctx)(msg.EnvCommandTopic); tests capture
// the provider's messages instead of touching a broker.
type commandPublisher func(provider model.Provider[[]kafka.Message]) error

// pinkTextSender reports a result string to the invoking GM. Production
// wires message.NewProcessor(l, ctx).IssuePinkText; tests capture the
// text.
type pinkTextSender func(text string) error

// DeployCommandProducer matches "@playernpc add <name>". It resolves name
// and emits a DEPLOY command at execution time (not match time), so the
// Executor's own logger/context -- not the producer's -- back every
// character lookup, Kafka publish and pink-text call, per the existing
// command/monster pattern.
func DeployCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			match := deployRe.FindStringSubmatch(m)
			if match == nil {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			targetName := match[1]

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					cp := character.NewProcessor(l, ctx)
					lookup := func(name string) (character.Model, error) { return cp.GetByName()(name) }
					publish := func(p model.Provider[[]kafka.Message]) error {
						return producer.ProviderImpl(l)(ctx)(msg.EnvCommandTopic)(p)
					}
					msgProc := message.NewProcessor(l, ctx)
					pink := func(text string) error { return msgProc.IssuePinkText(f, 0, text, []uint32{c.Id()}) }
					return deployWithDeps(f, c, targetName, lookup, publish, pink)
				}
			}, true
		}
	}
}

// deployWithDeps is DeployCommandProducer's execution logic, factored out
// so a test can inject stub lookup/publish/pink functions instead of a
// real character HTTP client, Kafka broker and IssuePinkText call.
func deployWithDeps(f field.Model, c character.Model, targetName string, lookup characterLookup, publish commandPublisher, pink pinkTextSender) error {
	target, err := lookup(targetName)
	if err != nil {
		return pink(fmt.Sprintf("Character not found: %s.", targetName))
	}

	key := producer.CreateKey(int(target.Id()))
	value := msg.Command[msg.CommandDeployBody]{
		CharacterId: target.Id(),
		Type:        msg.CommandTypeDeploy,
		Requester: &msg.Requester{
			CharacterId: c.Id(),
			WorldId:     byte(f.WorldId()),
			ChannelId:   byte(f.ChannelId()),
			MapId:       uint32(f.MapId()),
		},
		Body: msg.CommandDeployBody{
			WorldId:            f.WorldId(),
			MapId:              f.MapId(),
			Position:           &msg.CommandPosition{X: c.X(), Y: c.Y()},
			EnforceEligibility: false,
		},
	}
	if err := publish(producer.SingleMessageProvider(key, value)); err != nil {
		return pink(fmt.Sprintf("Failed to deploy Player NPC for %s: %s.", target.Name(), err.Error()))
	}
	return pink(fmt.Sprintf("Accepted Player NPC deployment for %s; you will be notified of the result.", target.Name()))
}

// RemoveCommandProducer matches "@playernpc remove <name> [here]". Bulk
// removal (FR-8.2) is Processor.Remove in atlas-player-npcs -- N
// transactions and N emits, not one atomic operation -- so a single
// "remove" invocation may produce more than one COMMAND_SUCCEEDED/FAILED
// outcome event; kafka/consumer/playernpc pink-texts one line per event, so
// a bulk removal that fails partway through reports each occupant's result
// individually rather than one atomic all-or-nothing outcome. The pink text
// below, emitted synchronously here, is worded as an accepted request, not
// a completed result.
func RemoveCommandProducer(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
	return func(ctx context.Context) func(f field.Model, c character.Model, m string) (command.Executor, bool) {
		return func(f field.Model, c character.Model, m string) (command.Executor, bool) {
			match := removeRe.FindStringSubmatch(m)
			if match == nil {
				return nil, false
			}

			if !c.Gm() {
				return nil, false
			}

			targetName := match[1]
			here := len(match) >= 3 && match[2] != ""

			return func(l logrus.FieldLogger) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					cp := character.NewProcessor(l, ctx)
					lookup := func(name string) (character.Model, error) { return cp.GetByName()(name) }
					publish := func(p model.Provider[[]kafka.Message]) error {
						return producer.ProviderImpl(l)(ctx)(msg.EnvCommandTopic)(p)
					}
					msgProc := message.NewProcessor(l, ctx)
					pink := func(text string) error { return msgProc.IssuePinkText(f, 0, text, []uint32{c.Id()}) }

					var mapId *_map.Id
					if here {
						id := f.MapId()
						mapId = &id
					}
					return removeWithDeps(f, c, targetName, mapId, lookup, publish, pink)
				}
			}, true
		}
	}
}

// removeWithDeps is RemoveCommandProducer's execution logic, factored out
// for the same testability reason as deployWithDeps.
func removeWithDeps(f field.Model, c character.Model, targetName string, mapId *_map.Id, lookup characterLookup, publish commandPublisher, pink pinkTextSender) error {
	target, err := lookup(targetName)
	if err != nil {
		return pink(fmt.Sprintf("Character not found: %s.", targetName))
	}

	key := producer.CreateKey(int(target.Id()))
	value := msg.Command[msg.CommandRemoveBody]{
		CharacterId: target.Id(),
		Type:        msg.CommandTypeRemove,
		Requester: &msg.Requester{
			CharacterId: c.Id(),
			WorldId:     byte(f.WorldId()),
			ChannelId:   byte(f.ChannelId()),
			MapId:       uint32(f.MapId()),
		},
		Body: msg.CommandRemoveBody{MapId: mapId},
	}
	if err := publish(producer.SingleMessageProvider(key, value)); err != nil {
		return pink(fmt.Sprintf("Failed to remove Player NPC(s) for %s: %s.", target.Name(), err.Error()))
	}
	if mapId != nil {
		return pink(fmt.Sprintf("Accepted removal of %s's Player NPC on this map; you will be notified of the result.", target.Name()))
	}
	return pink(fmt.Sprintf("Accepted removal of %s's Player NPC(s); you will be notified of the result.", target.Name()))
}
