package script

import (
	"context"

	"atlas-portal-actions/character"
	consumer2 "atlas-portal-actions/kafka/consumer"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// commandEvent represents a Kafka command message
type commandEvent[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	PortalId  uint32 `json:"portalId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

// enterBody represents the body of an enter portal command
type enterBody struct {
	CharacterId uint32 `json:"characterId"`
	PortalName  string `json:"portalName"`
}

// InitConsumers initializes Kafka consumers for portal actions
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
		return func(groupId string) {
			rf(
				consumer2.NewConfig(l)("portal_actions_command")(EnvCommandTopic)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
			)
		}
	}
}

// InitHandlers initializes Kafka message handlers
func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterCommandFunc(l, db))))
	}
}

// handleEnterCommandFunc returns a handler function for enter commands
func handleEnterCommandFunc(l logrus.FieldLogger, db *gorm.DB) func(logrus.FieldLogger, context.Context, commandEvent[enterBody]) {
	return func(fl logrus.FieldLogger, ctx context.Context, command commandEvent[enterBody]) {
		handleEnterCommand(l, ctx, db, command)
	}
}

// handleEnterCommand handles a portal enter command
func handleEnterCommand(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, command commandEvent[enterBody]) {
	l.Debugf("Received portal enter command for character [%d] on portal [%s] (id=%d) in map [%d]",
		command.Body.CharacterId, command.Body.PortalName, command.PortalId, command.MapId)

	// Create field model from command
	f := field.NewBuilder(world.Id(command.WorldId), channel.Id(command.ChannelId), _map.Id(command.MapId)).Build()

	// Create processor with tenant context from Kafka message
	processor := NewProcessor(l, ctx, db)

	// Process the portal script (pass numeric portalId for use in operations like block_portal)
	result := processor.Process(f, command.Body.CharacterId, command.Body.PortalName, command.PortalId)

	if result.Error != nil {
		l.WithError(result.Error).Errorf("Failed to process portal script [%s] for character [%d]",
			command.Body.PortalName, command.Body.CharacterId)
		// On error, enable character actions so they're not stuck
		character.EnableActions(l)(ctx)(command.WorldId, command.ChannelId, command.Body.CharacterId)
		return
	}

	l.Debugf("Portal script [%s] result: allow=%t, matchedRule=%s",
		command.Body.PortalName, result.Allow, result.MatchedRule)

	// If not allowed, just enable character actions (they stay where they are)
	if !result.Allow {
		character.EnableActions(l)(ctx)(command.WorldId, command.ChannelId, command.Body.CharacterId)
		return
	}

	// If allowed with no explicit warp operation, enable actions
	// (the portal itself may handle the warp in atlas-portals)
	character.EnableActions(l)(ctx)(command.WorldId, command.ChannelId, command.Body.CharacterId)
}
