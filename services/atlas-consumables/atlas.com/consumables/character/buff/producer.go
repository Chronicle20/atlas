package buff

import (
	"atlas-consumables/character/buff/stat"
	buff2 "atlas-consumables/kafka/message/character/buff"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func applyCommandProvider(f field.Model, characterId uint32, fromId uint32, sourceId int32, duration int32, statups []stat.Model) model.Provider[[]kafka.Message] {
	changes := make([]buff2.StatChange, 0)
	for _, su := range statups {
		changes = append(changes, buff2.StatChange{
			Type:   string(su.Type),
			Amount: su.Amount,
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &buff2.Command[buff2.ApplyCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
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

func cancelCommandProvider(f field.Model, characterId uint32, sourceId int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff2.Command[buff2.CancelCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff2.CommandTypeCancel,
		Body: buff2.CancelCommandBody{
			SourceId: sourceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
