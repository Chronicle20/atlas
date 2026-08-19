package parcel

import "errors"

// Typed failure sentinels for the parcel custody state machine. Task 4's
// REST surface (and task-15's Kafka command consumer) map these via
// errors.Is to the client's PARCEL result arms — a malformed or out-of-turn
// request is rejected with the matching result, never a disconnect (design
// NFR: "never disconnect on a malformed request").
var (
	// ErrNotFound — no parcel exists with the requested id.
	ErrNotFound = errors.New("parcel not found")

	// ErrNotPending — the parcel is not in StatusPending, so it cannot be
	// received or discarded. Covers both "already resolved" (received/
	// discarded) and "already expired" — the second delivery of a replayed
	// receive/discard finds this and no-ops safely (NFR-3: award-once).
	ErrNotPending = errors.New("parcel is not pending")

	// ErrNotRecipient — the requesting character is not the parcel's
	// recipient.
	ErrNotRecipient = errors.New("requester is not the parcel's recipient")

	// ErrNotYetReceivable — the parcel is pending and owned by the
	// requester, but ReceivableAt has not yet passed.
	ErrNotYetReceivable = errors.New("parcel is not yet receivable")
)
