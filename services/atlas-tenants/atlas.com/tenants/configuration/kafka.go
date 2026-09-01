package configuration

import (
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

const (
	EventTopicConfigurationStatus topic.Token = "EVENT_TOPIC_CONFIGURATION_STATUS"
)

const (
	EventTypeRouteCreated           = "ROUTE_CREATED"
	EventTypeRouteUpdated           = "ROUTE_UPDATED"
	EventTypeRouteDeleted           = "ROUTE_DELETED"
	EventTypeVesselCreated          = "VESSEL_CREATED"
	EventTypeVesselUpdated          = "VESSEL_UPDATED"
	EventTypeVesselDeleted          = "VESSEL_DELETED"
	EventTypeInstanceRouteCreated   = "INSTANCE_ROUTE_CREATED"
	EventTypeInstanceRouteUpdated   = "INSTANCE_ROUTE_UPDATED"
	EventTypeInstanceRouteDeleted   = "INSTANCE_ROUTE_DELETED"
	EventTypeRpsRewardCreated       = "RPS_REWARD_CREATED"
	EventTypeRpsRewardUpdated       = "RPS_REWARD_UPDATED"
	EventTypeRpsRewardDeleted       = "RPS_REWARD_DELETED"
	EventTypeMtsConfigCreated       = "MTS_CONFIG_CREATED"
	EventTypeMtsConfigUpdated       = "MTS_CONFIG_UPDATED"
	EventTypeMtsConfigDeleted       = "MTS_CONFIG_DELETED"
	EventTypeTradeConfigCreated     = "TRADE_CONFIG_CREATED"
	EventTypeTradeConfigUpdated     = "TRADE_CONFIG_UPDATED"
	EventTypeTradeConfigDeleted     = "TRADE_CONFIG_DELETED"
	EventTypeRankingsCreated        = "RANKINGS_CREATED"
	EventTypeRankingsUpdated        = "RANKINGS_UPDATED"
	EventTypeRankingsDeleted        = "RANKINGS_DELETED"
	EventTypeKiteConfigCreated      = "KITE_CONFIG_CREATED"
	EventTypeKiteConfigUpdated      = "KITE_CONFIG_UPDATED"
	EventTypeKiteConfigDeleted      = "KITE_CONFIG_DELETED"
	EventTypePlayerNpcConfigCreated = "PLAYER_NPC_CONFIG_CREATED"
	EventTypePlayerNpcConfigUpdated = "PLAYER_NPC_CONFIG_UPDATED"
	EventTypePlayerNpcConfigDeleted = "PLAYER_NPC_CONFIG_DELETED"
	EventTypeImprintConfigCreated   = "IMPRINT_CONFIG_CREATED"
	EventTypeImprintConfigUpdated   = "IMPRINT_CONFIG_UPDATED"
	EventTypeImprintConfigDeleted   = "IMPRINT_CONFIG_DELETED"
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

// CreateMtsConfigStatusEventProvider creates a provider for mts config status events
func CreateMtsConfigStatusEventProvider(tenantId uuid.UUID, eventType string, configId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "mts-config",
		ResourceId:   configId,
	}
	return producer.SingleMessageProvider(key, value)
}

// CreateTradeConfigStatusEventProvider creates a provider for trade config status events
func CreateTradeConfigStatusEventProvider(tenantId uuid.UUID, eventType string, configId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "trade-config",
		ResourceId:   configId,
	}
	return producer.SingleMessageProvider(key, value)
}

// CreateImprintConfigStatusEventProvider creates a provider for imprint config status events
func CreateImprintConfigStatusEventProvider(tenantId uuid.UUID, eventType string, configId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "imprint-config",
		ResourceId:   configId,
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

// CreateRpsRewardStatusEventProvider creates a provider for rps-reward status events
func CreateRpsRewardStatusEventProvider(tenantId uuid.UUID, eventType string, rpsRewardId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "rps-reward",
		ResourceId:   rpsRewardId,
	}
	return producer.SingleMessageProvider(key, value)
}

// CreateKiteConfigStatusEventProvider creates a provider for kite-config status events
func CreateKiteConfigStatusEventProvider(tenantId uuid.UUID, eventType string, kiteConfigId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "kite-config",
		ResourceId:   kiteConfigId,
	}
	return producer.SingleMessageProvider(key, value)
}

// CreateRankingsStatusEventProvider creates a provider for rankings configuration status events
func CreateRankingsStatusEventProvider(tenantId uuid.UUID, eventType string, rankingsId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "rankings",
		ResourceId:   rankingsId,
	}
	return producer.SingleMessageProvider(key, value)
}

// CreatePlayerNpcConfigStatusEventProvider creates a provider for
// player-npcs configuration status events
func CreatePlayerNpcConfigStatusEventProvider(tenantId uuid.UUID, eventType string, playerNpcConfigId string) model.Provider[[]kafka.Message] {
	key := []byte(tenantId.String())
	value := ConfigurationStatusEvent{
		TenantId:     tenantId,
		Type:         eventType,
		ResourceType: "player-npcs",
		ResourceId:   playerNpcConfigId,
	}
	return producer.SingleMessageProvider(key, value)
}
