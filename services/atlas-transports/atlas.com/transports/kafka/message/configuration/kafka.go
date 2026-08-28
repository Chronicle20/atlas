package configuration

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/google/uuid"
)

const (
	EnvEventTopicConfigurationStatus topic.Token = "EVENT_TOPIC_CONFIGURATION_STATUS"
)

type StatusEvent struct {
	TenantId     uuid.UUID `json:"tenantId"`
	Type         string    `json:"type"`
	ResourceType string    `json:"resourceType"`
	ResourceId   string    `json:"resourceId"`
}
