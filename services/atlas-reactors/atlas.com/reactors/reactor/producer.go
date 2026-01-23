package reactor

import (
	"fmt"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(worldId byte, channelId byte, mapId uint32, classification uint32, name string, state int8, x int16, y int16, delay uint32, direction byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &Command[CreateCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      CommandTypeCreate,
		Body: CreateCommandBody{
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

func hitStatusEventProvider(r Model, destroyed bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(r.Id()))
	value := &statusEvent[hitStatusEventBody]{
		WorldId:   r.WorldId(),
		ChannelId: r.ChannelId(),
		MapId:     r.MapId(),
		ReactorId: r.Id(),
		Type:      EventStatusTypeHit,
		Body: hitStatusEventBody{
			Classification: r.Classification(),
			State:          r.State(),
			X:              r.X(),
			Y:              r.Y(),
			Direction:      r.Direction(),
			Destroyed:      destroyed,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// hitActionsCommandProvider creates a HIT command for atlas-reactor-actions
func hitActionsCommandProvider(r Model, characterId uint32, skillId uint32, isSkill bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(r.Id()))
	value := &reactorActionsCommand[hitActionsBody]{
		WorldId:        r.WorldId(),
		ChannelId:      r.ChannelId(),
		MapId:          r.MapId(),
		ReactorId:      r.Id(),
		Classification: formatClassification(r.Classification()),
		ReactorName:    r.Name(),
		ReactorState:   r.State(),
		X:              r.X(),
		Y:              r.Y(),
		Type:           CommandTypeActionsHit,
		Body: hitActionsBody{
			CharacterId: characterId,
			SkillId:     skillId,
			IsSkill:     isSkill,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// triggerActionsCommandProvider creates a TRIGGER command for atlas-reactor-actions
func triggerActionsCommandProvider(r Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(r.Id()))
	value := &reactorActionsCommand[triggerActionsBody]{
		WorldId:        r.WorldId(),
		ChannelId:      r.ChannelId(),
		MapId:          r.MapId(),
		ReactorId:      r.Id(),
		Classification: formatClassification(r.Classification()),
		ReactorName:    r.Name(),
		ReactorState:   r.State(),
		X:              r.X(),
		Y:              r.Y(),
		Type:           CommandTypeActionsTrigger,
		Body: triggerActionsBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// formatClassification converts the classification uint32 to a string for script lookup
func formatClassification(classification uint32) string {
	return fmt.Sprintf("%d", classification)
}
