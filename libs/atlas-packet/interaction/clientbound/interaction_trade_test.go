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

// tradeArmCase is one op × version cell: the version's tenant context plus the
// mode byte its CTradingRoomDlg::OnPacket switch dispatches this arm on. Every
// mode/address below was read from that version's own IDB by decompiling
// CTradingRoomDlg::OnPacket and following its switch refs — see
// docs/tasks/task-205-player-trade/version-matrix.md §1.
type tradeArmCase struct {
	name   string
	region string
	major  uint16
	mode   byte
}

// TestInteractionTradePutItemByteOutput pins the mode-15 arm per version.
//
// Derived read order (identical on all ten; addresses in the markers):
//
//	Decode1 mode        the OnPacket dispatch byte
//	Decode1 side        recipient-relative room side, indexes this[side + N]
//	Decode1 tradeSlot   1..9 within that side's grid
//	GW_ItemSlotBase::Decode   the staged item
//
// OPAQUE-LEDGER EXCEPTION (VERIFYING_A_PACKET §5): the trailing bytes are the
// shared model.Asset codec, which the client reads inside GW_ItemSlotBase::Decode
// — the audit reports record that row as "opaque type: model.Asset — register
// boundary", so those bytes cannot cite a per-field decompile line here. They are
// derived from the Atlas encoder plus model.Asset's own tier-1 verification. The
// three leading bytes DO each cite a decompile line and are the fields this arm
// owns; a reorder of them fails independently of asset-codec churn.
//
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradePutItem version=gms_v48 ida=0x5e6ace
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradePutItem version=gms_v61 ida=0x68b37f
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradePutItem version=gms_v72 ida=0x6fdce7
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradePutItem version=gms_v79 ida=0x7357bf
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradePutItem version=gms_v83 ida=0x7c1fb7
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradePutItem version=gms_v84 ida=0x7e80fd
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradePutItem version=gms_v87 ida=0x81566e
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradePutItem version=gms_v92 ida=0x743f80
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradePutItem version=gms_v95 ida=0x763a60
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradePutItem version=jms_v185 ida=0x845dd0
func TestInteractionTradePutItemByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// side(01) tradeSlot(03) then the model.Asset blob for a stackable
	// templateId 2000000, quantity 50.
	const tail = "01030280841e0000ffffffffffffffff320000000000"
	for _, c := range []tradeArmCase{
		{"gms_v48", "GMS", 48, 13},
		{"gms_v61", "GMS", 61, 14},
		{"gms_v72", "GMS", 72, 14},
		{"gms_v79", "GMS", 79, 15},
		{"gms_v83", "GMS", 83, 15},
		{"gms_v84", "GMS", 84, 15},
		{"gms_v87", "GMS", 87, 15},
		{"gms_v92", "GMS", 92, 15},
		{"gms_v95", "GMS", 95, 15},
		{"jms_v185", "JMS", 185, 13},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(50, 0, 0)
			raw := NewInteractionTradePutItem(c.mode, 1, 3, a).Encode(l, pt.CreateContext(c.region, c.major, 1))(nil)
			want := hex.EncodeToString([]byte{c.mode}) + tail
			if got := hex.EncodeToString(raw); got != want {
				t.Errorf("%s bytes: got %s, want %s", c.name, got, want)
			}
		})
	}
}

// TestInteractionTradeAddMesoByteOutput pins the mode-16 arm per version.
//
// Derived read order (identical on all ten):
//
//	Decode1 mode
//	Decode1 side
//	Decode4 amount   ASSIGNED (this[side + N] = Decode4()), never accumulated
//
// The assignment semantics are why an authoritative re-echo of the last valid
// amount is how the server corrects an out-of-range stage (design §4.2), and
// they are the whole meso-rejection path on jms_v185, which has no
// TRADE_MESO_LIMIT arm.
//
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeAddMeso version=gms_v48 ida=0x5e6ba5
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeAddMeso version=gms_v61 ida=0x68b456
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeAddMeso version=gms_v72 ida=0x6fddbe
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeAddMeso version=gms_v79 ida=0x735896
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeAddMeso version=gms_v83 ida=0x7c208e
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeAddMeso version=gms_v84 ida=0x7e81d4
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeAddMeso version=gms_v87 ida=0x815745
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeAddMeso version=gms_v92 ida=0x743e70
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeAddMeso version=gms_v95 ida=0x763950
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeAddMeso version=jms_v185 ida=0x845ea7
func TestInteractionTradeAddMesoByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// side(01) amount(1000000 = 0x000F4240, little-endian 40420f00)
	const tail = "0140420f00"
	for _, c := range []tradeArmCase{
		{"gms_v48", "GMS", 48, 14},
		{"gms_v61", "GMS", 61, 15},
		{"gms_v72", "GMS", 72, 15},
		{"gms_v79", "GMS", 79, 16},
		{"gms_v83", "GMS", 83, 16},
		{"gms_v84", "GMS", 84, 16},
		{"gms_v87", "GMS", 87, 16},
		{"gms_v92", "GMS", 92, 16},
		{"gms_v95", "GMS", 95, 16},
		{"jms_v185", "JMS", 185, 14},
	} {
		t.Run(c.name, func(t *testing.T) {
			raw := NewInteractionTradeAddMeso(c.mode, 1, 1000000).Encode(l, pt.CreateContext(c.region, c.major, 1))(nil)
			want := hex.EncodeToString([]byte{c.mode}) + tail
			if got := hex.EncodeToString(raw); got != want {
				t.Errorf("%s bytes: got %s, want %s", c.name, got, want)
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

// TestInteractionTradeConfirmByteOutput pins the confirm arm per version.
//
// Derived read order (identical on all ten): Decode1 mode, and NOTHING else —
// the arm reads no body. It sets the local confirm flag, redraws, and on
// gms_v83+ / jms_v185 immediately auto-sends the serverbound CRC attestation
// (TRANSACTION). A stray trailing byte here would be read as the next packet.
//
// The same function is the TRANSACTION sender, which is why the export keys the
// bodyless RECEIVE shape as CTradingRoomDlg::OnTrade#TradeConfirm and the SEND
// shape as the un-suffixed CTradingRoomDlg::OnTrade.
//
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeConfirm version=gms_v48 ida=0x5e6bd3
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeConfirm version=gms_v61 ida=0x68b484
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeConfirm version=gms_v72 ida=0x6fddec
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeConfirm version=gms_v79 ida=0x7358c4
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeConfirm version=gms_v83 ida=0x7c20bc
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeConfirm version=gms_v84 ida=0x7e8202
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeConfirm version=gms_v87 ida=0x815773
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeConfirm version=gms_v92 ida=0x744440
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeConfirm version=gms_v95 ida=0x763f20
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeConfirm version=jms_v185 ida=0x845ed5
func TestInteractionTradeConfirmByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, c := range []tradeArmCase{
		{"gms_v48", "GMS", 48, 15},
		{"gms_v61", "GMS", 61, 16},
		{"gms_v72", "GMS", 72, 16},
		{"gms_v79", "GMS", 79, 17},
		{"gms_v83", "GMS", 83, 17},
		{"gms_v84", "GMS", 84, 17},
		{"gms_v87", "GMS", 87, 17},
		{"gms_v92", "GMS", 92, 17},
		{"gms_v95", "GMS", 95, 17},
		{"jms_v185", "JMS", 185, 15},
	} {
		t.Run(c.name, func(t *testing.T) {
			raw := NewInteractionTradeConfirm(c.mode).Encode(l, pt.CreateContext(c.region, c.major, 1))(nil)
			want := hex.EncodeToString([]byte{c.mode})
			if got := hex.EncodeToString(raw); got != want {
				t.Errorf("%s bytes: got %s, want %s", c.name, got, want)
			}
		})
	}
}

// TestInteractionTradeMesoLimitByteOutput pins the meso-limit arm per version.
//
// Derived read order: Decode1 mode, and NOTHING else. The arm shows the
// daily-meso-limit notice, clears the local confirm flag and re-enables both
// confirm buttons. Its StringPool id is per-version and not stable
// (v48 3626 · v61 3901 · v72 3953 · v79 3956 · v83 SP_3977 · v84 3980 ·
// v87 3985 · v92 4051 · v95 4018), which is what identifies it in the
// symbol-stripped builds.
//
// jms_v185 is deliberately ABSENT from this table: its CTradingRoomDlg::OnPacket
// @0x845d95 is a complete switch with exactly three cases (13/14/15) and no
// default, so the arm does not exist there. Recorded as an n-a disposition in
// docs/packets/audits/jms_v185/_unimplemented.json with that enumeration, not
// inferred from a failed name search. No cash trading room on ANY version has
// the arm either (every CCashTradingRoomDlg::OnPacket switch has three cases).
//
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeMesoLimit version=gms_v48 ida=0x5e6be7
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeMesoLimit version=gms_v61 ida=0x68b498
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeMesoLimit version=gms_v72 ida=0x6fde00
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeMesoLimit version=gms_v79 ida=0x7358d8
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeMesoLimit version=gms_v83 ida=0x7c21bd
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeMesoLimit version=gms_v84 ida=0x7e8303
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeMesoLimit version=gms_v87 ida=0x815877
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeMesoLimit version=gms_v92 ida=0x744680
// packet-audit:verify packet=interaction/clientbound/InteractionInteractionTradeMesoLimit version=gms_v95 ida=0x764160
func TestInteractionTradeMesoLimitByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, c := range []tradeArmCase{
		{"gms_v48", "GMS", 48, 18},
		{"gms_v61", "GMS", 61, 19},
		{"gms_v72", "GMS", 72, 19},
		{"gms_v79", "GMS", 79, 20},
		{"gms_v83", "GMS", 83, 21},
		{"gms_v84", "GMS", 84, 21},
		{"gms_v87", "GMS", 87, 21},
		{"gms_v92", "GMS", 92, 21},
		{"gms_v95", "GMS", 95, 21},
	} {
		t.Run(c.name, func(t *testing.T) {
			raw := NewInteractionTradeMesoLimit(c.mode).Encode(l, pt.CreateContext(c.region, c.major, 1))(nil)
			want := hex.EncodeToString([]byte{c.mode})
			if got := hex.EncodeToString(raw); got != want {
				t.Errorf("%s bytes: got %s, want %s", c.name, got, want)
			}
		})
	}
}

// TestInteractionTradeConfirmRoundTrip proves Decode drains cleanly and
// consumes no extra bytes for the mode-17 bodyless arm — the byte pin above
// only shows what Encode produces, not that Decode reads exactly that much.
func TestInteractionTradeConfirmRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewInteractionTradeConfirm(17)
			output := InteractionTradeConfirm{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// TestInteractionTradeMesoLimitRoundTrip proves Decode drains cleanly and
// consumes no extra bytes for the mode-21 bodyless arm — the byte pin above
// only shows what Encode produces, not that Decode reads exactly that much.
func TestInteractionTradeMesoLimitRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewInteractionTradeMesoLimit(21)
			output := InteractionTradeMesoLimit{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// TestTradeLeaveReasonKeysAreDistinct pins design §4.3: the trade leave path
// gets its OWN leaveReason keys and never borrows the shop or mini-game keys'
// numeric values (DOM-25 / the precedent at interaction_body.go:167-179).
func TestTradeLeaveReasonKeysAreDistinct(t *testing.T) {
	tradeKeys := []string{
		CharacterInteractionLeaveReasonTradeCancelled,
		CharacterInteractionLeaveReasonTradeSuccess,
		CharacterInteractionLeaveReasonTradeFailed,
		CharacterInteractionLeaveReasonTradeCannotCarry,
		CharacterInteractionLeaveReasonTradeDifferentMap,
		CharacterInteractionLeaveReasonTradeCrcFailed,
	}
	otherKeys := []string{
		CharacterInteractionLeaveReasonShopClosed,
		CharacterInteractionLeaveReasonUserBanned,
		CharacterInteractionLeaveReasonOutOfStock,
		CharacterInteractionLeaveReasonMiniGameClosed,
		CharacterInteractionLeaveReasonMiniGameLeft,
		CharacterInteractionLeaveReasonMiniGameExpelled,
	}

	seen := make(map[string]bool)
	for _, k := range tradeKeys {
		if k == "" {
			t.Fatal("empty trade leave reason key")
		}
		if seen[k] {
			t.Errorf("duplicate trade leave reason key %q", k)
		}
		seen[k] = true
	}
	for _, k := range otherKeys {
		if seen[k] {
			t.Errorf("trade leave reason key %q collides with a non-trade family key", k)
		}
	}
}
