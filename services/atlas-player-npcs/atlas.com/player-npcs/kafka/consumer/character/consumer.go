// Package character consumes atlas-character's EVENT_TOPIC_CHARACTER_STATUS
// LEVEL_CHANGED event (Task 17, design §8.2 -- FR-1.5's automatic-deploy
// trigger). Every failure is logged at warn and swallowed: a failed
// deployment must never block or roll back a level-up.
package character

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/configuration"
	consumer2 "atlas-player-npcs/kafka/consumer"
	playernpcconsumer "atlas-player-npcs/kafka/consumer/playernpc"
	charmsg "atlas-player-npcs/kafka/message/character"
	npcmsg "atlas-player-npcs/kafka/message/playernpc"
	"atlas-player-npcs/playernpc"
	"atlas-player-npcs/routing"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// alreadyDeployedLookupPageSize bounds the GetByMap page the
// already-deployed check scans -- see redeployLookupPageSize's twin in
// kafka/consumer/playernpc/consumer.go for the same occupancy-bound
// rationale.
const alreadyDeployedLookupPageSize = 1000

// CharacterProcessorProvider and ConfigurationProcessorProvider let tests
// substitute stub processors instead of the real HTTP-backed ones; main.go
// wires character.NewProcessor / configuration.NewProcessor.
type (
	CharacterProcessorProvider     func(l logrus.FieldLogger, ctx context.Context) character.Processor
	ConfigurationProcessorProvider func(l logrus.FieldLogger, ctx context.Context) configuration.Processor
)

// Dependencies bundles the three read clients handleLevelChanged needs.
// PlayerNpc is playernpcconsumer.ProcessorProvider (shared with
// kafka/consumer/playernpc) -- GetByMap is how this consumer checks
// whether the character already has a Player NPC on the target map before
// emitting a DEPLOY command that would only be rejected downstream.
type Dependencies struct {
	Character     CharacterProcessorProvider
	Configuration ConfigurationProcessorProvider
	PlayerNpc     playernpcconsumer.ProcessorProvider
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("player_npc_character_status")(charmsg.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(deps Dependencies) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(deps Dependencies) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			t, _ := topic.EnvProvider(l)(charmsg.EnvEventTopicCharacterStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleLevelChanged(deps)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handleLevelChanged is design §8.2's consumer. The level check happens
// before any fetch -- current below the (job-agnostic today, design §3.3)
// max level is the overwhelmingly common case and must stay cheap.
// Otherwise: fetch the character, reject a GM, reject when the tenant has
// automatic deploy disabled, resolve the FR-2.1 target map, and skip if
// the character already has a Player NPC there -- only then emit DEPLOY.
func handleLevelChanged(deps Dependencies) message.Handler[charmsg.StatusEvent[charmsg.LevelChangedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e charmsg.StatusEvent[charmsg.LevelChangedStatusEventBody]) {
		if e.Type != charmsg.StatusEventTypeLevelChanged {
			return
		}
		// job.MaxLevelFor is job-agnostic today (every job's cap is 200,
		// design §3.3) -- job.BeginnerId is an arbitrary representative,
		// not a magic number, so this cheap gate stays correct without a
		// job id, which the fetch below has not happened yet to provide.
		if e.Body.Current < job.MaxLevelFor(job.BeginnerId) {
			return
		}

		c, err := deps.Character(l, ctx).GetById(e.CharacterId)
		if err != nil {
			l.WithError(err).Warnf("Unable to fetch character [%d] to evaluate automatic Player NPC deployment.", e.CharacterId)
			return
		}
		if c.Gm() {
			return
		}

		te := tenant.MustFromContext(ctx)
		cfg := deps.Configuration(l, ctx).GetByTenantId(te.Id())
		if !cfg.AutoDeployEnabled() {
			return
		}

		set := constants.For(te.Region(), te.MajorVersion(), te.MinorVersion())
		targetMapId := routing.HallOfFameMapFor(set, c.JobId())

		deployed, err := alreadyDeployed(deps.PlayerNpc(l, ctx), e.WorldId, targetMapId, e.CharacterId)
		if err != nil {
			l.WithError(err).Warnf("Unable to check existing Player NPC deployment for character [%d] on map [%d].", e.CharacterId, targetMapId)
			return
		}
		if deployed {
			return
		}

		if err := emitDeployCommand(l, ctx, e.CharacterId, e.WorldId, targetMapId); err != nil {
			l.WithError(err).Warnf("Unable to emit automatic Player NPC deploy command for character [%d] on map [%d].", e.CharacterId, targetMapId)
		}
	}
}

// alreadyDeployed reports whether characterId already has a Player NPC on
// (worldId, mapId), via Processor.GetByMap -- the only Processor surface
// that can answer this without changing Task 15's signatures.
func alreadyDeployed(pp playernpc.Processor, worldId world.Id, mapId _map.Id, characterId uint32) (bool, error) {
	ms, err := pp.GetByMap(worldId, mapId, model.Page{Number: 1, Size: alreadyDeployedLookupPageSize})
	if err != nil {
		return false, err
	}
	for _, m := range ms {
		if m.CharacterId() == characterId {
			return true, nil
		}
	}
	return false, nil
}

// emitDeployCommand publishes DEPLOY to COMMAND_TOPIC_PLAYER_NPC --
// enforceEligibility: true, since this is the automatic-deploy path
// (FR-1.1), never the GM path.
func emitDeployCommand(l logrus.FieldLogger, ctx context.Context, characterId uint32, worldId world.Id, mapId _map.Id) error {
	key := producer.CreateKey(int(characterId))
	value := &npcmsg.Command[npcmsg.CommandDeployBody]{
		CharacterId: characterId,
		Type:        npcmsg.CommandTypeDeploy,
		Body: npcmsg.CommandDeployBody{
			WorldId:            worldId,
			MapId:              mapId,
			EnforceEligibility: true,
		},
	}
	provider := producer.SingleMessageProvider(key, value)
	return producer.ProviderImpl(l)(ctx)(npcmsg.EnvCommandTopic)(provider)
}
