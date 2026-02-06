package configuration

import "github.com/google/uuid"

const (
	EnvEventTopicConfigurationStatus = "EVENT_TOPIC_CONFIGURATION_STATUS"
)

type StatusEvent struct {
	TenantId     uuid.UUID `json:"tenantId"`
	Type         string    `json:"type"`
	ResourceType string    `json:"resourceType"`
	ResourceId   string    `json:"resourceId"`
}
