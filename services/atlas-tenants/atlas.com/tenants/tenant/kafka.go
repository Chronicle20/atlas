package tenant

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
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
	return producer.SingleMessageProvider(key, value)
}
