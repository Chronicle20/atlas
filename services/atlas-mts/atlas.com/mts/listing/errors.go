package listing

import "errors"

// Typed failure sentinels for the buy/bid validation paths. The Kafka command
// consumer maps these (via errors.Is) to the client's CITC::NoticeFailReason
// codes so the player sees a specific message instead of the generic
// "Failed to purchase the item" (design: task-102 descriptive failure
// notices; codes IDA-verified identical across gms v83/v84/v87/v95).
var (
	// ErrInsufficientPrepaid — the buyer/bidder's prepaid NX cannot cover the
	// marked-up amount (client reason 'B' = 66, "You do not have enough NX").
	ErrInsufficientPrepaid = errors.New("insufficient prepaid NX")

	// ErrListingUnavailable — the listing is not active (already sold,
	// cancelled, expired, or lost a race) or is the wrong sale type for the
	// attempted operation (client reason 'Q' = 81, "The item has been sold").
	ErrListingUnavailable = errors.New("listing unavailable")

	// ErrConsecutiveBid — the bidder is already the current high bidder, so bidding
	// again against themselves is rejected. It maps to the generic bid-failure reason
	// so the channel writes the client's bare BidAuctionFailed arm ("you cannot make
	// a consecutive bid").
	ErrConsecutiveBid = errors.New("consecutive bid by the current high bidder")
)
