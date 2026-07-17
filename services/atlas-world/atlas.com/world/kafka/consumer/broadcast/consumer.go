package broadcast

import (
	"atlas-world/broadcast"
	consumer2 "atlas-world/kafka/consumer"
	broadcast2 "atlas-world/kafka/message/broadcast"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// InitConsumers registers the world-broadcast command consumer.
// Deliberately does NOT pass consumer.LastOffset: unlike the channel-status
// consumer (whose events are re-emitted on a fixed cadence, so a missed
// message after a restart is self-healing), EnqueueCommand is a one-shot
// command with no re-emission - starting from LastOffset would silently
// drop every enqueue command produced while this consumer group was down.
func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("world_broadcast_command")(broadcast2.EnvCommandTopicWorldBroadcast)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(broadcast2.EnvCommandTopicWorldBroadcast)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnqueueCommand))); err != nil {
			return err
		}
		return nil
	}
}

func handleEnqueueCommand(l logrus.FieldLogger, ctx context.Context, cmd broadcast2.EnqueueCommand) {
	if cmd.Family != broadcast.FamilyTV && cmd.Family != broadcast.FamilyAvatar {
		l.Errorf("Unhandled broadcast family [%s].", cmd.Family)
		return
	}

	e := broadcast.Entry{
		Id:              uuid.New(),
		CharacterId:     cmd.CharacterId,
		DurationSeconds: cmd.DurationSeconds,
		Payload: broadcast.Payload{
			ChannelId:     cmd.ChannelId,
			SenderName:    cmd.SenderName,
			SenderMedal:   cmd.SenderMedal,
			Messages:      cmd.Messages,
			WhispersOn:    cmd.WhispersOn,
			ItemId:        cmd.ItemId,
			TvMessageType: cmd.TvMessageType,
			SenderLook:    cmd.SenderLook,
			ReceiverName:  cmd.ReceiverName,
			ReceiverLook:  cmd.ReceiverLook,
		},
	}

	if err := broadcast.NewProcessor(l, ctx).Enqueue(world.Id(cmd.WorldId), cmd.Family, e); err != nil {
		l.WithError(err).Errorf("Failed to enqueue broadcast entry for character [%d] family [%s].", cmd.CharacterId, cmd.Family)
	}
}
