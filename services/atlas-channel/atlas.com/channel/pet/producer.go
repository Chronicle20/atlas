package pet

import (
	pet2 "atlas-channel/kafka/message/pet"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func SpawnProvider(characterId uint32, petId uint32, lead bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.SpawnCommandBody]{
		ActorId: characterId,
		PetId:   petId,
		Type:    pet2.CommandPetSpawn,
		Body: pet2.SpawnCommandBody{
			Lead: lead,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DespawnProvider(characterId uint32, petId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.DespawnCommandBody]{
		ActorId: characterId,
		PetId:   petId,
		Type:    pet2.CommandPetDespawn,
		Body:    pet2.DespawnCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func AttemptCommandProvider(petId uint32, commandId byte, byName bool, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.AttemptCommandCommandBody]{
		ActorId: characterId,
		PetId:   petId,
		Type:    pet2.CommandPetAttemptCommand,
		Body: pet2.AttemptCommandCommandBody{
			CommandId: commandId,
			ByName:    byName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func SetExcludesCommandProvider(characterId uint32, petId uint32, items []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.SetExcludeCommandBody]{
		ActorId: characterId,
		PetId:   petId,
		Type:    pet2.CommandPetSetExclude,
		Body: pet2.SetExcludeCommandBody{
			Items: items,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
