package saga

import (
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// Re-export types from atlas-saga shared library
type (
	Type   = sharedsaga.Type
	Saga   = sharedsaga.Saga
	Status = sharedsaga.Status
	Action = sharedsaga.Action

	// Payload types
	AwardFamePayload = sharedsaga.AwardFamePayload
)

// Re-export constants from atlas-saga shared library
const (
	InventoryTransaction = sharedsaga.InventoryTransaction

	Pending   = sharedsaga.Pending
	Completed = sharedsaga.Completed
	Failed    = sharedsaga.Failed

	AwardFame = sharedsaga.AwardFame
)
