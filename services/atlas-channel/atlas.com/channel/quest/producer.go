package quest

import (
	quest2 "atlas-channel/kafka/message/quest"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func StartConversationCommandProvider(m _map.Model, questId uint32, npcId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &quest2.Command[quest2.StartQuestConversationCommandBody]{
		QuestId:     questId,
		NpcId:       npcId,
		CharacterId: characterId,
		Type:        quest2.CommandTypeStartQuestConversation,
		Body: quest2.StartQuestConversationCommandBody{
			WorldId:   m.WorldId(),
			ChannelId: m.ChannelId(),
			MapId:     m.MapId(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
