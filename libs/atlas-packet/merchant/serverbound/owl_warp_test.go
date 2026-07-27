package serverbound

import (
	"encoding/binary"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=merchant/serverbound/OwlWarp version=gms_v83 ida=0x8a4423
// packet-audit:verify packet=merchant/serverbound/OwlWarp version=gms_v95 ida=0x848e80
// packet-audit:verify packet=merchant/serverbound/OwlWarp version=gms_v79 ida=0x80ce8a
// packet-audit:verify packet=merchant/serverbound/OwlWarp version=gms_v72 ida=0x7c58df
// packet-audit:verify packet=merchant/serverbound/OwlWarp version=gms_v61 ida=0x718c84
func TestOwlWarpRoundTrip(t *testing.T) {
	input := NewOwlWarp(30001, 910000005)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := &OwlWarp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.OwnerId() != 30001 {
				t.Errorf("ownerId = %d, want 30001", output.OwnerId())
			}
			if output.MapId() != 910000005 {
				t.Errorf("mapId = %d, want 910000005", output.MapId())
			}
		})
	}
}

// TestOwlWarpWireShape pins [int dwMiniRoomSN][int dwFieldID] — the client
// echoes both ints from the clicked record verbatim (v83 sub_8A4423, v95
// CUIShopScanResult::OnButtonClicked 0x848e80).
func TestOwlWarpWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewOwlWarp(30001, 910000005)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if len(b) != 8 {
				t.Fatalf("wire size = %d bytes, want 8: % x", len(b), b)
			}
			if binary.LittleEndian.Uint32(b[0:4]) != 30001 {
				t.Errorf("dwMiniRoomSN = %d, want 30001", binary.LittleEndian.Uint32(b[0:4]))
			}
			if binary.LittleEndian.Uint32(b[4:8]) != 910000005 {
				t.Errorf("dwFieldID = %d, want 910000005", binary.LittleEndian.Uint32(b[4:8]))
			}
		})
	}
}
