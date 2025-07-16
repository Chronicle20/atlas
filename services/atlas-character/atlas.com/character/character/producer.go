package character

import (
	character2 "atlas-character/kafka/message/character"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func awardLevelCommandProvider(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.AwardLevelCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       channel.WorldId(),
		Type:          character2.CommandAwardLevel,
		Body: character2.AwardLevelCommandBody{
			ChannelId: channel.Id(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdEventProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatusEventCreatedBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          character2.StatusEventTypeCreated,
		Body: character2.StatusEventCreatedBody{
			Name: name,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func loginEventProvider(transactionId uuid.UUID, characterId uint32, field field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatusEventLoginBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       field.WorldId(),
		Type:          character2.StatusEventTypeLogin,
		Body: character2.StatusEventLoginBody{
			ChannelId: field.ChannelId(),
			MapId:     field.MapId(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func logoutEventProvider(transactionId uuid.UUID, characterId uint32, field field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatusEventLogoutBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       field.WorldId(),
		Type:          character2.StatusEventTypeLogout,
		Body: character2.StatusEventLogoutBody{
			ChannelId: field.ChannelId(),
			MapId:     field.MapId(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeChannelEventProvider(transactionId uuid.UUID, characterId uint32, oldField field.Model, newField field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.ChangeChannelEventLoginBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       newField.WorldId(),
		Type:          character2.StatusEventTypeChannelChanged,
		Body: character2.ChangeChannelEventLoginBody{
			ChannelId:    newField.ChannelId(),
			OldChannelId: oldField.ChannelId(),
			MapId:        newField.MapId(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func mapChangedEventProvider(transactionId uuid.UUID, characterId uint32, oldField field.Model, newField field.Model, targetPortalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatusEventMapChangedBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       newField.WorldId(),
		Type:          character2.StatusEventTypeMapChanged,
		Body: character2.StatusEventMapChangedBody{
			ChannelId:      newField.ChannelId(),
			OldMapId:       oldField.MapId(),
			TargetMapId:    newField.MapId(),
			TargetPortalId: targetPortalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func jobChangedEventProvider(transactionId uuid.UUID, characterId uint32, channel channel.Model, jobId job.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.JobChangedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       channel.WorldId(),
		Type:          character2.StatusEventTypeJobChanged,
		Body: character2.JobChangedStatusEventBody{
			ChannelId: channel.Id(),
			JobId:     jobId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func experienceChangedEventProvider(transactionId uuid.UUID, characterId uint32, channel channel.Model, experience []ExperienceModel, current uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))

	ds := make([]character2.ExperienceDistributions, 0)
	for _, e := range experience {
		ds = append(ds, character2.ExperienceDistributions{
			ExperienceType: e.experienceType,
			Amount:         e.amount,
			Attr1:          e.attr1,
		})
	}

	value := &character2.StatusEvent[character2.ExperienceChangedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       channel.WorldId(),
		Type:          character2.StatusEventTypeExperienceChanged,
		Body: character2.ExperienceChangedStatusEventBody{
			ChannelId:     channel.Id(),
			Current:       current,
			Distributions: ds,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func levelChangedEventProvider(transactionId uuid.UUID, characterId uint32, channel channel.Model, amount byte, current byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.LevelChangedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       channel.WorldId(),
		Type:          character2.StatusEventTypeLevelChanged,
		Body: character2.LevelChangedStatusEventBody{
			ChannelId: channel.Id(),
			Amount:    amount,
			Current:   current,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func deletedEventProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatusEventDeletedBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          character2.StatusEventTypeDeleted,
		Body:          character2.StatusEventDeletedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func mesoChangedStatusEventProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, amount int32, actorId uint32, actorType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.MesoChangedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          character2.StatusEventTypeMesoChanged,
		Body: character2.MesoChangedStatusEventBody{
			ActorId:   actorId,
			ActorType: actorType,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func notEnoughMesoErrorStatusEventProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatusEventErrorBody[character2.NotEnoughMesoErrorStatusBodyBody]]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          character2.StatusEventTypeError,
		Body: character2.StatusEventErrorBody[character2.NotEnoughMesoErrorStatusBodyBody]{
			Error: character2.StatusEventErrorTypeNotEnoughMeso,
			Body: character2.NotEnoughMesoErrorStatusBodyBody{
				Amount: amount,
			},
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func fameChangedStatusEventProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, amount int8, actorId uint32, actorType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.FameChangedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          character2.StatusEventTypeFameChanged,
		Body: character2.FameChangedStatusEventBody{
			ActorId:   actorId,
			ActorType: actorType,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func creationFailedEventProvider(transactionId uuid.UUID, worldId world.Id, name string, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(0) // Use 0 as key since no character ID exists on creation failure
	value := &character2.StatusEvent[character2.StatusEventCreationFailedBody]{
		TransactionId: transactionId,
		CharacterId:   0, // No character ID on creation failure
		WorldId:       worldId,
		Type:          character2.StatusEventTypeCreationFailed,
		Body: character2.StatusEventCreationFailedBody{
			Name:    name,
			Message: message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statChangedProvider(transactionId uuid.UUID, channel channel.Model, characterId uint32, updates []string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatusEventStatChangedBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          character2.StatusEventTypeStatChanged,
		WorldId:       channel.WorldId(),
		Body: character2.StatusEventStatChangedBody{
			ChannelId:       channel.Id(),
			ExclRequestSent: true,
			Updates:         updates,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
