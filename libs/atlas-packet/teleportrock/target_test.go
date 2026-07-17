package teleportrock

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func decodeTarget(t *testing.T, b []byte) Target {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	out := Target{}
	out.Decode(l)(&r)
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
	out := decodeTarget(t, append(got, 0x00, 0x00, 0x00, 0x00)) // + trailing updateTime budget
	if !out.Valid() || out.ByName() || out.TargetMap() != 100000000 {
		t.Fatalf("decode: %+v", out)
	}
}

func TestTargetByNameRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewTargetByName("Adele")
	w := response.NewWriter(l)
	in.Encode(w)
	out := decodeTarget(t, append(w.Bytes(), 0x00, 0x00, 0x00, 0x00))
	if !out.Valid() || !out.ByName() || out.TargetName() != "Adele" {
		t.Fatalf("decode: %+v", out)
	}
}

// The client sends no target payload at all when the dialog resolves with
// neither a name nor a valid map (design §1 Q1 caveat). Only the trailing
// updateTime remains — decode must flag invalid, not read past the buffer.
func TestTargetAbsentPayloadIsInvalid(t *testing.T) {
	out := decodeTarget(t, []byte{0x12, 0x34, 0x56, 0x78}) // 4 bytes = updateTime only
	if out.Valid() {
		t.Fatalf("absent payload must decode as invalid")
	}
}

func TestTargetByMapTruncatedIsInvalid(t *testing.T) {
	// byName=0 but only the 4 trailing updateTime bytes remain — no map id.
	out := decodeTarget(t, []byte{0x00, 0x12, 0x34, 0x56, 0x78})
	if out.Valid() {
		t.Fatalf("byName=0 without a map id must decode as invalid")
	}
}

func TestTargetEmptyNameIsInvalid(t *testing.T) {
	// byName=1, zero-length string, trailing updateTime.
	out := decodeTarget(t, []byte{0x01, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78})
	if out.Valid() {
		t.Fatalf("empty target name must decode as invalid")
	}
}
