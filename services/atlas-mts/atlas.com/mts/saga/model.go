package saga

import (
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// Re-export the types and constants atlas-mts needs from the shared atlas-saga
// library. Mirrors the character-factory saga package: the service constructs
// sagas against these local aliases and emits them to COMMAND_TOPIC_SAGA.
type (
	Type   = sharedsaga.Type
	Saga   = sharedsaga.Saga
	Status = sharedsaga.Status
	Action = sharedsaga.Action
	Step   = sharedsaga.Step[any]

	// Payload types used by the list flow.
	AwardMesosPayload    = sharedsaga.AwardMesosPayload
	TransferToMtsPayload = sharedsaga.TransferToMtsPayload
)

const (
	// Saga types
	MtsOperation = sharedsaga.MtsOperation

	// Status constants
	Pending   = sharedsaga.Pending
	Completed = sharedsaga.Completed
	Failed    = sharedsaga.Failed

	// Action constants
	AwardMesos    = sharedsaga.AwardMesos
	TransferToMts = sharedsaga.TransferToMts
)
