package shop

type ShopType byte

const (
	CharacterShop ShopType = 1
	HiredMerchant ShopType = 2
)

type State byte

const (
	Draft       State = 1
	Open        State = 2
	Maintenance State = 3
	Closed      State = 4
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
