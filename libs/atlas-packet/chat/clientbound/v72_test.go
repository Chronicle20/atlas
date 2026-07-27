package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 SERVERMESSAGE clientbound fixtures — CWvsContext::OnBroadcastMsg
// @0x91aaac (GMS_v72.1_U_DEVM.exe, port 13339), task-123 legacy phase 1.

// TestWorldMessageMegaphoneByteOutputV72 — mode==2 has no header-extras arm
// (the `cmp edi,3 -> jz SuperMegaphone; cmp edi,8/9 -> ...` chain @0x91ab41+
// skips mode 2 entirely). Wire: Decode1(mode) + DecodeStr(message).
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMegaphone version=gms_v72 ida=0x91aaac
func TestWorldMessageMegaphoneByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewWorldMessageMegaphone(2, "hi")
	want := []byte{0x02, 0x02, 0x00, 0x68, 0x69}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 megaphone: got % x want % x", got, want)
	}
}

// TestWorldMessageSuperMegaphoneByteOutputV72 — `cmp edi,3; jz loc_91AC46`
// @0x91ab41-0x91ab44 -> loc_91AC46: Decode1(channelId) + Decode1(whispersOn)
// @0x91ac48-0x91ac5a, AFTER the unconditional DecodeStr(message). Wire:
// mode(1) + message(str) + channelId(1) + whispersOn(1).
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSuperMegaphone version=gms_v72 ida=0x91aaac
func TestWorldMessageSuperMegaphoneByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := WorldMessageSuperMegaphone{mode: 3, message: "hi", channelId: 5, whispersOn: true}
	want := []byte{0x03, 0x02, 0x00, 0x68, 0x69, 0x05, 0x01}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 super megaphone: got % x want % x", got, want)
	}
}
