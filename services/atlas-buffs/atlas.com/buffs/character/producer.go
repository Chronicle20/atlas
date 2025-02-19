package character

import (
	"atlas-buffs/buff/stat"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func appliedStatusEventProvider(worldId byte, characterId uint32, sourceId uint32, duration int32, changes []stat.Model) model.Provider[[]kafka.Message] {
	statups := make([]statChange, 0)
	for _, su := range changes {
		statups = append(statups, statChange{
			Type:   su.Type(),
			Amount: su.Amount(),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &statusEvent[appliedStatusEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        EventStatusTypeBuffApplied,
		Body: appliedStatusEventBody{
			SourceId: sourceId,
			Duration: duration,
			Changes:  statups,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func expiredStatusEventProvider(worldId byte, characterId uint32, sourceId uint32, duration int32, changes []stat.Model) model.Provider[[]kafka.Message] {
	statups := make([]statChange, 0)
	for _, su := range changes {
		statups = append(statups, statChange{
			Type:   su.Type(),
			Amount: su.Amount(),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &statusEvent[expiredStatusEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        EventStatusTypeBuffExpired,
		Body: expiredStatusEventBody{
			SourceId: sourceId,
			Duration: duration,
			Changes:  statups,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
