package party_quest

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_PARTY_QUEST"

	CommandTypeRegister     = "REGISTER"
	CommandTypeStageAdvance = "STAGE_ADVANCE"
)

type Command[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type RegisterCommandBody struct {
	QuestId   string     `json:"questId"`
	PartyId   uint32     `json:"partyId,omitempty"`
	ChannelId channel.Id `json:"channelId"`
	MapId     uint32     `json:"mapId"`
}

type StageAdvanceCommandBody struct {
	InstanceId uuid.UUID `json:"instanceId"`
}

func RegisterCommandProvider(worldId world.Id, characterId uint32, questId string, channelId channel.Id, mapId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command[RegisterCommandBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        CommandTypeRegister,
		Body: RegisterCommandBody{
			QuestId:   questId,
			ChannelId: channelId,
			MapId:     mapId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func StageAdvanceCommandProvider(worldId world.Id, characterId uint32, instanceId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command[StageAdvanceCommandBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        CommandTypeStageAdvance,
		Body: StageAdvanceCommandBody{
			InstanceId: instanceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
