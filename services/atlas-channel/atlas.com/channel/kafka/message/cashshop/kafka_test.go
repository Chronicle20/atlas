package cashshop

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// The atlas-cashshop mirror of these bodies is a hand-maintained copy in a
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

// task-227 task 37 added TransactionId to RequestPurchaseCommandBody /
// PurchaseEventBody / ErrorEventBody in both services' hand-mirrored copies.
// Pin the channel side's shape too, so a rename on either side fails loudly.
// Zero UUID means "no correlation" -- what atlas-channel sends today, since
// it is not minting real ids until task 38/39 -- and must round-trip as the
// all-zero UUID string, not an omitted field (omitempty is a no-op on
// uuid.UUID's [16]byte).
func TestRequestPurchaseCommandBodyWireShape(t *testing.T) {
	b, err := json.Marshal(RequestPurchaseCommandBody{Currency: 1, SerialNumber: 9001})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"transactionId":"00000000-0000-0000-0000-000000000000","currency":1,"serialNumber":9001}`
	if string(b) != want {
		t.Fatalf("wire shape drifted:\n got %s\nwant %s", b, want)
	}
}

func TestChannelPurchaseEventBodyDecodesTransactionId(t *testing.T) {
	txId := uuid.MustParse("44444444-5555-6666-7777-888888888888")
	raw := `{"templateId":5000000,"price":4000,"compartmentId":"33333333-4444-5555-6666-777777777777","assetId":42,"itemId":0,"transactionId":"44444444-5555-6666-7777-888888888888"}`
	var body PurchaseEventBody
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatal(err)
	}
	if body.TransactionId != txId {
		t.Fatalf("transactionId did not decode: got %s want %s", body.TransactionId, txId)
	}
}

func TestChannelErrorEventBodyDecodesTransactionId(t *testing.T) {
	txId := uuid.MustParse("55555555-6666-7777-8888-999999999999")
	raw := `{"error":"NOT_ENOUGH_CASH","transactionId":"55555555-6666-7777-8888-999999999999"}`
	var body ErrorEventBody
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatal(err)
	}
	if body.TransactionId != txId {
		t.Fatalf("transactionId did not decode: got %s want %s", body.TransactionId, txId)
	}
}
