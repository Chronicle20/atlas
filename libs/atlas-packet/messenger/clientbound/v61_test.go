package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 MESSENGER family — clientbound demuxer sub_5BEC69 @0x5bec69
// (GMS_v61.1_U_DEVM.exe, port 13338). Decode1(mode) then dispatch; the v61 mode
// table was renumbered vs v72 (see template_gms_61_1.json MessengerOperation):
//
//	ADD=4  sub_5BF5AE @0x5bf5ae  Decode1(position) + avatar(sub_5BFB65) + DecodeStr(name)
//	                             — NO channelId, NO trailing pad byte.
//
// v72+ OnEnter (MessengerAdd) reads mode+position+avatar+name+channelId+pad; the
// channelId+pad were added at GMS>=72. add.go now gates them off for the legacy
// range (GMS <72) — see legacyAdd(). model.Avatar gates only on GMS<=28 vs >28,
// so the v61 (>28) avatar bytes are identical to the IDA-verified v83 (>28)
// avatar (MessengerAdd v83 @0x8511fc). Therefore the v61 Add wire is byte-for-byte
// the verified v83 Add wire MINUS the trailing 2 bytes (channelId + pad) — the
// cross-version-equality discipline (door/party v61).
//
// packet-audit:verify packet=messenger/clientbound/MessengerAdd version=gms_v61 ida=0x5bf5ae
func TestMessengerAddV61Body(t *testing.T) {
	v61 := pt.CreateContext("GMS", 61, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	ava := testAvatar()

	got := NewMessengerAdd(4, 2, ava, "TestPlayer", 3).Encode(nil, v61)(nil)
	v83bytes := NewMessengerAdd(4, 2, ava, "TestPlayer", 3).Encode(nil, v83)(nil)

	// v61 == v83 with the trailing channelId (1 byte) + pad (1 byte) removed.
	if len(v83bytes) < 2 {
		t.Fatalf("v83 encode too short: % x", v83bytes)
	}
	want := v83bytes[:len(v83bytes)-2]
	if !bytes.Equal(got, want) {
		t.Errorf("v61 MessengerAdd body mismatch\n got: % x\nwant: % x (v83 minus channelId+pad)", got, want)
	}
	// Guard the exact tail: v61 ends with the name string, not channelId/pad.
	if got[len(got)-1] != 'r' { // last byte of "TestPlayer"
		t.Errorf("v61 Add must end with the name (no channelId/pad); got tail % x", got[len(got)-4:])
	}

	// Round-trip: mode/position/name preserved; channelId is not on the wire (stays 0).
	out := Add{}
	pt.RoundTrip(t, v61, NewMessengerAdd(4, 2, ava, "TestPlayer", 3).Encode, out.Decode, nil)
	if out.Mode() != 4 || out.Position() != 2 || out.Name() != "TestPlayer" {
		t.Errorf("v61 Add round-trip: mode=%d position=%d name=%q", out.Mode(), out.Position(), out.Name())
	}
	if out.ChannelId() != 0 {
		t.Errorf("v61 Add channelId must be absent from the wire (0); got %d", out.ChannelId())
	}
}
