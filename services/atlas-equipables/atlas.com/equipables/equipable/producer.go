package equipable

import (
	"atlas-equipables/kafka/message/equipable"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreatedStatusEventProvider(e Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(e.Id()))
	value := &equipable.StatusEvent[equipable.AttributeBody]{
		Id:   e.Id(),
		Type: equipable.StatusEventCreated,
		Body: equipable.AttributeBody{
			Strength:       e.Strength(),
			Dexterity:      e.Dexterity(),
			Intelligence:   e.Intelligence(),
			Luck:           e.Luck(),
			HP:             e.HP(),
			MP:             e.MP(),
			WeaponAttack:   e.WeaponAttack(),
			MagicAttack:    e.MagicAttack(),
			WeaponDefense:  e.WeaponDefense(),
			MagicDefense:   e.MagicDefense(),
			Accuracy:       e.Accuracy(),
			Avoidability:   e.Avoidability(),
			Hands:          e.Hands(),
			Speed:          e.Speed(),
			Jump:           e.Jump(),
			Slots:          e.Slots(),
			OwnerName:      e.OwnerName(),
			Locked:         e.Locked(),
			Spikes:         e.Spikes(),
			KarmaUsed:      e.KarmaUsed(),
			Cold:           e.Cold(),
			CanBeTraded:    e.CanBeTraded(),
			LevelType:      e.LevelType(),
			Level:          e.Level(),
			Experience:     e.Experience(),
			HammersApplied: e.HammersApplied(),
			Expiration:     e.Expiration(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func UpdatedStatusEventProvider(e Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(e.Id()))
	value := &equipable.StatusEvent[equipable.AttributeBody]{
		Id:   e.Id(),
		Type: equipable.StatusEventUpdated,
		Body: equipable.AttributeBody{
			Strength:       e.Strength(),
			Dexterity:      e.Dexterity(),
			Intelligence:   e.Intelligence(),
			Luck:           e.Luck(),
			HP:             e.HP(),
			MP:             e.MP(),
			WeaponAttack:   e.WeaponAttack(),
			MagicAttack:    e.MagicAttack(),
			WeaponDefense:  e.WeaponDefense(),
			MagicDefense:   e.MagicDefense(),
			Accuracy:       e.Accuracy(),
			Avoidability:   e.Avoidability(),
			Hands:          e.Hands(),
			Speed:          e.Speed(),
			Jump:           e.Jump(),
			Slots:          e.Slots(),
			OwnerName:      e.OwnerName(),
			Locked:         e.Locked(),
			Spikes:         e.Spikes(),
			KarmaUsed:      e.KarmaUsed(),
			Cold:           e.Cold(),
			CanBeTraded:    e.CanBeTraded(),
			LevelType:      e.LevelType(),
			Level:          e.Level(),
			Experience:     e.Experience(),
			HammersApplied: e.HammersApplied(),
			Expiration:     e.Expiration(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeletedStatusEventProvider(id uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(id))
	value := &equipable.StatusEvent[equipable.DeletedStatusEventBody]{
		Id:   id,
		Type: equipable.StatusEventDeleted,
		Body: equipable.DeletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
