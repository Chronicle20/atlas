package pet

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func spawnEventProvider(m Model, tm *temporalData) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &statusEvent[spawnedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    StatusEventTypeSpawned,
		Body: spawnedStatusEventBody{
			TemplateId: m.TemplateId(),
			Name:       m.Name(),
			Slot:       m.Slot(),
			Level:      m.Level(),
			Tameness:   m.Tameness(),
			Fullness:   m.Fullness(),
			X:          tm.X(),
			Y:          tm.Y(),
			Stance:     tm.Stance(),
			FH:         tm.FH(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func despawnEventProvider(m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.OwnerId()))
	value := &statusEvent[despawnedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    StatusEventTypeDespawned,
		Body: despawnedStatusEventBody{
			TemplateId: m.TemplateId(),
			Name:       m.Name(),
			Slot:       m.Slot(),
			Level:      m.Level(),
			Tameness:   m.Tameness(),
			Fullness:   m.Fullness(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func moveEventProvider(m _map.Model, p Model, mov Movement) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(p.OwnerId()))
	value := &movementEvent{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MapId:     uint32(m.MapId()),
		PetId:     p.Id(),
		Slot:      p.Slot(),
		OwnerId:   p.OwnerId(),
		Movement:  mov,
	}
	return producer.SingleMessageProvider(key, value)
}
