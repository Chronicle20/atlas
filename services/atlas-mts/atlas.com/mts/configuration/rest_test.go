package configuration

import (
	"reflect"
	"testing"
)

func TestTransformRoundTrip(t *testing.T) {
	m := Model{
		listingFee:        5001,
		commissionRate:    0.11,
		commissionBase:    501,
		maxActiveListings: 11,
		minLevel:          12,
		auctionMinHours:   25,
		auctionMaxHours:   169,
		fixedSaleHours:    170,
		priceFloor:        111,
		pageSize:          17,
		minBidIncrement:   2,
	}

	got := Extract(Transform(m))

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
