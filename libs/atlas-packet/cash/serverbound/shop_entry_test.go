package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CWvsContext::SendMigrateToShopRequest@0x9dc280:
//
// After the guest-account, migrating, throttle (500ms via m_bExclRequestSent
// / m_tExclRequestSent), anti-macro, and initial-quiz guards all pass (and
// the field option flag 0x10 is clear), the send block does:
//
//	v16 = 43;                                    /*0x9dc49e opcode = ENTER_CASHSHOP*/
//	COutPacket::COutPacket(&oPacket, v16);        /*0x9dc4aa*/
//	update_time = get_update_time();              /*0x9dc4b7*/
//	COutPacket::Encode4(&oPacket, update_time);    /*0x9dc4c1*/
//	CClientSocket::SendPacket(..., &oPacket);      /*0x9dc4d1*/
//
// This is NOT the bodiless ENTER_MTS shape (CWvsContext::SendMigrateToITCRequest,
// per tools/packet-audit/cmd/run.go) — ENTER_CASHSHOP carries a single
// leading Encode4(update_time) field, matching atlas
// cash/serverbound/shop_entry.go ShopEntry.Encode (updateTime uint32,
// WriteInt) exactly. No wire fix needed.
//
// packet-audit:verify packet=cash/serverbound/CashShopEntry version=gms_v95 ida=0x9dc280
func TestShopEntryByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := ShopEntry{updateTime: 0x11223344}
	expected := []byte{0x44, 0x33, 0x22, 0x11}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 shop entry golden mismatch: got %v want %v", actual, expected)
	}
	if len(actual) != 4 {
		t.Errorf("v95 shop entry golden length: got %d want 4", len(actual))
	}
}

func TestShopEntryRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopEntry{updateTime: 100}
			output := ShopEntry{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
