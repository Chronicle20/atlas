package saga

import (
	sagaMsg "atlas-saga-orchestrator/kafka/message/saga"
)

// DetermineErrorCode determines the appropriate error code based on the saga type and failed step.
// This is used to provide context-appropriate error messages to clients.
func DetermineErrorCode(s Saga, failedStep Step[any]) string {
	switch s.SagaType() {
	case StorageOperation:
		return determineStorageErrorCode(failedStep)
	default:
		return sagaMsg.ErrorCodeUnknown
	}
}

// determineStorageErrorCode determines the error code for storage operation failures.
func determineStorageErrorCode(step Step[any]) string {
	switch step.Action() {
	case AwardMesos:
		// AwardMesos with negative amount is a fee charge
		// If this fails, it means the character doesn't have enough mesos
		return sagaMsg.ErrorCodeNotEnoughMesos
	case AcceptToCharacter:
		// Character inventory couldn't accept the item (inventory full)
		return sagaMsg.ErrorCodeInventoryFull
	case AcceptToStorage:
		// Storage couldn't accept the item (storage full)
		return sagaMsg.ErrorCodeStorageFull
	default:
		return sagaMsg.ErrorCodeUnknown
	}
}
