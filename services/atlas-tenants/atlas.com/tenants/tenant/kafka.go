package tenant

import (
	"encoding/json"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"log"
)

const (
	EVENT_TOPIC_TENANT_STATUS = "tenant.status"
)

// StatusEvent is a generic event for tenant status changes
type StatusEvent[T any] struct {
	TenantId uuid.UUID `json:"tenantId"`
	Type     string    `json:"type"`
	Body     T         `json:"body"`
}

// StatusEventCreatedBody is the body for a tenant created event
type StatusEventCreatedBody struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
}

// StatusEventUpdatedBody is the body for a tenant updated event
type StatusEventUpdatedBody struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
}

// StatusEventDeletedBody is the body for a tenant deleted event
type StatusEventDeletedBody struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
}

// CreateStatusEventProvider creates a provider for tenant status events
func CreateStatusEventProvider(tenantId uuid.UUID, eventType string, name string, region string, majorVersion uint16, minorVersion uint16) model.Provider[[]kafka.Message] {
	var body interface{}
	switch eventType {
	case "CREATED":
		body = StatusEventCreatedBody{
			Name:         name,
			Region:       region,
			MajorVersion: majorVersion,
			MinorVersion: minorVersion,
		}
	case "UPDATED":
		body = StatusEventUpdatedBody{
			Name:         name,
			Region:       region,
			MajorVersion: majorVersion,
			MinorVersion: minorVersion,
		}
	case "DELETED":
		body = StatusEventDeletedBody{
			Name:         name,
			Region:       region,
			MajorVersion: majorVersion,
			MinorVersion: minorVersion,
		}
	}

	key := []byte(tenantId.String())
	value := StatusEvent[interface{}]{
		TenantId: tenantId,
		Type:     eventType,
		Body:     body,
	}

	return model.FixedProvider([]kafka.Message{
		{
			Topic: EVENT_TOPIC_TENANT_STATUS,
			Key:   key,
			Value: mustMarshal(value),
		},
	})
}

// mustMarshal marshals the value to JSON and panics on error
func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}
	return data
}
