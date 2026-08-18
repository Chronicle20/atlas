package character

// PresenceState discriminates a character's liveness on the durable location
// record held by atlas-maps. It crosses the atlas-maps -> atlas-channel REST
// boundary as a wire string, which is why it lives here rather than in either
// service.
type PresenceState string

const (
	// PresenceStateOffline means the character is not logged in. It is the
	// zero value: an unwritten column, an absent REST attribute, and an
	// unrecognised value all resolve here, so /find fails toward
	// "not findable" rather than asserting a channel it cannot support.
	PresenceStateOffline PresenceState = "OFFLINE"

	// PresenceStateInField means the character is logged in and on a map.
	PresenceStateInField PresenceState = "IN_FIELD"

	// PresenceStateInCashShop means the character is logged in and inside the
	// cash shop. This covers the MTS as well: the ITC renders inside the
	// cash-shop CStage and emits the identical CHARACTER_ENTER event, so
	// atlas-maps cannot distinguish them and does not need to.
	PresenceStateInCashShop PresenceState = "IN_CASH_SHOP"
)

// ParsePresenceState converts a wire value into a PresenceState, resolving
// anything it does not recognise — including the empty string — to
// PresenceStateOffline.
func ParsePresenceState(s string) PresenceState {
	switch PresenceState(s) {
	case PresenceStateInField:
		return PresenceStateInField
	case PresenceStateInCashShop:
		return PresenceStateInCashShop
	default:
		return PresenceStateOffline
	}
}
