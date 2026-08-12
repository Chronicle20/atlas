package character

import (
	"atlas-buffs/buff/stat"
	character2 "atlas-buffs/kafka/message/character"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func appliedStatusEventProvider(worldId world.Id, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time, noExpiry bool) model.Provider[[]kafka.Message] {
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
			Level:     level,
			Duration:  duration,
			Changes:   statups,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
			NoExpiry:  noExpiry,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeHPCommandProvider(worldId world.Id, channelId channel.Id, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.CharacterCommand[character2.ChangeHPCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        character2.CommandChangeHP,
		Body: character2.ChangeHPCommandBody{
			ChannelId: channelId,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// periodicEffectStatusEventProvider announces one visual pulse for a periodic
// tick. Keyed by characterId, same as the CHANGE_HP command it rides beside,
// so a character's pulse and resource change stay ordered relative to each
// other.
func periodicEffectStatusEventProvider(worldId world.Id, channelId channel.Id, characterId uint32, skillId uint32, statType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.PeriodicEffectStatusEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character2.EventStatusTypePeriodicEffect,
		Body: character2.PeriodicEffectStatusEventBody{
			ChannelId: channelId,
			SkillId:   skillId,
			StatType:  statType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func expiredStatusEventProvider(worldId world.Id, characterId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time, noExpiry bool) model.Provider[[]kafka.Message] {
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
			Level:     level,
			Duration:  duration,
			Changes:   statups,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
			NoExpiry:  noExpiry,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statUpdatedStatusEventProvider(worldId world.Id, characterId uint32, sourceId int32, level byte, duration int32, changes []stat.Model, createdAt time.Time, expiresAt time.Time) model.Provider[[]kafka.Message] {
	statups := make([]character2.StatChange, 0)
	for _, su := range changes {
		statups = append(statups, character2.StatChange{
			Type:   su.Type(),
			Amount: su.Amount(),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatUpdatedStatusEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character2.EventStatusTypeStatUpdated,
		Body: character2.StatUpdatedStatusEventBody{
			SourceId:  sourceId,
			Level:     level,
			Duration:  duration,
			Changes:   statups,
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
