package clientbound

import (
	"context"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// TestTownPortalWire pins the v83 PARTY_OPERATION town-portal body the client
// reads in CWvsContext::OnPartyResult (v83 @0xa3e31c case 0x25 / v84 @0xa89cf3
// case 0x28): Decode1 mode, Decode1 slot, Decode4 townId, Decode4 targetId,
// Decode2 x, Decode2 y = 14 bytes. x/y are 2 bytes here (unlike the 4-byte
// PARTYDATA aTownPortal coordinates).
func TestTownPortalWire(t *testing.T) {
	m := NewTownPortal(0x25, 2, _map.Id(104000000), _map.Id(100000000), 1234, -567)
	b := m.Encode(logrus.New(), context.Background())(nil)
	if len(b) != 14 {
		t.Fatalf("town-portal length: got %d, want 14 (mode1+slot1+town4+target4+x2+y2)", len(b))
	}
	if b[0] != 0x25 || b[1] != 2 {
		t.Fatalf("mode/slot header: got [%#x %#x], want [0x25 0x02]", b[0], b[1])
	}

	out := TownPortal{}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	out.Decode(logrus.New(), context.Background())(&r, nil)
	if out.Mode() != 0x25 || out.Slot() != 2 || out.TownMapId() != _map.Id(104000000) ||
		out.TargetMapId() != _map.Id(100000000) || out.X() != 1234 || out.Y() != -567 {
		t.Fatalf("round-trip mismatch: %s", out.String())
	}
}

// TestTownPortalClear pins the door-removed clear: both map ids encode the
// empty-map sentinel so the client's render loop skips the slot.
func TestTownPortalClear(t *testing.T) {
	m := NewTownPortalClear(0x25, 3)
	b := m.Encode(logrus.New(), context.Background())(nil)
	if len(b) != 14 {
		t.Fatalf("clear length: got %d, want 14", len(b))
	}
	out := TownPortal{}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	out.Decode(logrus.New(), context.Background())(&r, nil)
	if out.Slot() != 3 || out.TownMapId() != _map.EmptyMapId || out.TargetMapId() != _map.EmptyMapId {
		t.Fatalf("clear must encode EmptyMapId both ids: %s", out.String())
	}
}
