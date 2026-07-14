package shop

import merchantconst "github.com/Chronicle20/atlas/libs/atlas-constants/merchant"

// ShopType and State alias the shared atlas-constants enums so every consumer
// of the REST/Kafka contract (atlas-channel in particular) derives the same
// byte values at compile time instead of hand-mirroring them.
type ShopType = merchantconst.ShopType

const (
	CharacterShop = merchantconst.ShopTypeCharacter
	HiredMerchant = merchantconst.ShopTypeHiredMerchant
)

type State = merchantconst.ShopState

const (
	Draft       = merchantconst.ShopStateDraft
	Open        = merchantconst.ShopStateOpen
	Maintenance = merchantconst.ShopStateMaintenance
	Closed      = merchantconst.ShopStateClosed
)

type CloseReason byte

const (
	CloseReasonNone          CloseReason = 0
	CloseReasonSoldOut       CloseReason = 1
	CloseReasonManualClose   CloseReason = 2
	CloseReasonDisconnect    CloseReason = 3
	CloseReasonExpired       CloseReason = 4
	CloseReasonServerRestart CloseReason = 5
	CloseReasonEmpty         CloseReason = 6
)

// LogoutOutcome is what the character-logout reaper should do with one of the
// character's shops.
type LogoutOutcome byte

const (
	// LogoutNone leaves the shop untouched (Open hired merchants run
	// owner-detached; Closed shops are history).
	LogoutNone LogoutOutcome = iota
	// LogoutClose fully closes the shop (reason Disconnect).
	LogoutClose
	// LogoutExitMaintenance returns a hired merchant from the owner's
	// management view to autonomous running (auto-closing it when empty).
	LogoutExitMaintenance
)

// LogoutAction implements the logout policy (merchant-lifecycle-audit Q5):
// Draft shops of either type are owner-attached setup sessions and close;
// personal shops close in every live state; an Open hired merchant survives;
// one caught in Maintenance reverts to running via exit-maintenance —
// leaving it in Maintenance would strand it unenterable forever.
func LogoutAction(shopType ShopType, state State) LogoutOutcome {
	if state == Closed {
		return LogoutNone
	}
	if shopType == CharacterShop {
		return LogoutClose
	}
	switch state {
	case Draft:
		return LogoutClose
	case Maintenance:
		return LogoutExitMaintenance
	default:
		return LogoutNone
	}
}
