package character

import (
	"atlas-buffs/buff/stat"
	character2 "atlas-buffs/kafka/message/character"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"time"
)

func appliedStatusEventProvider(worldId byte, characterId uint32, fromId uint32, sourceId int32, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time) model.Provider[[]kafka.Message] {
	statups := make([]character2.StatChange, 0)
	for _, su := range changes {
		statups = append(statups, character2.StatChange{
			Type:   su.Type(),
			Amount: su.Amount(),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.AppliedStatusEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character2.EventStatusTypeBuffApplied,
		Body: character2.AppliedStatusEventBody{
			FromId:    fromId,
			SourceId:  sourceId,
			Duration:  duration,
			Changes:   statups,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func expiredStatusEventProvider(worldId byte, characterId uint32, sourceId int32, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time) model.Provider[[]kafka.Message] {
	statups := make([]character2.StatChange, 0)
	for _, su := range changes {
		statups = append(statups, character2.StatChange{
			Type:   su.Type(),
			Amount: su.Amount(),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.ExpiredStatusEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character2.EventStatusTypeBuffExpired,
		Body: character2.ExpiredStatusEventBody{
			SourceId:  sourceId,
			Duration:  duration,
			Changes:   statups,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
