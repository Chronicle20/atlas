package script

import (
	"context"

	"atlas-map-actions/character"
	consumer2 "atlas-map-actions/kafka/consumer"

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

type commandEvent[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type enterBody struct {
	CharacterId uint32 `json:"characterId"`
	ScriptName  string `json:"scriptName"`
	ScriptType  string `json:"scriptType"`
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(groupId string) {
		return func(groupId string) {
			rf(
				consumer2.NewConfig(l)("map_actions_command")(EnvCommandTopic)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
			)
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterCommandFunc(l, db)))); err != nil {
			return err
		}
		return nil
	}
}

func handleEnterCommandFunc(l logrus.FieldLogger, db *gorm.DB) func(logrus.FieldLogger, context.Context, commandEvent[enterBody]) {
	return func(fl logrus.FieldLogger, ctx context.Context, command commandEvent[enterBody]) {
		handleEnterCommand(l, ctx, db, command)
	}
}

func handleEnterCommand(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, c commandEvent[enterBody]) {
	l.Debugf("Received map enter command for character [%d] script [%s] type [%s] in map [%d].",
		c.Body.CharacterId, c.Body.ScriptName, c.Body.ScriptType, c.MapId)

	ch := channel.NewModel(c.WorldId, c.ChannelId)
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()

	processor := NewProcessor(l, ctx, db)

	result := processor.Process(f, c.Body.CharacterId, c.Body.ScriptName, c.Body.ScriptType)

	if result.Error != nil {
		l.WithError(result.Error).Errorf("Failed to process map script [%s] for character [%d].",
			c.Body.ScriptName, c.Body.CharacterId)
	}

	character.EnableActions(l)(ctx)(ch, c.Body.CharacterId)
}
