package configuration

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	EventTopicConfigurationStatus = "configuration.status"
	EventTypeRouteCreated         = "ROUTE_CREATED"
	EventTypeRouteUpdated         = "ROUTE_UPDATED"
	EventTypeRouteDeleted         = "ROUTE_DELETED"
	EventTypeVesselCreated             = "VESSEL_CREATED"
	EventTypeVesselUpdated             = "VESSEL_UPDATED"
	EventTypeVesselDeleted             = "VESSEL_DELETED"
	EventTypeInstanceRouteCreated      = "INSTANCE_ROUTE_CREATED"
	EventTypeInstanceRouteUpdated      = "INSTANCE_ROUTE_UPDATED"
	EventTypeInstanceRouteDeleted      = "INSTANCE_ROUTE_DELETED"
)

// ConfigurationStatusEvent is a generic event for configuration status changes
type ConfigurationStatusEvent struct {
	TenantId     uuid.UUID `json:"tenantId"`
	Type         string    `json:"type"`
	ResourceType string    `json:"resourceType"`
	ResourceId   string    `json:"resourceId"`
}

// CreateRouteStatusEventProvider creates a provider for route status events
func CreateRouteStatusEventProvider(tenantId uuid.UUID, eventType string, routeId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "route",
		ResourceId:   routeId,
	}
	return producer.SingleMessageProvider(key, value)
}

// CreateVesselStatusEventProvider creates a provider for vessel status events
func CreateVesselStatusEventProvider(tenantId uuid.UUID, eventType string, vesselId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "vessel",
		ResourceId:   vesselId,
	}
	return producer.SingleMessageProvider(key, value)
}

// CreateInstanceRouteStatusEventProvider creates a provider for instance route status events
func CreateInstanceRouteStatusEventProvider(tenantId uuid.UUID, eventType string, instanceRouteId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "instance-route",
		ResourceId:   instanceRouteId,
	}
	return producer.SingleMessageProvider(key, value)
}
