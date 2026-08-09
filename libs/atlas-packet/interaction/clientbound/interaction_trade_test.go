package clientbound

import (
	"encoding/hex"
	"testing"
	"time"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestInteractionTradePutItemRoundTrip pins the mode-15 arm: Decode1 side,
// Decode1 trade slot, then GW_ItemSlotBase (v83 sub_7C1FB7 @0x7c1fb7). It
// encodes, decodes the raw bytes back into a fresh InteractionTradePutItem,
// and asserts the decoded side/tradeSlot/asset fields equal the input and
// that the reader is fully drained afterward.
func TestInteractionTradePutItemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			a := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(50, 0, 0)
			input := NewInteractionTradePutItem(15, 1, 3, a)

			l, _ := testlog.NewNullLogger()
			raw := input.Encode(l, ctx)(nil)
			if len(raw) < 3 {
				t.Fatalf("body too short: %d bytes", len(raw))
			}
			if raw[0] != 15 {
				t.Errorf("mode: got %d, want 15", raw[0])
			}
			if raw[1] != 1 {
				t.Errorf("side: got %d, want 1", raw[1])
			}
			if raw[2] != 3 {
				t.Errorf("tradeSlot: got %d, want 3", raw[2])
			}

			req := request.Request(raw)
			reader := request.NewRequestReader(&req, 0)
			var out InteractionTradePutItem
			out.Decode(l, ctx)(&reader, nil)

			if out.Mode() != input.Mode() {
				t.Errorf("decoded mode = %d, want %d", out.Mode(), input.Mode())
			}
			if out.Side() != input.Side() {
				t.Errorf("decoded side = %d, want %d", out.Side(), input.Side())
			}
			if out.TradeSlot() != input.TradeSlot() {
				t.Errorf("decoded tradeSlot = %d, want %d", out.TradeSlot(), input.TradeSlot())
			}
			if out.Asset().TemplateId() != input.Asset().TemplateId() {
				t.Errorf("decoded asset templateId = %d, want %d", out.Asset().TemplateId(), input.Asset().TemplateId())
			}
			if out.Asset().Quantity() != input.Asset().Quantity() {
				t.Errorf("decoded asset quantity = %d, want %d", out.Asset().Quantity(), input.Asset().Quantity())
			}
			if got := reader.GetRestAsBytes(); len(got) != 0 {
				t.Errorf("reader not drained: %d bytes remaining", len(got))
			}
		})
	}
}

// TestInteractionTradePutItemHeaderBytes pins the fixed three-byte header ahead
// of the asset blob so a field reorder is caught independently of asset codec
// churn.
func TestInteractionTradePutItemHeaderBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	a := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(50, 0, 0)
	raw := NewInteractionTradePutItem(15, 0, 1, a).Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if got := hex.EncodeToString(raw[:3]); got != "0f0001" {
		t.Errorf("header bytes: got %s, want 0f0001", got)
	}
}

// TestInteractionTradeAddMesoBytes pins the mode-16 arm: Decode1 side, Decode4
// amount (v83 sub_7C208E @0x7c208e). The client ASSIGNS the amount
// (this[v3+115] = Decode4), so an authoritative re-echo of the last valid
// amount is how the server corrects an out-of-range stage (design §4.2).
func TestInteractionTradeAddMesoBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			raw := NewInteractionTradeAddMeso(16, 1, 1000000).Encode(l, ctx)(nil)
			if got := hex.EncodeToString(raw); got != "100140420f00" {
				t.Errorf("bytes: got %s, want 100140420f00", got)
			}
		})
	}
}

func TestInteractionTradeAddMesoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewInteractionTradeAddMeso(16, 0, 987654321)
			output := InteractionTradeAddMeso{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Side() != input.Side() {
				t.Errorf("side: got %v, want %v", output.Side(), input.Side())
			}
			if output.Amount() != input.Amount() {
				t.Errorf("amount: got %v, want %v", output.Amount(), input.Amount())
			}
		})
	}
}

// TestInteractionTradeConfirmIsBodyless pins design §1.2: mode 17
// (CTradingRoomDlg::OnTrade @0x7c20bc) reads NO body — it sets this[112]=1,
// redraws, and immediately auto-sends serverbound 0x14 with the client's own
// CRC list. A stray trailing byte here would be read as the next packet.
func TestInteractionTradeConfirmIsBodyless(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			if got := hex.EncodeToString(NewInteractionTradeConfirm(17).Encode(l, ctx)(nil)); got != "11" {
				t.Errorf("bytes: got %s, want 11", got)
			}
		})
	}
}

// TestInteractionTradeMesoLimitIsBodyless pins design §1.2/§11.2: mode 21
// (sub_7C21BD @0x7c21bd) reads no body; it shows SP_3977 ("Players that are
// level 15 and below may only trade 1 million mesos per day"), clears
// this[111] and re-enables both confirm buttons. CCashTradingRoomDlg::OnPacket
// @0x4833b4 has NO mode-21 arm.
func TestInteractionTradeMesoLimitIsBodyless(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			if got := hex.EncodeToString(NewInteractionTradeMesoLimit(21).Encode(l, ctx)(nil)); got != "15" {
				t.Errorf("bytes: got %s, want 15", got)
			}
		})
	}
}
