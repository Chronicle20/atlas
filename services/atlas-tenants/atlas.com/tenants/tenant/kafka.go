package tenant

import (
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

const (
	EventTopicTenantStatus = "tenant.status"
	EventTypeCreated       = "CREATED"
	EventTypeUpdated       = "UPDATED"
	EventTypeDeleted       = "DELETED"
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
	Environment  string `json:"environment"`
}

// StatusEventUpdatedBody is the body for a tenant updated event
type StatusEventUpdatedBody struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
	Environment  string `json:"environment"`
}

// StatusEventDeletedBody is the body for a tenant deleted event
type StatusEventDeletedBody struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
	Environment  string `json:"environment"`
}

// CreateStatusEventProvider creates a provider for tenant status events
func CreateStatusEventProvider(tenantId uuid.UUID, eventType string, name string, region string, majorVersion uint16, minorVersion uint16, environment string) model.Provider[[]kafka.Message] {
	var body interface{}
	switch eventType {
	case "CREATED":
		body = StatusEventCreatedBody{
			Name:         name,
			Region:       region,
			MajorVersion: majorVersion,
			MinorVersion: minorVersion,
			Environment:  environment,
		}
	case "UPDATED":
		body = StatusEventUpdatedBody{
			Name:         name,
			Region:       region,
			MajorVersion: majorVersion,
			MinorVersion: minorVersion,
			Environment:  environment,
		}
	case "DELETED":
		body = StatusEventDeletedBody{
			Name:         name,
			Region:       region,
			MajorVersion: majorVersion,
			MinorVersion: minorVersion,
			Environment:  environment,
		}
	}

	key := []byte(tenantId.String())
	value := StatusEvent[interface{}]{
		TenantId: tenantId,
		Type:     eventType,
		Body:     body,
	}
	return producer.SingleMessageProvider(key, value)
}
