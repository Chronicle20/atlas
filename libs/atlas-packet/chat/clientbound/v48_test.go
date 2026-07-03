package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 SERVERMESSAGE (clientbound op 55 / 0x37) simple-mode verification —
// CWvsContext::OnBroadcastMsg @0x71c356 (GMS_v48_1_DEVM.exe, port 13337).
//
// Read order at the top of OnBroadcastMsg (every byte traced to a v48 decompile
// line):
//
//	v3 = CInPacket::Decode1(a2)          @0x71c372 → mode byte.
//	if ( v3 != 4 || (Decode1 != 0) ) {   @0x71c39d (mode!=4 short-circuits true)
//	    CInPacket::DecodeStr(&v58)       @0x71c3a6 → message string.
//	    if ( v3 == 3 ) Decode1+Decode1   @0x71c3df/0x71c3ea (SuperMegaphone only)
//	}
//	switch ( v3 ) {  @0x71c3f5
//	    case 0 (Notice)   @0x71c3fd: builds UI/ChatLog from message, reads NO further wire.
//	    case 1 (PopUp)    @0x71c5a1: Notice from message, NO further wire.
//	    case 2 (Megaphone)@0x71c5b1: chatlog, NO further wire.
//	    case 5 (PinkText) @0x71c9c0: ChatLogAdd(11), NO further wire.
//	    case 7 (Blue/NPC) @0x71ca17: additionally CInPacket::Decode4 @0x71ca56.
//	    ...
//	}
//
// The v48 switch has exactly cases 0..9 (broadcast dispatcher is arms 0-9 in v48
// — NARROWER than v61's 0-10, no item-megaphone arm). The v48 seed template
// operations table (NOTICE..NPC = 0..7) matches these simple/blue-text arms.
//
// For the SIMPLE modes (Notice=0, PopUp=1, Megaphone=2, PinkText=5) the entire
// wire is Decode1(mode) + DecodeStr(message) — byte-identical to the IDA-verified
// gms_v61 OnBroadcastMsg @0x844d49 (TestWorldMessageSimpleV61Body). v48 is below
// every v61+ gate; the WorldMessageSimple codec is version-agnostic
// (WriteByte(mode) + WriteAsciiString(message)), so v48 mirrors v61 exactly.
// WriteAsciiString = uint16-LE length + ASCII bytes ("hi" = 02 00 68 69).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSimple version=gms_v48 ida=0x71c356
func TestWorldMessageSimpleV48Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)

	// Notice mode (0): Decode1(mode) + DecodeStr("hi").
	// 0x00 | 0x02 0x00 'h' 'i'
	t.Run("notice", func(t *testing.T) {
		input := WorldMessageSimple{mode: 0, message: "hi"}
		want := []byte{0x00, 0x02, 0x00, 0x68, 0x69}
		got := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(got, want) {
			t.Errorf("v48 notice: got % x want % x", got, want)
		}
	})

	// PinkText mode (5): same wire shape — Decode1(mode) + DecodeStr("hi").
	// 0x05 | 0x02 0x00 'h' 'i'
	t.Run("pinktext", func(t *testing.T) {
		input := WorldMessageSimple{mode: 5, message: "hi"}
		want := []byte{0x05, 0x02, 0x00, 0x68, 0x69}
		got := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(got, want) {
			t.Errorf("v48 pinktext: got % x want % x", got, want)
		}
	})
}
