package serverbound

import (
	"encoding/binary"
	"testing"

	"github.com/sirupsen/logrus"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestItemUseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUse{updateTime: 12345, source: 5, itemId: 5000000}
			output := ItemUse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Source() != input.Source() {
				t.Errorf("source: got %v, want %v", output.Source(), input.Source())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}

			// updateTime is only carried in the common ItemUse envelope for
			// versions where the client writes it leading (front of packet,
			// before source+itemId). For trailing versions it lives in the
			// per-arm sub-body (see item_use_pet_consumable.go etc.), so the
			// common decoder never reads it and it stays zero-valued here.
			updateTimeFirst := (v.Region == "GMS" && v.MajorVersion >= 87) || v.Region == "JMS"
			if updateTimeFirst {
				if output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
				}
			} else {
				if output.UpdateTime() != 0 {
					t.Errorf("updateTime: expected common envelope to leave updateTime unread (0) for trailing variant %s, got %v", v.Name, output.UpdateTime())
				}
			}
		})
	}
}

// TestItemUseUpdateTimeLeading proves the exact byte layout per version,
// IDA-verified (CWvsContext::SendConsumeCashItemUseRequest):
//   - leading (front of packet, before source+itemId): gms_v87 (0xa9fef9,
//     opcode 0x52), gms_v95 (0x9eb3e0, opcode 0x55), jms_v185 (0xaef2f5,
//     opcode 0x47).
//   - trailing (end of sub-body, not part of this common envelope):
//     gms_v83 (design/fixture-verified) and gms_v84 (byte-identical to v83
//     per the task-083 audit).
//
// This is distinct from TestItemUseRoundTrip: a self-consistent Encode/Decode
// round trip cannot detect a wrong-but-symmetric predicate, since Encode and
// Decode always agree with each other. This test inspects the raw wire bytes
// directly to pin the actual position.
func TestItemUseUpdateTimeLeading(t *testing.T) {
	cases := []struct {
		name         string
		region       string
		majorVersion uint16
		leading      bool
	}{
		{"GMS v83", "GMS", 83, false},
		{"GMS v84", "GMS", 84, false},
		{"GMS v87", "GMS", 87, true},
		{"GMS v95", "GMS", 95, true},
		{"JMS v185", "JMS", 185, true},
	}

	input := ItemUse{updateTime: 0xAABBCCDD, source: 0x1234, itemId: 0x05000001}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := pt.CreateContext(c.region, c.majorVersion, 1)
			bytes := pt.Encode(t, ctx, input.Encode, nil)

			if c.leading {
				if len(bytes) != 10 {
					t.Fatalf("expected 10 bytes (updateTime+source+itemId) for leading variant, got %d", len(bytes))
				}
				gotUpdateTime := binary.LittleEndian.Uint32(bytes[0:4])
				if gotUpdateTime != uint32(input.updateTime) {
					t.Errorf("updateTime not at front: got %#x, want %#x", gotUpdateTime, input.updateTime)
				}
				gotSource := int16(binary.LittleEndian.Uint16(bytes[4:6]))
				if gotSource != input.source {
					t.Errorf("source: got %#x, want %#x", gotSource, input.source)
				}
				gotItemId := binary.LittleEndian.Uint32(bytes[6:10])
				if gotItemId != input.itemId {
					t.Errorf("itemId: got %#x, want %#x", gotItemId, input.itemId)
				}
			} else {
				if len(bytes) != 6 {
					t.Fatalf("expected 6 bytes (source+itemId only, updateTime trailing/elsewhere) for trailing variant, got %d", len(bytes))
				}
				gotSource := int16(binary.LittleEndian.Uint16(bytes[0:2]))
				if gotSource != input.source {
					t.Errorf("source: got %#x, want %#x", gotSource, input.source)
				}
				gotItemId := binary.LittleEndian.Uint32(bytes[2:6])
				if gotItemId != input.itemId {
					t.Errorf("itemId: got %#x, want %#x", gotItemId, input.itemId)
				}
			}
		})
	}
}

// TestItemUseUpdateTimeFirst pins the updateTime position per version.
// IDA-verified (task-137 design §1.1, CWvsContext::SendConsumeCashItemUseRequest):
// the six GMS versions <= v84 (v48 0x70e495, v61 0x832a5d, v72 0x904fe2,
// v79 0x95634a, v83 0xa0a63f, v84 0xa54a2f) encode updateTime TRAILING; GMS
// v87 (0xa9fef9), v95 (0x9eb3e0) and JMS185 (0xaef2f5) encode it LEADING,
// immediately after the opcode.
func TestItemUseUpdateTimeFirst(t *testing.T) {
	cases := []struct {
		name    string
		region  string
		major   uint16
		leading bool
	}{
		{"GMS v48 trailing", "GMS", 48, false},
		{"GMS v61 trailing", "GMS", 61, false},
		{"GMS v72 trailing", "GMS", 72, false},
		{"GMS v79 trailing", "GMS", 79, false},
		{"GMS v83 trailing", "GMS", 83, false},
		{"GMS v84 trailing", "GMS", 84, false},
		{"GMS v87 leading", "GMS", 87, true},
		{"GMS v95 leading", "GMS", 95, true},
		{"JMS v185 leading", "JMS", 185, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.region, tc.major, 1)

			tn := tenant.MustFromContext(ctx)
			if got := UpdateTimeFirst(tn); got != tc.leading {
				t.Errorf("UpdateTimeFirst: got %v, want %v", got, tc.leading)
			}

			// leading layout: updateTime(4), slot(2), itemId(4)
			// trailing layout: slot(2), itemId(4) — trailing updateTime is read by the arm codec
			var raw []byte
			if tc.leading {
				raw = []byte{
					0x64, 0x00, 0x00, 0x00, // updateTime = 100
					0x07, 0x00, // slot = 7
					0xD0, 0xAA, 0x4D, 0x00, // itemId = 5090000
				}
			} else {
				raw = []byte{
					0x07, 0x00, // slot = 7
					0xD0, 0xAA, 0x4D, 0x00, // itemId = 5090000
				}
			}
			req := request.Request(raw)
			reader := request.NewRequestReader(&req, 0)

			p := ItemUse{}
			p.Decode(logrus.New(), ctx)(&reader, map[string]interface{}{})

			if p.Source() != 7 {
				t.Errorf("source: got %d, want 7", p.Source())
			}
			if p.ItemId() != 5090000 {
				t.Errorf("itemId: got %d, want 5090000", p.ItemId())
			}
			if tc.leading && p.UpdateTime() != 100 {
				t.Errorf("updateTime: got %d, want 100", p.UpdateTime())
			}
		})
	}
}
