package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 login-family byte fixtures, all read off GMS_v48_1_DEVM.exe.
//
// CLogin::OnPacket @0x5007c4 is the family dispatcher; each handler below is the
// leaf it routes to. These four are shape-stable against v83 — no version gate
// crosses them — so the fixtures pin the v48 wire without any codec change.

// TestPinUpdateBytesV48 — CLogin::OnUpdatePinCodeResult @0x503c92 reads exactly
// one Decode1 @0x503c96 and branches on it (non-zero → CLoginUtilDlg::Error(15),
// zero → sub_518CD0(8)). No further CInPacket call: the body is a single byte.
//
// packet-audit:verify packet=login/clientbound/PinUpdate version=gms_v48 ida=0x503c92
func TestPinUpdateBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	for _, mode := range []byte{0x00, 0x01} {
		got := NewPinUpdate(mode).Encode(l, ctx)(nil)
		if !bytes.Equal(got, []byte{mode}) {
			t.Errorf("v48 PinUpdate(mode=%d): got % x, want % x", mode, got, []byte{mode})
		}
	}
}

// TestPinOperationBytesV48 — CLogin::OnCheckPinCodeResult @0x503956 reads exactly
// one Decode1 @0x50396b and switches on it (0, 1, 2, 3, 4, 7). Every branch from
// there builds a COutPacket and sends — outbound only, no second read. The
// clientbound body is a single mode byte.
//
// packet-audit:verify packet=login/clientbound/PinOperation version=gms_v48 ida=0x503956
func TestPinOperationBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	for _, mode := range []byte{0x00, 0x01, 0x02, 0x03, 0x04} {
		got := NewPinOperation(mode).Encode(l, ctx)(nil)
		if !bytes.Equal(got, []byte{mode}) {
			t.Errorf("v48 PinOperation(mode=%d): got % x, want % x", mode, got, []byte{mode})
		}
	}
}

// TestServerStatusBytesV48 — CLogin::OnCheckUserLimitResult @0x5011d6 reads TWO
// Decode1s (@0x5011eb and @0x5011ee) and passes both to
// CUIWorldSelect::UserLimitResult(v2, v3). Atlas models the field as a single
// uint16 and writes it with WriteShort, which is byte-identical for every status
// the server actually sends (0 normal, 1 highly populated, 2 full): the short's
// low byte lands in the client's first Decode1 and the zero high byte in the
// second. This test pins that equivalence explicitly so a future widening of the
// field cannot silently desync v48.
//
// packet-audit:verify packet=login/clientbound/ServerStatus version=gms_v48 ida=0x5011d6
func TestServerStatusBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	for _, status := range []uint16{0, 1, 2} {
		got := NewServerStatus(status).Encode(l, ctx)(nil)
		want := []byte{byte(status), 0x00} // Decode1 @0x5011eb, Decode1 @0x5011ee
		if !bytes.Equal(got, want) {
			t.Errorf("v48 ServerStatus(%d): got % x, want % x", status, got, want)
		}
	}
}

// TestServerListEndBytesV48 — the terminator arm of CLogin::OnWorldInformation
// @0x50120a: Decode1(worldId) @0x501225, and when that byte is negative the
// handler goes straight to CLogin::ChangeStep @0x5012b8 without reading anything
// else. The body is the single 0xFF sentinel.
//
// packet-audit:verify packet=login/clientbound/ServerListEnd version=gms_v48 ida=0x50120a
func TestServerListEndBytesV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	got := ServerListEnd{}.Encode(l, ctx)(nil)
	if !bytes.Equal(got, []byte{0xFF}) {
		t.Errorf("v48 ServerListEnd: got % x, want ff", got)
	}
}
