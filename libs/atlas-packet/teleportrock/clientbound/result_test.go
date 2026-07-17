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
// IDA (live MapleStory_dump.exe v83, port 13342, task-124 verify pass):
// CWvsContext::OnMapTransferResult @0xa25268 — mode(Decode1)+targetList(Decode1),
// then for mode in {2,3} exactly 5 (targetList==0) or 10 (targetList==1) x
// Decode4 mapId. The registry op MAP_TRANSFER_RESULT keys to this fname; the
// packet id is the bare struct name (no pkg qualifier — MapTransferList is
// globally unique in libs/atlas-packet), matching candidatesFromFName's
// unqualified entry for this fname (cmd/run.go).
//
// packet-audit:verify packet=teleportrock/clientbound/MapTransferList version=gms_v83 ida=0xa25268
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

// task-124 v95 verify pass (live GMS_v95.0_U_DEVM.exe, port 13341):
// CWvsContext::OnMapTransferResult @0x9f9f90 — byte-identical read order to
// v83: mode(Decode1)+targetList(Decode1) @0x9f9fca/0x9f9fcd, then for mode in
// {2,3} exactly 5 (targetList==0) or 10 (targetList==1) x Decode4 mapId into
// adwMapTransfer/adwMapTransferEx @0x9fa01d-23. Confirms the "identical v83
// 0xA25268 / v95 0x9F9F90" claim in the file-level comment above.
//
// packet-audit:verify packet=teleportrock/clientbound/MapTransferList version=gms_v95 ida=0x9f9f90
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

// IDA (live MapleStory_dump.exe v83, port 13342, task-124 verify pass):
// CWvsContext::OnMapTransferResult @0xa25268, same function as the list form
// above — modes 5-11 (CANNOT_GO / UNABLE_TO_LOCATE / CANNOT_GO_CONTINENT /
// CURRENT_MAP / MAP_NOT_AVAILABLE / MAPLE_ISLAND_LEVEL7, all IDA-confirmed
// against the v83 seed template's teleportrock `operations` table) read only
// mode(Decode1)+targetList(Decode1) and never reach the Decode4 mapId loop
// (that read is guarded on `v3 == 3 && !v4`, i.e. only the list modes). The
// analyzer's flat diff cannot special-case per-mode field counts against a
// single guarded raw-call sequence shared with MapTransferList, so it grades
// this candidate FlatInvalid ("atlas short — missing trailing field") even
// though the 2-byte body is exactly what v83 sends for modes 5-11 — the same
// class of runtime/mode-guard tooling gap documented on the cash ItemUsePointReset
// >=87 fixtures (item_use_point_reset_test.go). Evidence pinned to carry this
// cell via the linked-fixture path.
//
// packet-audit:verify packet=teleportrock/clientbound/MapTransferError version=gms_v83 ida=0xa25268
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

// task-124 v95 verify pass (live GMS_v95.0_U_DEVM.exe, port 13341):
// CWvsContext::OnMapTransferResult @0x9f9f90, same function as the list form
// above — modes 5-11 read only mode(Decode1)+targetList(Decode1) and never
// reach the Decode4 mapId loop (guarded on `case 2/3` in the v95 switch,
// byte-identical branch structure to v83's `v3 == 3 && !v4` guard). Same
// class of runtime/mode-guard flat-diff tooling gap as v83 (report graded
// FlatInvalid "atlas short — missing trailing field" even though the 2-byte
// body is exactly what v95 sends for modes 5-11). Evidence pinned to carry
// this cell via the linked-fixture path, mirroring the v83 convention above.
//
// packet-audit:verify packet=teleportrock/clientbound/MapTransferError version=gms_v95 ida=0x9f9f90
func TestMapTransferErrorGoldenV95(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 95, 1)
	m := NewMapTransferError(5, false)
	got := m.Encode(l, ctx)(nil)
	want := []byte{0x05, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

// task-124 v84 verify pass (live GMS_v84.1_U_DEVM.exe, port 13345):
// CWvsContext::OnMapTransferResult @0xa70963 — unnamed in the v84 IDB
// (sub_A70963) until this pass, renamed live; dispatched from
// CWvsContext::OnPacket's case 0x2A @0xa51dfd (CWvsContext::OnPacket itself
// was already named in this IDB). Byte-identical read order to v83
// 0xa25268: mode(Decode1)+targetList(Decode1) @0xa70983/0xa70986, then for
// mode in {2,3} exactly 5 (targetList==0) or 10 (targetList==1) x Decode4
// mapId @0xa709d6 loop. Confirms the "identical v83 0xA25268 / v95 0x9F9F90"
// claim in the file-level comment above for v84 too.
//
// packet-audit:verify packet=teleportrock/clientbound/MapTransferList version=gms_v84 ida=0xa70963
func TestMapTransferListRegularGoldenV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
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

// IDA (live GMS_v84.1_U_DEVM.exe v84, port 13345, task-124 verify pass):
// CWvsContext::OnMapTransferResult @0xa70963, same function as the list form
// above — modes 5-11 (error/notice form, StringPool ids 2985/2950/2956/2953/
// 2957 — a small offset from v83's 2984/2949/2955/2952/2956 that does not
// affect wire bytes) read only mode(Decode1)+targetList(Decode1) and never
// reach the Decode4 mapId loop. Same class of runtime/mode-guard tooling gap
// as v83/v95 (analyzer grades this candidate FlatInvalid even though the
// 2-byte body is exactly what v84 sends for modes 5-11). Evidence pinned to
// carry this cell via the linked-fixture path, mirroring the v83/v95
// convention above.
//
// packet-audit:verify packet=teleportrock/clientbound/MapTransferError version=gms_v84 ida=0xa70963
func TestMapTransferErrorGoldenV84(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 84, 1)
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
