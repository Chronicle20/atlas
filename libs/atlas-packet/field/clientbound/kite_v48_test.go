package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 kite (message-box) byte fixtures. Opcodes come from
// CMessageBoxPool::OnPacket @0x54329b, which routes 197 -> OnCreateFailed,
// 198 -> OnMessageBoxEnterField and 199 -> OnMessageBoxLeaveField (v61 uses
// 207/208/209). All three are shape-stable against v83; no codec change needed.

// TestKiteSpawnBytesV48 — CMessageBoxPool::OnMessageBoxEnterField @0x5432f7 reads
// Decode4(id), Decode4(templateId), DecodeStr(message), DecodeStr(name),
// Decode2(x), Decode2(y).
//
// packet-audit:verify packet=field/clientbound/FieldKiteSpawn version=gms_v48 ida=0x5432f7
func TestKiteSpawnBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewKiteSpawn(0x01020304, 5000, "hi", "bob", 300, 2).Encode(l, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{
		0x04, 0x03, 0x02, 0x01, // id         — Decode4
		0x88, 0x13, 0x00, 0x00, // templateId — Decode4
		0x02, 0x00, 'h', 'i', // message    — DecodeStr
		0x03, 0x00, 'b', 'o', 'b', // name       — DecodeStr
		0x2C, 0x01, // x = 300    — Decode2
		0x02, 0x00, // y          — Decode2
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 kite spawn:\n got % x\nwant % x", got, want)
	}
}

// TestKiteDestroyBytesV48 — CMessageBoxPool::OnMessageBoxLeaveField @0x5438de
// reads Decode1(animationType) then Decode4(id).
//
// packet-audit:verify packet=field/clientbound/FieldKiteDestroy version=gms_v48 ida=0x5438de
func TestKiteDestroyBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := NewKiteDestroy(0x01020304, 1).Encode(l, pt.CreateContext("GMS", 48, 1))(nil)
	want := []byte{0x01, 0x04, 0x03, 0x02, 0x01}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 kite destroy:\n got % x\nwant % x", got, want)
	}
}

// TestKiteErrorBytesV48 — CMessageBoxPool::OnCreateFailed @0x5432ce makes NO
// CInPacket call at all: it builds string 460 and hands it to CUtilDlg::Notice
// @0x5432f4. The body is empty, which is what the codec emits.
//
// packet-audit:verify packet=field/clientbound/FieldKiteError version=gms_v48 ida=0x5432ce
func TestKiteErrorBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	if got := NewKiteError().Encode(l, pt.CreateContext("GMS", 48, 1))(nil); len(got) != 0 {
		t.Errorf("v48 kite error: got % x, want empty", got)
	}
}
