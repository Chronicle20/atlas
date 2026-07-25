package clientbound

import (
	"testing"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestTownPortalWire pins the PARTY_OPERATION town-portal body across versions.
// Body = mode1+slot1+town4+target4+[skillId4 GMS v95+]+x2+y2: 14 bytes (v83/
// v84/v87/jms), 18 bytes (GMS v95+). x/y are 2 bytes (Decode2), unlike the
// 4-byte PARTYDATA aTownPortal coordinates. Round-trips per version. Per-version
// IDA provenance (OnPartyResult cases) is recorded on the TownPortal type doc;
// no packet-audit:verify marker is claimed until the evidence+report chain is
// promoted (the byte test stands alone).
func TestTownPortalWire(t *testing.T) {
	want := map[string]int{
		"GMS v28": 14, "GMS v83": 14, "GMS v84": 14, "GMS v86": 14,
		"GMS v87": 14, "GMS v95": 18, "JMS v185": 14,
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := NewTownPortal(0x25, 2, _map.Id(104000000), _map.Id(100000000), 1234, -567)
			b := m.Encode(logrus.New(), ctx)(nil)
			wl, ok := want[v.Name]
			if !ok {
				t.Fatalf("no expected length for %s", v.Name)
			}
			if len(b) != wl {
				t.Fatalf("length: got %d, want %d", len(b), wl)
			}
			if b[0] != 0x25 || b[1] != 2 {
				t.Fatalf("mode/slot header: got [%#x %#x], want [0x25 0x02]", b[0], b[1])
			}

			out := TownPortal{}
			req := request.Request(b)
			r := request.NewRequestReader(&req, 0)
			out.Decode(logrus.New(), ctx)(&r, nil)
			if out.Mode() != 0x25 || out.Slot() != 2 || out.TownMapId() != _map.Id(104000000) ||
				out.TargetMapId() != _map.Id(100000000) || out.X() != 1234 || out.Y() != -567 {
				t.Fatalf("round-trip mismatch: %s", out.String())
			}
		})
	}
}

// TestTownPortalClear pins the door-removed clear: both map ids encode the
// empty-map sentinel so the client's render loop skips the slot.
func TestTownPortalClear(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	m := NewTownPortalClear(0x25, 3)
	b := m.Encode(logrus.New(), ctx)(nil)
	out := TownPortal{}
	req := request.Request(b)
	r := request.NewRequestReader(&req, 0)
	out.Decode(logrus.New(), ctx)(&r, nil)
	if out.Slot() != 3 || out.TownMapId() != _map.EmptyMapId || out.TargetMapId() != _map.EmptyMapId {
		t.Fatalf("clear must encode EmptyMapId both ids: %s", out.String())
	}
}
