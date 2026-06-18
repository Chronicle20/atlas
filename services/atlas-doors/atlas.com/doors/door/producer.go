package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdEventProvider(m Model, forCharacterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[CreatedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		PairId: m.PairId(), OwnerCharacterId: m.OwnerCharacterId(), PartyId: m.PartyId(),
		ForCharacterId: forCharacterId,
		Type:           EventDoorStatusCreated,
		Body: CreatedBody{
			AreaDoorId: m.AreaDoorId(), TownDoorId: m.TownDoorId(), TownMapId: m.TownMapId(),
			Slot: m.Slot(), TownPortalId: m.TownPortalId(),
			AreaX: m.AreaX(), AreaY: m.AreaY(), TownX: m.TownX(), TownY: m.TownY(),
			SkillId: m.SkillId(), SkillLevel: m.SkillLevel(), ExpiresAt: timeToMs(m.ExpiresAt()),
		},
	}
	return producer.SingleMessageProvider(key, &value)
}

func removedEventProvider(m Model, reason string, forCharacterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[RemovedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		PairId: m.PairId(), OwnerCharacterId: m.OwnerCharacterId(), PartyId: m.PartyId(),
		ForCharacterId: forCharacterId,
		Type:           EventDoorStatusRemoved,
		Body: RemovedBody{AreaDoorId: m.AreaDoorId(), TownDoorId: m.TownDoorId(),
			TownMapId: m.TownMapId(), Slot: m.Slot(), Reason: reason},
	}
	return producer.SingleMessageProvider(key, &value)
}

func slotChangedEventProvider(m Model, oldSlot byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.Field().MapId()))
	value := StatusEvent[SlotChangedBody]{
		WorldId: m.Field().WorldId(), ChannelId: m.Field().ChannelId(),
		MapId: m.Field().MapId(), Instance: m.Field().Instance(),
		PairId: m.PairId(), OwnerCharacterId: m.OwnerCharacterId(), PartyId: m.PartyId(),
		Type: EventDoorStatusSlotChanged,
		Body: SlotChangedBody{AreaDoorId: m.AreaDoorId(), TownDoorId: m.TownDoorId(),
			TownMapId: m.TownMapId(), OldSlot: oldSlot, NewSlot: m.Slot(),
			TownPortalId: m.TownPortalId(), TownX: m.TownX(), TownY: m.TownY(),
			AreaX: m.AreaX(), AreaY: m.AreaY()},
	}
	return producer.SingleMessageProvider(key, &value)
}
