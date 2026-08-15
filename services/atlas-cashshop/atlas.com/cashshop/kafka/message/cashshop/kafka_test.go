package cashshop

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// The atlas-channel mirror of these bodies is a hand-maintained copy in a
// separate Go module — the json tags are the ONLY contract. Pin them so a
// rename here fails loudly instead of silently dropping fields at runtime.
func TestOpenSurpriseCommandBodyWireShape(t *testing.T) {
	b, err := json.Marshal(OpenSurpriseCommandBody{
		TransactionId: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		AccountId:     10,
		CashId:        1234567890,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"transactionId":"00000000-0000-0000-0000-000000000001","accountId":10,"cashId":1234567890}`
	if string(b) != want {
		t.Fatalf("wire shape drifted:\n got %s\nwant %s", b, want)
	}
}

func TestSurpriseOpenedEventBodyWireShape(t *testing.T) {
	b, err := json.Marshal(SurpriseOpenedEventBody{
		CompartmentId:    uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		BoxCashId:        1234567890,
		BoxRemaining:     2,
		RewardAssetId:    77,
		RewardTemplateId: 5222001,
		RewardCount:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"compartmentId":"00000000-0000-0000-0000-000000000002","boxCashId":1234567890,"boxRemaining":2,"rewardAssetId":77,"rewardTemplateId":5222001,"rewardCount":1}`
	if string(b) != want {
		t.Fatalf("wire shape drifted:\n got %s\nwant %s", b, want)
	}
}

func TestSurpriseFailedEventBodyWireShape(t *testing.T) {
	b, err := json.Marshal(SurpriseFailedEventBody{Reason: "LOCKER_FULL"})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"reason":"LOCKER_FULL"}` {
		t.Fatalf("wire shape drifted: %s", b)
	}
}

// The channel decodes RequestPurchaseCommandBody / PurchaseEventBody /
// ErrorEventBody from hand-mirrored copies (kafka/message/cashshop/kafka.go
// in atlas-channel), so a JSON tag change on one side is a silent field drop
// on the other. Pin the wire shape task-227 task 37 added: TransactionId is
// an opaque correlation id (zero UUID means "no correlation" -- every
// existing caller today) echoed back on both outcome events so a caller
// juggling multiple concurrent purchases for one character can tell them
// apart.
func TestRequestPurchaseCommandBodyWireShape(t *testing.T) {
	id := uuid.MustParse("22222222-3333-4444-5555-666666666666")
	b, err := json.Marshal(RequestPurchaseCommandBody{TransactionId: id, Currency: 1, SerialNumber: 9001})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"transactionId":"22222222-3333-4444-5555-666666666666","currency":1,"serialNumber":9001}`
	if string(b) != want {
		t.Fatalf("wire shape drifted:\n got %s\nwant %s", b, want)
	}
}

func TestPurchaseEventBodyWireShape(t *testing.T) {
	compartmentId := uuid.MustParse("33333333-4444-5555-6666-777777777777")
	txId := uuid.MustParse("44444444-5555-6666-7777-888888888888")
	b, err := json.Marshal(PurchaseEventBody{
		TemplateId:    5000000,
		Price:         4000,
		CompartmentId: compartmentId,
		AssetId:       42,
		TransactionId: txId,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"templateId":5000000,"price":4000,"compartmentId":"33333333-4444-5555-6666-777777777777","assetId":42,"transactionId":"44444444-5555-6666-7777-888888888888"}`
	if string(b) != want {
		t.Fatalf("wire shape drifted:\n got %s\nwant %s", b, want)
	}
}

func TestErrorEventBodyWireShape(t *testing.T) {
	txId := uuid.MustParse("55555555-6666-7777-8888-999999999999")
	b, err := json.Marshal(ErrorEventBody{Error: "NOT_ENOUGH_CASH", TransactionId: txId})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"error":"NOT_ENOUGH_CASH","transactionId":"55555555-6666-7777-8888-999999999999"}`
	if string(b) != want {
		t.Fatalf("wire shape drifted:\n got %s\nwant %s", b, want)
	}
}

// TestErrorEventBodyZeroTransactionIdWireShape pins that the zero UUID -- what
// every existing non-purchase ErrorStatusEventProvider caller sends today
// (storage-increase, character-slot-increase arms) -- round-trips as the
// all-zero UUID string, not an omitted field: omitempty is a no-op on
// uuid.UUID ([16]byte), so backward compatibility here means "always present,
// zero means no correlation" rather than "absent means no correlation".
func TestErrorEventBodyZeroTransactionIdWireShape(t *testing.T) {
	b, err := json.Marshal(ErrorEventBody{Error: "UNKNOWN_ERROR"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"error":"UNKNOWN_ERROR","transactionId":"00000000-0000-0000-0000-000000000000"}`
	if string(b) != want {
		t.Fatalf("wire shape drifted:\n got %s\nwant %s", b, want)
	}
}

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
