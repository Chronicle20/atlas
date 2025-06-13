package compartment

import (
	"atlas-cashshop/kafka/message/cashshop/compartment"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

// CreateStatusEventProvider creates a provider for compartment creation events
// According to the requirements, it should always include compartmentId and type, and include capacity in the body
func CreateStatusEventProvider(compartmentId string, compartmentType string, capacity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(0) // Using 0 as the key since we don't have a numeric ID to use
	value := &compartment.StatusEvent[compartment.StatusEventCreatedBody]{
		CompartmentId:   compartmentId,
		Type:            compartment.StatusEventTypeCreated,
		CompartmentType: compartmentType,
		Body: compartment.StatusEventCreatedBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// UpdateStatusEventProvider creates a provider for compartment update events
func UpdateStatusEventProvider(compartmentId string, compartmentType string, capacity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(0) // Using 0 as the key since we don't have a numeric ID to use
	value := &compartment.StatusEvent[compartment.StatusEventUpdatedBody]{
		CompartmentId:   compartmentId,
		Type:            compartment.StatusEventTypeUpdated,
		CompartmentType: compartmentType,
		Body: compartment.StatusEventUpdatedBody{
			Capacity: capacity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// DeleteStatusEventProvider creates a provider for compartment deletion events
// According to the requirements, it should always include compartmentId and type, and have an empty body
func DeleteStatusEventProvider(compartmentId string, compartmentType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(0) // Using 0 as the key since we don't have a numeric ID to use
	value := &compartment.StatusEvent[compartment.StatusEventDeletedBody]{
		CompartmentId:   compartmentId,
		Type:            compartment.StatusEventTypeDeleted,
		CompartmentType: compartmentType,
		Body:            compartment.StatusEventDeletedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
