package character

import (
	character2 "atlas-saga-orchestrator/kafka/message/character"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func ChangeMapProvider(transactionId uuid.UUID, characterId uint32, field field.Model, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMapBody]{
		TransactionId: transactionId,
		WorldId:       field.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandChangeMap,
		Body: character2.ChangeMapBody{
			ChannelId: field.ChannelId(),
			MapId:     field.MapId(),
			Instance:  field.Instance(),
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AwardExperienceProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, distributions []character2.ExperienceDistributions) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.AwardExperienceCommandBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandAwardExperience,
		Body: character2.AwardExperienceCommandBody{
			ChannelId:     ch.Id(),
			Distributions: distributions,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AwardLevelProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, amount byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.AwardLevelCommandBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandAwardLevel,
		Body: character2.AwardLevelCommandBody{
			ChannelId: ch.Id(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AwardMesosProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, actorId uint32, actorType string, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.RequestChangeMesoBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandRequestChangeMeso,
		Body: character2.RequestChangeMesoBody{
			ActorId:   actorId,
			ActorType: actorType,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AwardFameProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.RequestChangeFameBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandRequestChangeFame,
		Body: character2.RequestChangeFameBody{
			ActorId:   0,        // System/NPC-initiated fame change (no player actor)
			ActorType: "SYSTEM", // Fame awarded by NPC/quest system
			Amount:    int8(amount),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeJobProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, jobId job.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeJobCommandBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandChangeJob,
		Body: character2.ChangeJobCommandBody{
			ChannelId: ch.Id(),
			JobId:     jobId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeHairProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, styleId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeHairCommandBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandChangeHair,
		Body: character2.ChangeHairCommandBody{
			ChannelId: ch.Id(),
			StyleId:   styleId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeFaceProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, styleId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeFaceCommandBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandChangeFace,
		Body: character2.ChangeFaceCommandBody{
			ChannelId: ch.Id(),
			StyleId:   styleId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeSkinProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, styleId byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeSkinCommandBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandChangeSkin,
		Body: character2.ChangeSkinCommandBody{
			ChannelId: ch.Id(),
			StyleId:   styleId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func SetHPProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, amount uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.SetHPBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandSetHP,
		Body: character2.SetHPBody{
			ChannelId: ch.Id(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeductExperienceProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.DeductExperienceCommandBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandDeductExperience,
		Body: character2.DeductExperienceCommandBody{
			ChannelId: ch.Id(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ResetStatsProvider(transactionId uuid.UUID, ch channel.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ResetStatsCommandBody]{
		TransactionId: transactionId,
		WorldId:       ch.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandResetStats,
		Body: character2.ResetStatsCommandBody{
			ChannelId: ch.Id(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestCreateCharacterProvider(transactionId uuid.UUID, accountId uint32, worldId world.Id, name string, level byte, strength uint16, dexterity uint16, intelligence uint16, luck uint16, hp uint16, mp uint16, jobId job.Id, gender byte, face uint32, hair uint32, skin byte, mapId _map.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &character2.Command[character2.CreateCharacterCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   0, // Character ID is not known yet for creation
		Type:          character2.CommandCreateCharacter,
		Body: character2.CreateCharacterCommandBody{
			AccountId:    accountId,
			WorldId:      worldId,
			Name:         name,
			Level:        level,
			Strength:     strength,
			Dexterity:    dexterity,
			Intelligence: intelligence,
			Luck:         luck,
			MaxHp:        hp,
			MaxMp:        mp,
			JobId:        jobId,
			Gender:       gender,
			Hair:         hair,
			Face:         face,
			SkinColor:    skin,
			MapId:        mapId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
