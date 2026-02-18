package monster

import (
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	EnvCommandTopic              = "COMMAND_TOPIC_MONSTER"
	CommandTypeApplyStatusField  = "APPLY_STATUS_FIELD"
	CommandTypeCancelStatusField = "CANCEL_STATUS_FIELD"
	CommandTypeUseSkillField     = "USE_SKILL_FIELD"
	CommandTypeDestroyField      = "DESTROY_FIELD"
)

type FieldCommand[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

type ApplyStatusFieldBody struct {
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	Duration          uint32           `json:"duration"`
	TickInterval      uint32           `json:"tickInterval"`
}

type CancelStatusFieldBody struct {
	StatusTypes []string `json:"statusTypes"`
}

func ApplyStatusFieldCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID, statuses map[string]int32, duration uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := FieldCommand[ApplyStatusFieldBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Instance:  instance,
		Type:      CommandTypeApplyStatusField,
		Body: ApplyStatusFieldBody{
			SourceType: "GM_COMMAND",
			Statuses:   statuses,
			Duration:   duration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

type UseSkillFieldBody struct {
	SkillId    uint16 `json:"skillId"`
	SkillLevel uint16 `json:"skillLevel"`
}

func UseSkillFieldCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID, skillId uint16, skillLevel uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := FieldCommand[UseSkillFieldBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Instance:  instance,
		Type:      CommandTypeUseSkillField,
		Body: UseSkillFieldBody{
			SkillId:    skillId,
			SkillLevel: skillLevel,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

type DestroyFieldBody struct{}

func DestroyFieldCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := FieldCommand[DestroyFieldBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Instance:  instance,
		Type:      CommandTypeDestroyField,
		Body:      DestroyFieldBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func CancelStatusFieldCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID, statusTypes []string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := FieldCommand[CancelStatusFieldBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Instance:  instance,
		Type:      CommandTypeCancelStatusField,
		Body: CancelStatusFieldBody{
			StatusTypes: statusTypes,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
