package testsupport

import (
	"encoding/json"
	"testing"

	mtsmsg "atlas-mts/kafka/message/mts"

	"github.com/google/uuid"

	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

// TestBuyCommandProviderShape pins the simulated buy command to the exact
// envelope the channel emits and the mts consumer decodes: same Command struct,
// CommandBuy type tag, and buyer-keyed partition key.
func TestBuyCommandProviderShape(t *testing.T) {
	txn := uuid.New()
	msgs, err := BuyCommandProvider(txn, 0, 42, 2001, 3001, true)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if want := string(kprod.CreateKey(int(uint32(2001)))); string(msgs[0].Key) != want {
		t.Fatalf("expected buyer-derived key %q, got %q", want, string(msgs[0].Key))
	}
	var c mtsmsg.Command[mtsmsg.BuyCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("decode as consumer would: %v", err)
	}
	if c.TransactionId != txn || c.Type != mtsmsg.CommandBuy {
		t.Fatalf("bad envelope: %+v", c)
	}
	if c.Body.WorldId != 0 || c.Body.Serial != 42 || c.Body.BuyerId != 2001 || c.Body.BuyerAccountId != 3001 || !c.Body.BuyNow {
		t.Fatalf("bad body: %+v", c.Body)
	}
}

// TestPlaceBidCommandProviderShape pins the simulated bid command likewise.
func TestPlaceBidCommandProviderShape(t *testing.T) {
	txn := uuid.New()
	msgs, err := PlaceBidCommandProvider(txn, 0, 42, 2001, 3001, 7500)()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if want := string(kprod.CreateKey(int(uint32(2001)))); string(msgs[0].Key) != want {
		t.Fatalf("expected bidder-derived key %q, got %q", want, string(msgs[0].Key))
	}
	var c mtsmsg.Command[mtsmsg.PlaceBidCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &c); err != nil {
		t.Fatalf("decode as consumer would: %v", err)
	}
	if c.TransactionId != txn || c.Type != mtsmsg.CommandPlaceBid {
		t.Fatalf("bad envelope: %+v", c)
	}
	if c.Body.WorldId != 0 || c.Body.Serial != 42 || c.Body.BidderId != 2001 || c.Body.BidderAccountId != 3001 || c.Body.Amount != 7500 {
		t.Fatalf("bad body: %+v", c.Body)
	}
}
