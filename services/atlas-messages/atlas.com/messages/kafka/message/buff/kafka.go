package buff

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
	EnvCommandTopic  = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply = "APPLY"
)

type Command[E any] struct {
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Instance    uuid.UUID  `json:"instance"`
	CharacterId uint32     `json:"characterId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

type ApplyCommandBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Duration int32        `json:"duration"`
	Changes  []StatChange `json:"changes"`
}

type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func ApplyCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID, characterId uint32, fromId uint32, sourceId int32, duration int32, changes []StatChange) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := Command[ApplyCommandBody]{
		WorldId:     worldId,
		ChannelId:   channelId,
		MapId:       mapId,
		Instance:    instance,
		CharacterId: characterId,
		Type:        CommandTypeApply,
		Body: ApplyCommandBody{
			FromId:   fromId,
			SourceId: sourceId,
			Duration: duration,
			Changes:  changes,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
