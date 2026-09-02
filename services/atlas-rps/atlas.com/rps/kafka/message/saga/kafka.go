// Package saga carries the Kafka topic name atlas-rps uses to submit sagas
// to atlas-saga-orchestrator. atlas-rps is a saga producer only here (the
// payout saga on Collect - Task 12); it does not consume the saga status
// event topic, so no status-event envelope is defined.
package saga

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	// EnvCommandTopic names the environment variable holding
	// atlas-saga-orchestrator's command topic. Mirrors
	// atlas-npc-conversations/kafka/message/saga.EnvCommandTopic.
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_SAGA"
)
