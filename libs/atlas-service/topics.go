// Package service's topic tokens. These four reads are intentionally
// optional -- an unset variable degrades to legacy single-environment
// mode rather than failing the process -- so they use os.LookupEnv
// directly rather than topic.EnvProvider, whose contract is now fatal.
package service

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvEventTopicConfigurationEnvironmentStatus topic.Token = "EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS"
	EnvEventTopicConfigurationTenantStatus      topic.Token = "EVENT_TOPIC_CONFIGURATION_TENANT_STATUS"
	EnvEventTopicConfigurationServiceStatus     topic.Token = "EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"
)
