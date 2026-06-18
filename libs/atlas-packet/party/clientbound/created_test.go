package clientbound

import (
	"encoding/binary"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v95 ida=0xa10efc
// packet-audit:verify packet=party/clientbound/PartyCreated version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v84 ida=0xa89cf3
func TestCreatedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCreated(7, 12345)
			output := Created{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
		})
	}
}

// TestCreatedNoDoorSentinel verifies that a Created without door fields encodes
// the empty-map sentinel (999999999 / 0x3B9AC9FF) for both map ids and zero
// for x/y — byte-identical to the previous hard-coded output.
func TestCreatedNoDoorSentinel(t *testing.T) {
	emptyLE := make([]byte, 4)
	binary.LittleEndian.PutUint32(emptyLE, uint32(_map.EmptyMapId))

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := pt.Encode(t, ctx, NewCreated(7, 12345).Encode, nil)
			// Door block starts at offset 5 (1 mode byte + 4 partyId bytes).
			doorBlock := got[5:]
			// bytes 0-3: townMapId
			if doorBlock[0] != emptyLE[0] || doorBlock[1] != emptyLE[1] ||
				doorBlock[2] != emptyLE[2] || doorBlock[3] != emptyLE[3] {
				t.Errorf("townMapId sentinel: got %x, want %x", doorBlock[0:4], emptyLE)
			}
			// bytes 4-7: targetMapId
			if doorBlock[4] != emptyLE[0] || doorBlock[5] != emptyLE[1] ||
				doorBlock[6] != emptyLE[2] || doorBlock[7] != emptyLE[3] {
				t.Errorf("targetMapId sentinel: got %x, want %x", doorBlock[4:8], emptyLE)
			}
			// bytes 8-9: door x (must be 0x00 0x00)
			if doorBlock[8] != 0 || doorBlock[9] != 0 {
				t.Errorf("doorX: got %x %x, want 00 00", doorBlock[8], doorBlock[9])
			}
			// bytes 10-11: door y (must be 0x00 0x00)
			if doorBlock[10] != 0 || doorBlock[11] != 0 {
				t.Errorf("doorY: got %x %x, want 00 00", doorBlock[10], doorBlock[11])
			}
		})
	}
}

// TestCreatedWithDoor verifies that a Created built with door fields encodes the
// town map id, target map id, and minimap x/y instead of the empty sentinel.
func TestCreatedWithDoor(t *testing.T) {
	const (
		wantTownMapId   _map.Id = 10000
		wantTargetMapId _map.Id = 104040000
		wantX           int16   = -300
		wantY           int16   = 150
	)

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCreated(7, 12345).WithDoor(wantTownMapId, wantTargetMapId, wantX, wantY)
			got := pt.Encode(t, ctx, input.Encode, nil)

			// Door block starts at offset 5.
			doorBlock := got[5:]

			townLE := binary.LittleEndian.Uint32(doorBlock[0:4])
			if townLE != uint32(wantTownMapId) {
				t.Errorf("townMapId: got %d, want %d", townLE, uint32(wantTownMapId))
			}

			targetLE := binary.LittleEndian.Uint32(doorBlock[4:8])
			if targetLE != uint32(wantTargetMapId) {
				t.Errorf("targetMapId: got %d, want %d", targetLE, uint32(wantTargetMapId))
			}

			xLE := int16(binary.LittleEndian.Uint16(doorBlock[8:10]))
			if xLE != wantX {
				t.Errorf("doorX: got %d, want %d", xLE, wantX)
			}

			yLE := int16(binary.LittleEndian.Uint16(doorBlock[10:12]))
			if yLE != wantY {
				t.Errorf("doorY: got %d, want %d", yLE, wantY)
			}
		})
	}
}
