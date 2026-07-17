package clientbound

import (
	"bytes"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Wire (design §1 Q4, identical v83 0xA25268 / v95 0x9F9F90): byte mode, byte
// targetList (0=regular 1=VIP), then for list modes 5 or 10 x int mapId padded
// with EmptyMapId (999999999 = FF C9 9A 3B LE).
//
// packet-audit:verify packet=teleportrock/clientbound/MapTransferResult version=gms_v83 ida=0xA25268
// packet-audit:verify packet=teleportrock/clientbound/MapTransferResult version=gms_v95 ida=0x9F9F90
func TestMapTransferListRegularGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	m := NewMapTransferList(3, false, []_map.Id{100000000, 220000000})
	got := m.Encode(l, ctx)(nil)
	want := []byte{
		0x03,                   // mode = REGISTER_LIST
		0x00,                   // targetList = regular
		0x00, 0xE1, 0xF5, 0x05, // 100000000
		0x00, 0xEF, 0x1C, 0x0D, // 220000000
		0xFF, 0xC9, 0x9A, 0x3B, // EmptyMapId
		0xFF, 0xC9, 0x9A, 0x3B,
		0xFF, 0xC9, 0x9A, 0x3B,
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

func TestMapTransferListVipPadsToTen(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	m := NewMapTransferList(2, true, []_map.Id{100000000})
	got := m.Encode(l, ctx)(nil)
	if len(got) != 2+10*4 {
		t.Fatalf("VIP list body must be 42 bytes, got %d", len(got))
	}
	if got[0] != 0x02 || got[1] != 0x01 {
		t.Fatalf("header: % x", got[:2])
	}
	// slots 1..9 must be EmptyMapId
	for i := 0; i < 9; i++ {
		off := 2 + 4 + i*4
		if !bytes.Equal(got[off:off+4], []byte{0xFF, 0xC9, 0x9A, 0x3B}) {
			t.Fatalf("slot %d not padded: % x", i+1, got[off:off+4])
		}
	}
}

func TestMapTransferErrorGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	m := NewMapTransferError(5, false)
	got := m.Encode(l, ctx)(nil)
	want := []byte{0x05, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

func TestMapTransferResultCrossVersionStable(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := NewMapTransferList(3, false, []_map.Id{100000000})
	base := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			got := m.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
			if !bytes.Equal(got, base) {
				t.Errorf("%s differs from v83\n got: % x\nv83: % x", v.Name, got, base)
			}
		})
	}
}
