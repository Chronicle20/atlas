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
	case TradeTransaction:
		// task-205. A trade_settlement composite expands into
		// release_from_character / accept_to_character / award_mesos — a subset of
		// the storage step set — so the storage mapping applies verbatim and is
		// reused rather than duplicated.
		//
		// Caveat this mapping does NOT resolve: award_mesos returns
		// NOT_ENOUGH_MESOS for either leg regardless of sign. The negative
		// (giver) leg genuinely fails on insufficiency; the positive (receiver)
		// leg fails on the meso cap, and is mislabelled here. That is tolerable
		// only because atlas-trades collapses EVERY TradeTransaction failure into
		// the client's LEAVE 8 "Trade unsuccessful" (design §5.3) — the code is
		// diagnostic detail on the FAILED event, never a branch the client sees.
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
