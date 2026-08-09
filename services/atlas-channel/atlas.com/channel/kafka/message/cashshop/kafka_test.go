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
