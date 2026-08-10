package cashshop

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// The channel decodes these bodies from a duplicated struct definition, so a
// JSON tag change on one side is a silent field drop on the other. Pin the
// wire shape.
func TestCouponCommandWireShape(t *testing.T) {
	b, err := json.Marshal(Command[RequestCouponRedemptionCommandBody]{
		CharacterId: 7,
		Type:        CommandTypeRequestCouponRedemption,
		Body:        RequestCouponRedemptionCommandBody{Code: "MAPLE2026"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"characterId":7,"type":"REQUEST_COUPON_REDEMPTION","body":{"code":"MAPLE2026"}}`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}
}

func TestCouponRedeemedWireShape(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	b, err := json.Marshal(StatusEvent[CouponRedeemedBody]{
		CharacterId: 7,
		Type:        StatusEventTypeCouponRedeemed,
		Body:        CouponRedeemedBody{CompartmentId: id, AssetIds: []uint32{9}, MaplePoints: 500, Credit: 250},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"characterId":7,"type":"COUPON_REDEEMED","body":{"compartmentId":"11111111-2222-3333-4444-555555555555","assetIds":[9],"maplePoints":500,"credit":250}}`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}
}

func TestCouponFailedWireShape(t *testing.T) {
	b, err := json.Marshal(StatusEvent[CouponFailedBody]{
		CharacterId: 7,
		Type:        StatusEventTypeCouponFailed,
		Body:        CouponFailedBody{Error: "COUPON_EXPIRED"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"characterId":7,"type":"COUPON_FAILED","body":{"error":"COUPON_EXPIRED"}}`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}
}
