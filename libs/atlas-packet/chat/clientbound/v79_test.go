package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79 SERVERMESSAGE clientbound fixtures — CWvsContext::OnBroadcastMsg
// @0x96c94f (GMS_v79_1_DEVM.exe, port 13340), task-123 legacy phase 1.

// TestWorldMessageMegaphoneByteOutputV79 — mode==2 has no header-extras arm
// (the `cmp esi,3 -> jz SuperMegaphone; cmp esi,8 -> ...` chain @0x96c9e4+
// skips mode 2 entirely). Wire: Decode1(mode) + DecodeStr(message).
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageMegaphone version=gms_v79 ida=0x96c94f
func TestWorldMessageMegaphoneByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := NewWorldMessageMegaphone(2, "hi")
	want := []byte{0x02, 0x02, 0x00, 0x68, 0x69}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 megaphone: got % x want % x", got, want)
	}
}

// TestWorldMessageSuperMegaphoneByteOutputV79 — `cmp esi,3; jz loc_96CAE9`
// @0x96c9e4-0x96c9e7 -> loc_96CAE9: Decode1(channelId) + Decode1(whispersOn)
// @0x96caeb-0x96cafd, AFTER the unconditional DecodeStr(message). Wire:
// mode(1) + message(str) + channelId(1) + whispersOn(1).
// packet-audit:verify packet=chat/clientbound/ChatWorldMessageSuperMegaphone version=gms_v79 ida=0x96c94f
func TestWorldMessageSuperMegaphoneByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := WorldMessageSuperMegaphone{mode: 3, message: "hi", channelId: 5, whispersOn: true}
	want := []byte{0x03, 0x02, 0x00, 0x68, 0x69, 0x05, 0x01}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 super megaphone: got % x want % x", got, want)
	}
}
