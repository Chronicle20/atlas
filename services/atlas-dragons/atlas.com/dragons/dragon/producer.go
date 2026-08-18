package dragon

import (
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func createdEventProvider(m Model) model.Provider[[]kafka.Message] {
	f := m.Field()
	key := producer.CreateKey(int(f.MapId()))
	value := StatusEvent[StatusEventCreatedBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(),
		MapId: f.MapId(), Instance: f.Instance(),
		OwnerCharacterId: m.OwnerCharacterId(),
		Type:             EventDragonStatusCreated,
		Body: StatusEventCreatedBody{
			X: m.X(), Y: m.Y(), Stance: m.Stance(), JobId: uint16(m.JobId()),
		},
	}
	return producer.SingleMessageProvider(key, &value)
}

func movedEventProvider(m Model, rawMovement []byte) model.Provider[[]kafka.Message] {
	f := m.Field()
	key := producer.CreateKey(int(f.MapId()))
	value := StatusEvent[StatusEventMovedBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(),
		MapId: f.MapId(), Instance: f.Instance(),
		OwnerCharacterId: m.OwnerCharacterId(),
		Type:             EventDragonStatusMoved,
		Body:             StatusEventMovedBody{RawMovement: rawMovement},
	}
	return producer.SingleMessageProvider(key, &value)
}

func destroyedEventProvider(m Model) model.Provider[[]kafka.Message] {
	f := m.Field()
	key := producer.CreateKey(int(f.MapId()))
	value := StatusEvent[StatusEventDestroyedBody]{
		WorldId: f.WorldId(), ChannelId: f.ChannelId(),
		MapId: f.MapId(), Instance: f.Instance(),
		OwnerCharacterId: m.OwnerCharacterId(),
		Type:             EventDragonStatusDestroyed,
		Body:             StatusEventDestroyedBody{},
	}
	return producer.SingleMessageProvider(key, &value)
}
