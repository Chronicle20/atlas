package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 MESSENGER family — clientbound dispatcher CUIMessenger::OnPacket sub_6D34F1
// @0x6d34f1 (GMS_v61.1_U_DEVM.exe, port 13338). Decode1(mode) then switch, mode
// table byte-identical to v72/v79/v83 (cases 0..8):
//
//	case 0 -> sub_6D144E @0x6d144e  OnEnter  -> MessengerAdd
//	         Decode1(position) + AvatarLook::Decode + DecodeStr(name)
//	         + Decode1(channelId) + Decode1(pad)  — SIX fields incl. mode.
//
// This CORRECTS the prior v61 fixture, which pointed at sub_5BF5AE @0x5bf5ae and
// asserted a 3-field "legacy" Add (position + avatar + name, no channelId/pad).
// sub_5BF5AE is NOT the messenger dispatcher's Add arm — it is a CMiniRoomBaseDlg
// arm (dispatched by sub_5BEC69 @0x5bec69, the mini-room OnPacketBase). The REAL
// v61 messenger Add (sub_6D144E, reached via CField::OnPacket case 243 ->
// sub_6D34F1 -> case 0) reads all six fields, identical to the IDA-verified v83
// Add (@0x8511fc). model.Avatar gates only on GMS<=28 vs >28, so the v61 (>28)
// avatar bytes equal v83. Therefore the v61 Add wire is byte-for-byte the verified
// v83 Add wire — cross-version equality.
//
// packet-audit:verify packet=messenger/clientbound/MessengerAdd version=gms_v61 ida=0x6d144e
func TestMessengerAddV61Body(t *testing.T) {
	v61 := pt.CreateContext("GMS", 61, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	// Deterministic single-equip avatar: v61 and v83 are both >=61 (same 3-int pet
	// path) and >28 (same look), so the avatar bytes are byte-equal; a single equip
	// entry avoids Go's randomized map iteration flaking the compare.
	ava := v79DetAvatar()

	got := NewMessengerAdd(0, 2, ava, "TestPlayer", 3).Encode(nil, v61)(nil)
	v83bytes := NewMessengerAdd(0, 2, ava, "TestPlayer", 3).Encode(nil, v83)(nil)

	// v61 == v83 (full six-field wire incl. channelId + pad).
	if !bytes.Equal(got, v83bytes) {
		t.Errorf("v61 MessengerAdd body mismatch\n got: % x\nwant: % x (== v83)", got, v83bytes)
	}
	// Guard the exact tail: channelId (3) then pad (0).
	if n := len(got); n < 2 || got[n-2] != 3 || got[n-1] != 0 {
		t.Errorf("v61 Add must end with channelId+pad (03 00); got tail % x", got[len(got)-2:])
	}

	// Round-trip: mode/position/name/channelId preserved.
	out := Add{}
	pt.RoundTrip(t, v61, NewMessengerAdd(0, 2, ava, "TestPlayer", 3).Encode, out.Decode, nil)
	if out.Mode() != 0 || out.Position() != 2 || out.Name() != "TestPlayer" || out.ChannelId() != 3 {
		t.Errorf("v61 Add round-trip: mode=%d position=%d name=%q channelId=%d",
			out.Mode(), out.Position(), out.Name(), out.ChannelId())
	}
}
