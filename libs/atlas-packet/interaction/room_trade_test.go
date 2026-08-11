package interaction

import (
	"encoding/hex"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestNewTradeRoomIsBaseFrameOnly pins design §1.3: CTradingRoomDlg's
// enter-result tail virtual (v83 vtable off_B37448 slot +72 -> nullsub_94
// @0x48314D) is EMPTY, so the trade room body is exactly the
// CMiniRoomBaseDlg::OnEnterResultBase frame — roomType, capacity(2), position,
// {slot, avatar, name} visitors, 0xFF — and nothing follows.
func TestNewTradeRoomIsBaseFrameOnly(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)

	rm := NewTradeRoom(TradeRoomType, 1, nil)
	if rm.RoomType() != TradeRoomType {
		t.Fatalf("roomType: got %v, want %v", rm.RoomType(), TradeRoomType)
	}
	if rm.Capacity() != 2 {
		t.Fatalf("capacity: got %d, want 2", rm.Capacity())
	}
	if rm.Position() != 1 {
		t.Fatalf("position: got %d, want 1", rm.Position())
	}

	got := hex.EncodeToString(rm.Encode(l, ctx)(nil))
	// roomType(03) capacity(02) position(01) terminator(FF), nothing after.
	if got != "030201ff" {
		t.Fatalf("empty trade room bytes: got %s, want 030201ff", got)
	}
}

func TestNewTradeRoomCashVariant(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)

	rm := NewTradeRoom(CashTradeRoomType, 0, nil)
	got := hex.EncodeToString(rm.Encode(l, ctx)(nil))
	if got != "060200ff" {
		t.Fatalf("empty cash trade room bytes: got %s, want 060200ff", got)
	}
}

// TestNewTradeRoomRoundTripsVisitors proves the visitor list survives the
// base-frame encode/decode via decodeVisitorForRoom's default arm
// (visitor.go:106), which already covers Trade and CashTrade. pt.RoundTrip
// only asserts the reader was fully consumed, so this test additionally
// decodes the room directly and asserts the visitor slot/name round-trip.
func TestNewTradeRoomRoundTripsVisitors(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)

	visitors := []Visitor{NewBaseVisitor(1, model.Avatar{}, "Partner")}
	in := NewTradeRoom(TradeRoomType, 1, visitors)

	out := Room{}
	pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)

	if len(out.Visitors()) != 1 {
		t.Fatalf("visitor count: got %d, want 1", len(out.Visitors()))
	}
	got := out.Visitors()[0]
	if got.Slot() != 1 {
		t.Fatalf("visitor slot: got %d, want 1", got.Slot())
	}
	if got.Name() != "Partner" {
		t.Fatalf("visitor name: got %q, want %q", got.Name(), "Partner")
	}
}
