package compartment

import (
	"atlas-inventory/kafka/message/compartment"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func CreatedEventStatusProvider(transactionId uuid.UUID, id uuid.UUID, characterId uint32, inventoryType inventory.Type, capacity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.CreatedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: id,
		Type:          compartment.StatusEventTypeCreated,
		Body: compartment.CreatedStatusEventBody{
			Type:     byte(inventoryType),
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeletedEventStatusProvider(transactionId uuid.UUID, id uuid.UUID, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.DeletedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: id,
		Type:          compartment.StatusEventTypeDeleted,
		Body:          compartment.DeletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func CapacityChangedEventStatusProvider(transactionId uuid.UUID, id uuid.UUID, characterId uint32, inventoryType inventory.Type, capacity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.CapacityChangedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: id,
		Type:          compartment.StatusEventTypeCapacityChanged,
		Body: compartment.CapacityChangedEventBody{
			Type:     byte(inventoryType),
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ReservedEventStatusProvider(transactionId uuid.UUID, id uuid.UUID, characterId uint32, itemId uint32, slot int16, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.ReservedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: id,
		Type:          compartment.StatusEventTypeReserved,
		Body: compartment.ReservedEventBody{
			TransactionId: transactionId, // TODO this needs removal from dependent services
			ItemId:        itemId,
			Slot:          slot,
			Quantity:      quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ReservationCancelledEventStatusProvider(transactionId uuid.UUID, id uuid.UUID, characterId uint32, itemId uint32, slot int16, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.ReservationCancelledEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: id,
		Type:          compartment.StatusEventTypeReservationCancelled,
		Body: compartment.ReservationCancelledEventBody{
			ItemId:   itemId,
			Slot:     slot,
			Quantity: quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func MergeCompleteEventStatusProvider(transactionId uuid.UUID, id uuid.UUID, characterId uint32, inventoryType inventory.Type) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.MergeCompleteEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: id,
		Type:          compartment.StatusEventTypeMergeComplete,
		Body: compartment.MergeCompleteEventBody{
			Type: byte(inventoryType),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func SortCompleteEventStatusProvider(transactionId uuid.UUID, id uuid.UUID, characterId uint32, inventoryType inventory.Type) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.SortCompleteEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: id,
		Type:          compartment.StatusEventTypeSortComplete,
		Body: compartment.SortCompleteEventBody{
			Type: byte(inventoryType),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AcceptedEventStatusProvider(transactionId uuid.UUID, id uuid.UUID, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.AcceptedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: id,
		Type:          compartment.StatusEventTypeAccepted,
		Body: compartment.AcceptedEventBody{
			TransactionId: transactionId, // TODO this needs removal from dependent services
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ReleasedEventStatusProvider(transactionId uuid.UUID, id uuid.UUID, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.ReleasedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: id,
		Type:          compartment.StatusEventTypeReleased,
		Body: compartment.ReleasedEventBody{
			TransactionId: transactionId, // TODO this needs removal from dependent services
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ErrorEventStatusProvider(transactionId uuid.UUID, id uuid.UUID, characterId uint32, errorCode string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.ErrorEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: id,
		Type:          compartment.StatusEventTypeError,
		Body: compartment.ErrorEventBody{
			ErrorCode:     errorCode,
			TransactionId: transactionId, // TODO this needs removal from dependent services
		},
	}
	return producer.SingleMessageProvider(key, value)
}
