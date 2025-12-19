package pet

import (
	"atlas-pets/kafka/message/pet"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdEventProvider(m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.CreatedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeCreated,
		Body:    pet.CreatedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func deletedEventProvider(petId uint32, ownerId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(ownerId))
	value := &pet.StatusEvent[pet.DeletedStatusEventBody]{
		PetId:   petId,
		OwnerId: ownerId,
		Type:    pet.StatusEventTypeDeleted,
		Body:    pet.DeletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func spawnEventProvider(m Model, tm *TemporalData) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.SpawnedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeSpawned,
		Body: pet.SpawnedStatusEventBody{
			TemplateId: m.TemplateId(),
			Name:       m.Name(),
			Slot:       m.Slot(),
			Level:      m.Level(),
			Closeness:  m.Closeness(),
			Fullness:   m.Fullness(),
			X:          tm.X(),
			Y:          tm.Y(),
			Stance:     tm.Stance(),
			FH:         tm.FH(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func despawnEventProvider(m Model, oldSlot int8, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.DespawnedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeDespawned,
		Body: pet.DespawnedStatusEventBody{
			TemplateId: m.TemplateId(),
			Name:       m.Name(),
			Slot:       m.Slot(),
			Level:      m.Level(),
			Closeness:  m.Closeness(),
			Fullness:   m.Fullness(),
			OldSlot:    oldSlot,
			Reason:     reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func commandResponseEventProvider(m Model, commandId byte, success bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.CommandResponseStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeCommandResponse,
		Body: pet.CommandResponseStatusEventBody{
			Slot:      m.Slot(),
			Closeness: m.Closeness(),
			Fullness:  m.Fullness(),
			CommandId: commandId,
			Success:   success,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func closenessChangedEventProvider(m Model, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.ClosenessChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeClosenessChanged,
		Body: pet.ClosenessChangedStatusEventBody{
			Slot:      m.Slot(),
			Closeness: m.Closeness(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func fullnessChangedEventProvider(m Model, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.FullnessChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeFullnessChanged,
		Body: pet.FullnessChangedStatusEventBody{
			Slot:     m.Slot(),
			Fullness: m.Fullness(),
			Amount:   amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func levelChangedEventProvider(m Model, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.LevelChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeLevelChanged,
		Body: pet.LevelChangedStatusEventBody{
			Slot:   m.Slot(),
			Level:  m.Level(),
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func slotChangedEventProvider(m Model, oldSlot int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.SlotChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeSlotChanged,
		Body: pet.SlotChangedStatusEventBody{
			OldSlot: oldSlot,
			NewSlot: m.Slot(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func excludeChangedEventProvider(m Model) model.Provider[[]kafka.Message] {
	items := make([]uint32, len(m.excludes))
	for i, e := range m.excludes {
		items[i] = e.ItemId()
	}

	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet.StatusEvent[pet.ExcludeChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeExcludeChanged,
		Body: pet.ExcludeChangedStatusEventBody{
			Items: items,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
