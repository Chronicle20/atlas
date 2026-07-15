// Package merchant holds the merchant-shop enums shared across service
// boundaries: atlas-merchant persists them and atlas-channel interprets them
// off the REST/Kafka wire. Both sides MUST derive from these constants —
// hand-mirrored byte values drift.
package merchant

// ShopType discriminates the two merchant shop kinds: a personal store run
// by the character in person (514-family permit) and a hired merchant that
// sells autonomously (503-family permit).
type ShopType byte

const (
	ShopTypeCharacter     ShopType = 1
	ShopTypeHiredMerchant ShopType = 2
)

// ShopState is the shop lifecycle state machine:
// Draft (owner-attached setup) -> Open (selling) <-> Maintenance (owner
// managing, visitors ejected) -> Closed (terminal).
type ShopState byte

const (
	ShopStateDraft       ShopState = 1
	ShopStateOpen        ShopState = 2
	ShopStateMaintenance ShopState = 3
	ShopStateClosed      ShopState = 4
)
