package parcel

// RejectReason names why a Duey send request was rejected. Task 17's remote
// checks (recipient resolution, the same-account check, mailbox capacity,
// the ticket check) add further reject reasons; this package covers only
// the checks that need no round trip.
type RejectReason string

const (
	RejectNone             RejectReason = ""
	RejectIncorrectRequest RejectReason = "INCORRECT_REQUEST"
	RejectMesoLimit        RejectReason = "MESO_LIMIT"
	RejectNotEnoughMesos   RejectReason = "NOT_ENOUGH_MESOS"
)

// SendInput carries the fields of a Duey send request that ValidateSend can
// check without a remote lookup.
type SendInput struct {
	MesoAmount  uint32
	Quantity    uint16
	Quick       bool
	Message     string
	SenderLevel byte
	SenderMeso  uint32
}

// ValidateSend runs the local-only checks on a Duey send request. The parcel
// cap and overflow checks run before the affordability check, so an absurd
// meso amount reports RejectIncorrectRequest rather than
// RejectNotEnoughMesos.
func ValidateSend(in SendInput) RejectReason {
	if in.MesoAmount == 0 && in.Quantity == 0 {
		return RejectIncorrectRequest
	}
	if in.MesoAmount > MaxParcelMeso {
		return RejectIncorrectRequest
	}

	total, ok := TotalCost(in.MesoAmount, in.Quick)
	if !ok {
		return RejectIncorrectRequest
	}

	if in.SenderLevel <= MesoLimitLevel && in.MesoAmount > MesoLimitAmount {
		return RejectMesoLimit
	}

	// The NPC send arm carries no message at all (design §0 finding C), so
	// the length check only applies to the quick-delivery arm.
	if in.Quick && len(in.Message) > MaxMessageLength {
		return RejectIncorrectRequest
	}

	if uint64(in.SenderMeso) < total {
		return RejectNotEnoughMesos
	}

	return RejectNone
}
