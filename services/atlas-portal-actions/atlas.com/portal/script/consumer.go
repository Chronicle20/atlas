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
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// commandEvent represents a Kafka command message
type commandEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	PortalId  uint32     `json:"portalId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
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
func handleEnterCommand(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, c commandEvent[enterBody]) {
	l.Debugf("Received portal enter command for character [%d] on portal [%s] (id=%d) in map [%d]",
		c.Body.CharacterId, c.Body.PortalName, c.PortalId, c.MapId)

	// Create field model from command
	ch := channel.NewModel(c.WorldId, c.ChannelId)
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()

	// Create processor with tenant context from Kafka message
	processor := NewProcessor(l, ctx, db)

	// Process the portal script (pass numeric portalId for use in operations like block_portal)
	result := processor.Process(f, c.Body.CharacterId, c.Body.PortalName, c.PortalId)

	if result.Error != nil {
		l.WithError(result.Error).Errorf("Failed to process portal script [%s] for character [%d]",
			c.Body.PortalName, c.Body.CharacterId)
		// On error, enable character actions so they're not stuck
		character.EnableActions(l)(ctx)(ch, c.Body.CharacterId)
		return
	}

	l.Debugf("Portal script [%s] result: allow=%t, matchedRule=%s",
		c.Body.PortalName, result.Allow, result.MatchedRule)

	// If not allowed, just enable character actions (they stay where they are)
	if !result.Allow {
		character.EnableActions(l)(ctx)(ch, c.Body.CharacterId)
		return
	}

	// If allowed with no explicit warp operation, enable actions
	// (the portal itself may handle the warp in atlas-portals)
	character.EnableActions(l)(ctx)(ch, c.Body.CharacterId)
}
