package teleportrock

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func decodeTarget(t *testing.T, b []byte, hasTrailingUpdateTime bool) Target {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	out := Target{}
	out.Decode(l, hasTrailingUpdateTime)(&r)
	return out
}

func TestTargetByMapRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewTargetByMap(100000000)
	w := response.NewWriter(l)
	in.Encode(w)
	// byName=0, mapId=100000000 LE, plus the trailing updateTime the wrapping op appends
	want := []byte{0x00, 0x00, 0xE1, 0xF5, 0x05}
	got := w.Bytes()
	if len(got) != len(want) {
		t.Fatalf("encoded length: got %d want %d (% x)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got %x want %x", i, got[i], want[i])
		}
	}
	out := decodeTarget(t, append(got, 0x00, 0x00, 0x00, 0x00), true) // + trailing updateTime budget
	if !out.Valid() || out.ByName() || out.TargetMap() != 100000000 {
		t.Fatalf("decode: %+v", out)
	}
}

func TestTargetByNameRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewTargetByName("Adele")
	w := response.NewWriter(l)
	in.Encode(w)
	out := decodeTarget(t, append(w.Bytes(), 0x00, 0x00, 0x00, 0x00), true)
	if !out.Valid() || !out.ByName() || out.TargetName() != "Adele" {
		t.Fatalf("decode: %+v", out)
	}
}

// The client sends no target payload at all when the dialog resolves with
// neither a name nor a valid map (design §1 Q1 caveat). Only the trailing
// updateTime remains — decode must flag invalid, not read past the buffer.
func TestTargetAbsentPayloadIsInvalid(t *testing.T) {
	out := decodeTarget(t, []byte{0x12, 0x34, 0x56, 0x78}, true) // 4 bytes = updateTime only
	if out.Valid() {
		t.Fatalf("absent payload must decode as invalid")
	}
}

func TestTargetByMapTruncatedIsInvalid(t *testing.T) {
	// byName=0 but only the 4 trailing updateTime bytes remain — no map id.
	out := decodeTarget(t, []byte{0x00, 0x12, 0x34, 0x56, 0x78}, true)
	if out.Valid() {
		t.Fatalf("byName=0 without a map id must decode as invalid")
	}
}

func TestTargetEmptyNameIsInvalid(t *testing.T) {
	// byName=1, zero-length string, trailing updateTime.
	out := decodeTarget(t, []byte{0x01, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78}, true)
	if out.Valid() {
		t.Fatalf("empty target name must decode as invalid")
	}
}

// The following hasTrailingUpdateTime=false cases model
// cash/serverbound.ItemUseTeleportRock on MajorVersion()>=87 (v87/v95/jms):
// task-124's v95 verify pass (live GMS_v95.0_U_DEVM.exe, port 13341) found
// CWvsContext::SendConsumeCashItemUseRequest @0x9eb3e0 writes update_time ONCE,
// in the common header BEFORE the case switch — already consumed by the
// parent ItemUse struct — so case 22 ($LN84_16 @0x9ee059)'s RunMapTransferItem
// target payload is the LAST thing on the wire; nothing trails it. Reserving
// a phantom 4-byte budget here (the bug this pass fixed) would misdecode a
// genuine 5-byte by-map payload as absent.

func TestTargetByMapRoundTripNoTrailingBudget(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewTargetByMap(100000000)
	w := response.NewWriter(l)
	in.Encode(w)
	got := w.Bytes() // byName=0 + mapId only — nothing else on the wire
	out := decodeTarget(t, got, false)
	if !out.Valid() || out.ByName() || out.TargetMap() != 100000000 {
		t.Fatalf("decode: %+v", out)
	}
}

func TestTargetByNameRoundTripNoTrailingBudget(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewTargetByName("Adele")
	w := response.NewWriter(l)
	in.Encode(w)
	out := decodeTarget(t, w.Bytes(), false)
	if !out.Valid() || !out.ByName() || out.TargetName() != "Adele" {
		t.Fatalf("decode: %+v", out)
	}
}

func TestTargetAbsentPayloadIsInvalidNoTrailingBudget(t *testing.T) {
	// Nothing at all remains — no trailing budget to fall back on for v87+.
	out := decodeTarget(t, []byte{}, false)
	if out.Valid() {
		t.Fatalf("absent payload must decode as invalid")
	}
}

func TestTargetByMapTruncatedIsInvalidNoTrailingBudget(t *testing.T) {
	// byName=0 but zero bytes remain — no map id, no selection.
	out := decodeTarget(t, []byte{0x00}, false)
	if out.Valid() {
		t.Fatalf("byName=0 without a map id must decode as invalid")
	}
}
