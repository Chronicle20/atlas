package rps

import (
	"atlas-channel/kafka/message/rps"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// SelectCommandProvider creates a SELECT command for the rps service.
func SelectCommandProvider(characterId uint32, worldId world.Id, channelId channel.Id, throw byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &rps.Command[rps.SelectCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		Type:        rps.CommandTypeSelect,
		Body:        rps.SelectCommandBody{Throw: throw},
	}
	return producer.SingleMessageProvider(key, value)
}

// ContinueCommandProvider creates a CONTINUE command for the rps service.
func ContinueCommandProvider(characterId uint32, worldId world.Id, channelId channel.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &rps.Command[rps.ContinueCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		Type:        rps.CommandTypeContinue,
		Body:        rps.ContinueCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

// CollectCommandProvider creates a COLLECT command for the rps service. The
// rps service treats COLLECT as collect-or-forfeit depending on session
// status - it is also the command emitted for the client's EXIT sub-op,
// since there is no dedicated collect sub-op on the wire (IDA-verified,
// Task 16).
func CollectCommandProvider(characterId uint32, worldId world.Id, channelId channel.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &rps.Command[rps.CollectCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		Type:        rps.CommandTypeCollect,
		Body:        rps.CollectCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
