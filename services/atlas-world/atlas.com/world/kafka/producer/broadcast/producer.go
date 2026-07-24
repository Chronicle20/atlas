package broadcast

import (
	message "atlas-world/kafka/message/broadcast"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// QueuedStatusEventProvider emits a QUEUED status event carrying the
// estimated wait time (0 when the entry activated immediately).
func QueuedStatusEventProvider(worldId world.Id, family string, characterId uint32, waitSeconds uint32) model.Provider[[]kafka.Message] {
	key := kproducer.CreateKey(int(worldId))
	value := &message.StatusEvent{
		Type:        message.StatusTypeQueued,
		Family:      family,
		WorldId:     byte(worldId),
		CharacterId: characterId,
		WaitSeconds: waitSeconds,
	}
	return kproducer.SingleMessageProvider(key, value)
}

// StartedStatusEventProvider emits a STARTED status event carrying the full
// render payload of the activated entry. TotalWaitSeconds is
// p.DurationSeconds (SEND_TV totalWaitTime). Takes message.StartedPayload
// rather than the domain broadcast.Entry directly — see StartedPayload's
// doc comment for why (import-cycle avoidance).
func StartedStatusEventProvider(worldId world.Id, family string, p message.StartedPayload) model.Provider[[]kafka.Message] {
	key := kproducer.CreateKey(int(worldId))
	value := &message.StatusEvent{
		Type:             message.StatusTypeStarted,
		Family:           family,
		WorldId:          byte(worldId),
		CharacterId:      p.CharacterId,
		TotalWaitSeconds: p.DurationSeconds,
		ChannelId:        p.ChannelId,
		SenderName:       p.SenderName,
		SenderMedal:      p.SenderMedal,
		Messages:         p.Messages,
		WhispersOn:       p.WhispersOn,
		ItemId:           p.ItemId,
		TvMessageType:    p.TvMessageType,
		SenderLook:       p.SenderLook,
		ReceiverName:     p.ReceiverName,
		ReceiverLook:     p.ReceiverLook,
	}
	return kproducer.SingleMessageProvider(key, value)
}

// EndedStatusEventProvider emits an ENDED status event for the entry whose
// active slot just expired. Carries only Family/WorldId/CharacterId.
func EndedStatusEventProvider(worldId world.Id, family string, characterId uint32) model.Provider[[]kafka.Message] {
	key := kproducer.CreateKey(int(worldId))
	value := &message.StatusEvent{
		Type:        message.StatusTypeEnded,
		Family:      family,
		WorldId:     byte(worldId),
		CharacterId: characterId,
	}
	return kproducer.SingleMessageProvider(key, value)
}
