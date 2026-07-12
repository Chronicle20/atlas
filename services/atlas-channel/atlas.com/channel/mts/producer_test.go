package mts

import (
	mtsmsg "atlas-channel/kafka/message/mts"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// TestBuyCommandProvider_FixedBuy asserts the BUY command body carries the wire
// serial (not a UUID), the buyer identity from the session, and BuyNow=false for a
// plain fixed-price buy.
func TestBuyCommandProvider_FixedBuy(t *testing.T) {
	tx := uuid.New()
	prov := BuyCommandProvider(tx, 1, 4242, 8001, 9001, false, mtsmsg.ResultKindItem)
	msgs, err := prov()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	var cmd mtsmsg.Command[mtsmsg.BuyCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != mtsmsg.CommandBuy {
		t.Errorf("type: want %s got %s", mtsmsg.CommandBuy, cmd.Type)
	}
	if cmd.TransactionId != tx {
		t.Errorf("transactionId: want %s got %s", tx, cmd.TransactionId)
	}
	if cmd.Body.Serial != 4242 {
		t.Errorf("serial: want 4242 got %d", cmd.Body.Serial)
	}
	if cmd.Body.WorldId != 1 {
		t.Errorf("worldId: want 1 got %d", cmd.Body.WorldId)
	}
	if cmd.Body.BuyerId != 8001 {
		t.Errorf("buyerId: want 8001 got %d", cmd.Body.BuyerId)
	}
	if cmd.Body.BuyerAccountId != 9001 {
		t.Errorf("buyerAccountId: want 9001 got %d", cmd.Body.BuyerAccountId)
	}
	if cmd.Body.BuyNow {
		t.Errorf("buyNow: want false for a fixed-price buy, got true")
	}
	if cmd.Body.ResultKind != mtsmsg.ResultKindItem {
		t.Errorf("resultKind: want %s got %s", mtsmsg.ResultKindItem, cmd.Body.ResultKind)
	}
}

// TestBuyCommandProvider_BuyNow asserts BuyNow=true for the BUY_AUCTION_IMM arm.
func TestBuyCommandProvider_BuyNow(t *testing.T) {
	prov := BuyCommandProvider(uuid.New(), 1, 4243, 8001, 9001, true, mtsmsg.ResultKindItem)
	msgs, err := prov()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var cmd mtsmsg.Command[mtsmsg.BuyCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !cmd.Body.BuyNow {
		t.Errorf("buyNow: want true for BUY_AUCTION_IMM, got false")
	}
	if cmd.Body.Serial != 4243 {
		t.Errorf("serial: want 4243 got %d", cmd.Body.Serial)
	}
}

// TestPlaceBidCommandProvider asserts the PLACE_BID command body carries the wire
// serial, the bidder identity from the session, and the raw bid amount.
func TestPlaceBidCommandProvider(t *testing.T) {
	prov := PlaceBidCommandProvider(uuid.New(), 1, 4244, 8002, 9002, 1500)
	msgs, err := prov()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	var cmd mtsmsg.Command[mtsmsg.PlaceBidCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Type != mtsmsg.CommandPlaceBid {
		t.Errorf("type: want %s got %s", mtsmsg.CommandPlaceBid, cmd.Type)
	}
	if cmd.Body.Serial != 4244 {
		t.Errorf("serial: want 4244 got %d", cmd.Body.Serial)
	}
	if cmd.Body.BidderId != 8002 {
		t.Errorf("bidderId: want 8002 got %d", cmd.Body.BidderId)
	}
	if cmd.Body.BidderAccountId != 9002 {
		t.Errorf("bidderAccountId: want 9002 got %d", cmd.Body.BidderAccountId)
	}
	if cmd.Body.Amount != 1500 {
		t.Errorf("amount: want 1500 got %d", cmd.Body.Amount)
	}
}
