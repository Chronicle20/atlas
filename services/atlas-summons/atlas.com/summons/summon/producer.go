package summon

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdEventProvider(m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[StatusEventCreatedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		SummonId: m.Id(), OwnerCharacterId: m.OwnerCharacterId(), SkillId: m.SkillId(),
		Type: EventSummonStatusCreated,
		Body: StatusEventCreatedBody{
			SkillLevel: m.SkillLevel(), MovementType: byte(m.MovementType()),
			X: m.X(), Y: m.Y(), Stance: m.Stance(),
			Puppet: m.IsPuppet(), Animated: m.Animated(),
		},
	}
	return producer.SingleMessageProvider(key, &value)
}

func movedEventProvider(m Model, rawMovement []byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[StatusEventMovedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		SummonId: m.Id(), OwnerCharacterId: m.OwnerCharacterId(), SkillId: m.SkillId(),
		Type: EventSummonStatusMoved,
		Body: StatusEventMovedBody{
			X: m.X(), Y: m.Y(), Stance: m.Stance(), RawMovement: rawMovement,
		},
	}
	return producer.SingleMessageProvider(key, &value)
}

func destroyedEventProvider(m Model, animated bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[StatusEventDestroyedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		SummonId: m.Id(), OwnerCharacterId: m.OwnerCharacterId(), SkillId: m.SkillId(),
		Type: EventSummonStatusDestroyed,
		Body: StatusEventDestroyedBody{Animated: animated},
	}
	return producer.SingleMessageProvider(key, &value)
}
