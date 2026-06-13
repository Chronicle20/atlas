package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v83 ida=0xa09118
// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v87 ida=0xa9e8f6
// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v95 ida=0x9d5d50
// packet-audit:verify packet=drop/serverbound/DropPickUp version=jms_v185 ida=0xaedb0f
// packet-audit:verify packet=drop/serverbound/DropPickUp version=gms_v84 ida=0xa5342c
func TestPickUpRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := PickUp{fieldKey: 1, updateTime: 100, x: 50, y: 60, dropId: 12345, crc: 99}
			output := PickUp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.DropId() != input.DropId() {
				t.Errorf("dropId: got %v, want %v", output.DropId(), input.DropId())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}
