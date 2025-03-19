package buff

import (
	"atlas-consumables/character/buff/stat"
	buff2 "atlas-consumables/kafka/message/character/buff"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func applyCommandProvider(m _map.Model, characterId uint32, fromId uint32, sourceId int32, duration int32, statups []stat.Model) model.Provider[[]kafka.Message] {
	changes := make([]buff2.StatChange, 0)
	for _, su := range statups {
		changes = append(changes, buff2.StatChange{
			Type:   string(su.Type),
			Amount: su.Amount,
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &buff2.Command[buff2.ApplyCommandBody]{
		WorldId:     byte(m.WorldId()),
		ChannelId:   byte(m.ChannelId()),
		CharacterId: characterId,
		Type:        buff2.CommandTypeApply,
		Body: buff2.ApplyCommandBody{
			FromId:   fromId,
			SourceId: sourceId,
			Duration: duration,
			Changes:  changes,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func cancelCommandProvider(m _map.Model, characterId uint32, sourceId int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff2.Command[buff2.CancelCommandBody]{
		WorldId:     byte(m.WorldId()),
		ChannelId:   byte(m.ChannelId()),
		CharacterId: characterId,
		Type:        buff2.CommandTypeCancel,
		Body: buff2.CancelCommandBody{
			SourceId: sourceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
