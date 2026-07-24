package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 SERVERMESSAGE (clientbound op 0x41) simple-mode verification —
// CWvsContext::OnBroadcastMsg @0x844d49 (GMS_v61.1_U_DEVM.exe, port 13338).
// CWvsContext::OnPacket dispatches case 0x41 → OnBroadcastMsg.
//
// Read order at the top of OnBroadcastMsg (every byte traced to a v61 decompile
// line):
//
//	v3 = CInPacket::Decode1(a2)          @0x844d66 → mode byte.
//	if ( v3 != 4 || (Decode1 != 0) ) {   @0x844d9b (mode!=4 short-circuits true)
//	    CInPacket::DecodeStr(&v70)       @0x844da7 → message string.
//	    if ( v3 == 3 ) Decode1+Decode1   @0x844ddf/0x844dea (SuperMegaphone only)
//	    else if ( v3 == 8 ) Decode1×3[+item] @0x844e00.. (ItemMegaphone only)
//	}
//	switch ( v3 ) {  @0x844e50
//	    case 0 (Notice)   @0x844e60: builds UI from message, reads NO further wire.
//	    case 1 (PopUp)    @0x845015: Notice from message, NO further wire.
//	    case 2 (Megaphone)@0x845025: curse-process + chatlog, NO further wire.
//	    case 5 (PinkText) @0x8454ba: v53=12; sub_47010A, NO further wire.
//	    ...
//	}
//
// The v61 switch has exactly cases 0..10 (the broadcast dispatcher is arms 0-10
// in v61 — mode 7 @0x845507 additionally CInPacket::Decode4 @0x845546, modes
// 3/8 read the header extras above; no case 11+). The v61 seed template
// operations table (NOTICE..MULTI_MEGAPHONE = 0..10) matches this switch.
//
// For the SIMPLE modes (Notice=0, PopUp=1, Megaphone=2, PinkText=5) the entire
// wire is Decode1(mode) + DecodeStr(message) — byte-identical to the
// IDA-verified gms_v72 OnBroadcastMsg @0x91aaac (TestWorldMessageSimpleByteOutputV72).
// v61 is below every v72+ gate; the WorldMessageSimple codec is version-agnostic
// (WriteByte(mode) + WriteAsciiString(message)), so v61 mirrors v72 exactly.
// WriteAsciiString = uint16-LE length + ASCII bytes ("hi" = 02 00 68 69).
//
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSimple version=gms_v61 ida=0x844d49
func TestWorldMessageSimpleV61Body(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)

	// Notice mode (0): Decode1(mode) + DecodeStr("hi").
	// 0x00 | 0x02 0x00 'h' 'i'
	t.Run("notice", func(t *testing.T) {
		input := WorldMessageSimple{mode: 0, message: "hi"}
		want := []byte{0x00, 0x02, 0x00, 0x68, 0x69}
		got := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(got, want) {
			t.Errorf("v61 notice: got % x want % x", got, want)
		}
	})

	// PinkText mode (5): same wire shape — Decode1(mode) + DecodeStr("hi").
	// 0x05 | 0x02 0x00 'h' 'i'
	t.Run("pinktext", func(t *testing.T) {
		input := WorldMessageSimple{mode: 5, message: "hi"}
		want := []byte{0x05, 0x02, 0x00, 0x68, 0x69}
		got := pt.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(got, want) {
			t.Errorf("v61 pinktext: got % x want % x", got, want)
		}
	})
}

// TestWorldMessageMegaphoneByteOutputV61 — CWvsContext::OnBroadcastMsg
// @0x844d49: mode==2 has no header-extras arm (the `if v3==3 ... else if
// v3==8` chain @0x844dd1-0x844df2 skips mode 2 entirely), so the wire is
// exactly Decode1(mode) + DecodeStr(message), byte-identical to v72/v83/v95.
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMegaphone version=gms_v61 ida=0x844d49
func TestWorldMessageMegaphoneByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := NewWorldMessageMegaphone(2, "hi")
	want := []byte{0x02, 0x02, 0x00, 0x68, 0x69}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 megaphone: got % x want % x", got, want)
	}
}

// TestWorldMessageSuperMegaphoneByteOutputV61 — CWvsContext::OnBroadcastMsg
// @0x844d49, `if ( v3 == 3 ) { v73=Decode1(v2); v72=Decode1(v2); }`
// @0x844dd1-0x844dea (channelId then whispersOn), AFTER the unconditional
// DecodeStr(message). Wire: mode(1) + message(str) + channelId(1) +
// whispersOn(1).
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSuperMegaphone version=gms_v61 ida=0x844d49
func TestWorldMessageSuperMegaphoneByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := WorldMessageSuperMegaphone{mode: 3, message: "hi", channelId: 5, whispersOn: true}
	want := []byte{0x03, 0x02, 0x00, 0x68, 0x69, 0x05, 0x01}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 super megaphone: got % x want % x", got, want)
	}
}
