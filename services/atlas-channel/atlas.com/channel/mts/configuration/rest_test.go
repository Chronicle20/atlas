package configuration

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip. Every field is given a distinct non-zero value so Extract's
// zero-fallback logic (DefaultConfig substitution) never masks a mapping bug.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:                "1",
		ListingFee:        6000,
		CommissionRate:    0.09,
		CommissionBase:    600,
		MaxActiveListings: 12,
		MinLevel:          15,
		AuctionMinHours:   30,
		AuctionMaxHours:   180,
		FixedSaleHours:    200,
		PriceFloor:        120,
		PageSize:          20,
		MinBidIncrement:   2,
	}

	m := Extract(rm)

	rm2 := Transform(m)

	m2 := Extract(rm2)

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
