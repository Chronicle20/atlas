package pet

import (
	pet2 "atlas-pets/kafka/message/pet"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func spawnEventProvider(m Model, tm *temporalData) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet2.StatusEvent[pet2.SpawnedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet2.StatusEventTypeSpawned,
		Body: pet2.SpawnedStatusEventBody{
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
	value := &pet2.StatusEvent[pet2.DespawnedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet2.StatusEventTypeDespawned,
		Body: pet2.DespawnedStatusEventBody{
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
	value := &pet2.StatusEvent[pet2.CommandResponseStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet2.StatusEventTypeCommandResponse,
		Body: pet2.CommandResponseStatusEventBody{
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
	value := &pet2.StatusEvent[pet2.ClosenessChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet2.StatusEventTypeClosenessChanged,
		Body: pet2.ClosenessChangedStatusEventBody{
			Slot:      m.Slot(),
			Closeness: m.Closeness(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func fullnessChangedEventProvider(m Model, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet2.StatusEvent[pet2.FullnessChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet2.StatusEventTypeFullnessChanged,
		Body: pet2.FullnessChangedStatusEventBody{
			Slot:     m.Slot(),
			Fullness: m.Fullness(),
			Amount:   amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func levelChangedEventProvider(m Model, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet2.StatusEvent[pet2.LevelChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet2.StatusEventTypeLevelChanged,
		Body: pet2.LevelChangedStatusEventBody{
			Slot:   m.Slot(),
			Level:  m.Level(),
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func slotChangedEventProvider(m Model, oldSlot int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &pet2.StatusEvent[pet2.SlotChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet2.StatusEventTypeSlotChanged,
		Body: pet2.SlotChangedStatusEventBody{
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
	value := &pet2.StatusEvent[pet2.ExcludeChangedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet2.StatusEventTypeExcludeChanged,
		Body: pet2.ExcludeChangedStatusEventBody{
			Items: items,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
