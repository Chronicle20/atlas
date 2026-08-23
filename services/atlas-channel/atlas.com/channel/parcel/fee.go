package parcel

import "math"

// Duey parcel constants. See design.md §6.3 and §6.4.
const (
	// SendSurcharge is the flat meso cost added to a non-quick (NPC arm)
	// send, on top of the tiered fee.
	SendSurcharge = uint32(5000)
	// MaxParcelMeso is the per-parcel meso ceiling the client's own input
	// dialog enforces; re-enforced server-side.
	MaxParcelMeso = uint32(100_000_000)
	// MesoLimitLevel is the character level at or below which
	// MesoLimitAmount applies to a single send.
	MesoLimitLevel = byte(15)
	// MesoLimitAmount is the per-transaction meso ceiling for characters
	// at or below MesoLimitLevel.
	MesoLimitAmount = uint32(1_000_000)
	// MaxMessageLength is the maximum length of a quick-delivery message.
	MaxMessageLength = 100
	// MailboxCapacity is the maximum number of parcels a mailbox may hold.
	MailboxCapacity = 10
)

// Fee computes the tiered Duey delivery fee for mesoAmount.
//
// This deliberately uses float64 arithmetic rather than integer arithmetic
// (design §6.3 contradicts NFR-8 on this point). The client computes the
// same fee in IEEE-754 double and quotes it to the player in the confirm
// dialog before the packet is ever sent (v72 sub_6590A1 @0x6590A1, v83
// sub_6EEDFE @0x6EEDFE). If the server charged an integer-derived figure
// while the client quoted a double-derived one, the player would be
// charged a number they were never shown. Matching the client's formula
// exactly, tier for tier and rate for rate, is the only way to keep the
// quote and the charge identical.
func Fee(mesoAmount uint32) uint32 {
	m := float64(mesoAmount)
	switch {
	case mesoAmount >= 100_000_000:
		return uint32(m * 0.06)
	case mesoAmount >= 25_000_000:
		return uint32(m * 0.05)
	case mesoAmount >= 10_000_000:
		return uint32(m * 0.04)
	case mesoAmount >= 5_000_000:
		return uint32(m * 0.03)
	case mesoAmount >= 1_000_000:
		return uint32(m * 0.018)
	case mesoAmount >= 100_000:
		return uint32(m * 0.008)
	default:
		return 0
	}
}

// TotalCost computes mesoAmount + Fee(mesoAmount) + (quick ? 0 : SendSurcharge)
// in uint64 to avoid overflow while summing. The returned bool is false when
// the sum exceeds math.MaxUint32, since the meso ledger operates in uint32.
func TotalCost(mesoAmount uint32, quick bool) (uint64, bool) {
	total := uint64(mesoAmount) + uint64(Fee(mesoAmount))
	if !quick {
		total += uint64(SendSurcharge)
	}
	return total, total <= math.MaxUint32
}
