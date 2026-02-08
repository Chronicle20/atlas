package drop

import (
	messageDropKafka "atlas-drops/kafka/message/drop"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func equipmentDataFromModel(m Model) messageDropKafka.EquipmentData {
	return messageDropKafka.EquipmentData{
		Strength:      m.Strength(),
		Dexterity:     m.Dexterity(),
		Intelligence:  m.Intelligence(),
		Luck:          m.Luck(),
		Hp:            m.Hp(),
		Mp:            m.Mp(),
		WeaponAttack:  m.WeaponAttack(),
		MagicAttack:   m.MagicAttack(),
		WeaponDefense: m.WeaponDefense(),
		MagicDefense:  m.MagicDefense(),
		Accuracy:      m.Accuracy(),
		Avoidability:  m.Avoidability(),
		Hands:         m.Hands(),
		Speed:         m.Speed(),
		Jump:          m.Jump(),
		Slots:         m.Slots(),
	}
}

func createdEventStatusProvider(drop Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(drop.Id()))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventCreatedBody]{
		TransactionId: drop.TransactionId(),
		WorldId:       drop.WorldId(),
		ChannelId:     drop.ChannelId(),
		MapId:         drop.MapId(),
		Instance:      drop.Instance(),
		DropId:        drop.Id(),
		Type:          messageDropKafka.StatusEventTypeCreated,
		Body: messageDropKafka.StatusEventCreatedBody{
			ItemId:          drop.ItemId(),
			Quantity:        drop.Quantity(),
			Meso:            drop.Meso(),
			Type:            drop.Type(),
			X:               drop.X(),
			Y:               drop.Y(),
			OwnerId:         drop.OwnerId(),
			OwnerPartyId:    drop.OwnerPartyId(),
			DropTime:        drop.DropTime(),
			DropperUniqueId: drop.DropperId(),
			DropperX:        drop.DropperX(),
			DropperY:        drop.DropperY(),
			PlayerDrop:      drop.PlayerDrop(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func expiredEventStatusProvider(transactionId uuid.UUID, field field.Model, dropId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventExpiredBody]{
		TransactionId: transactionId,
		WorldId:       field.WorldId(),
		ChannelId:     field.ChannelId(),
		MapId:         field.MapId(),
		Instance:      field.Instance(),
		DropId:        dropId,
		Type:          messageDropKafka.StatusEventTypeExpired,
		Body:          messageDropKafka.StatusEventExpiredBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func pickedUpEventStatusProvider(transactionId uuid.UUID, field field.Model, d Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(d.Id()))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventPickedUpBody]{
		TransactionId: transactionId,
		WorldId:       field.WorldId(),
		ChannelId:     field.ChannelId(),
		MapId:         field.MapId(),
		Instance:      field.Instance(),
		DropId:        d.Id(),
		Type:          messageDropKafka.StatusEventTypePickedUp,
		Body: messageDropKafka.StatusEventPickedUpBody{
			CharacterId:   characterId,
			ItemId:        d.ItemId(),
			Quantity:      d.Quantity(),
			Meso:          d.Meso(),
			PetSlot:       d.PetSlot(),
			EquipmentData: equipmentDataFromModel(d),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func reservedEventStatusProvider(transactionId uuid.UUID, field field.Model, d Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(d.Id()))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventReservedBody]{
		TransactionId: transactionId,
		WorldId:       field.WorldId(),
		ChannelId:     field.ChannelId(),
		MapId:         field.MapId(),
		Instance:      field.Instance(),
		DropId:        d.Id(),
		Type:          messageDropKafka.StatusEventTypeReserved,
		Body: messageDropKafka.StatusEventReservedBody{
			CharacterId:   d.OwnerId(),
			ItemId:        d.ItemId(),
			Quantity:      d.Quantity(),
			Meso:          d.Meso(),
			EquipmentData: equipmentDataFromModel(d),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func reservationFailureEventStatusProvider(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventReservationFailureBody]{
		TransactionId: transactionId,
		WorldId:       field.WorldId(),
		ChannelId:     field.ChannelId(),
		MapId:         field.MapId(),
		Instance:      field.Instance(),
		DropId:        dropId,
		Type:          messageDropKafka.StatusEventTypeReservationFailure,
		Body: messageDropKafka.StatusEventReservationFailureBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
