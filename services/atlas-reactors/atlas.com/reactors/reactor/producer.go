package reactor

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(worldId byte, channelId byte, mapId uint32, classification uint32, name string, state int8, x int16, y int16, delay uint32, direction byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &command[createCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      CommandTypeCreate,
		Body: createCommandBody{
			Classification: classification,
			Name:           name,
			State:          state,
			X:              x,
			Y:              y,
			Delay:          delay,
			Direction:      direction,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdStatusEventProvider(r Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(r.Id()))
	value := &statusEvent[createdStatusEventBody]{
		WorldId:   r.WorldId(),
		ChannelId: r.ChannelId(),
		MapId:     r.MapId(),
		ReactorId: r.Id(),
		Type:      EventStatusTypeCreated,
		Body: createdStatusEventBody{
			Classification: r.Classification(),
			Name:           r.Name(),
			State:          r.State(),
			EventState:     r.EventState(),
			Delay:          r.Delay(),
			Direction:      r.Direction(),
			X:              r.X(),
			Y:              r.Y(),
			UpdateTime:     r.UpdateTime(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func destroyedStatusEventProvider(r Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(r.Id()))
	value := &statusEvent[destroyedStatusEventBody]{
		WorldId:   r.WorldId(),
		ChannelId: r.ChannelId(),
		MapId:     r.MapId(),
		ReactorId: r.Id(),
		Type:      EventStatusTypeDestroyed,
		Body: destroyedStatusEventBody{
			State: r.State(),
			X:     r.X(),
			Y:     r.Y(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
