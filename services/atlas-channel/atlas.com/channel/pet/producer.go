package pet

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func spawnProvider(characterId uint32, petId uint64, lead bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &commandEvent[spawnCommandBody]{
		ActorId: characterId,
		PetId:   petId,
		Type:    CommandPetSpawn,
		Body: spawnCommandBody{
			Lead: lead,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func despawnProvider(characterId uint32, petId uint64) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &commandEvent[despawnCommandBody]{
		ActorId: characterId,
		PetId:   petId,
		Type:    CommandPetDespawn,
		Body:    despawnCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func attemptCommandProvider(petId uint64, commandId byte, byName bool, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &commandEvent[attemptCommandCommandBody]{
		ActorId: characterId,
		PetId:   petId,
		Type:    CommandPetAttemptCommand,
		Body: attemptCommandCommandBody{
			CommandId: commandId,
			ByName:    byName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func setExcludesCommandProvider(characterId uint32, petId uint64, items []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &commandEvent[setExcludeCommandBody]{
		ActorId: characterId,
		PetId:   petId,
		Type:    CommandPetSetExclude,
		Body: setExcludeCommandBody{
			Items: items,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
