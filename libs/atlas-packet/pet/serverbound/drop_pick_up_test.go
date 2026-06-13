package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v87 ida=0x749be8
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v95 ida=0x6a0820
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v83 ida=0x705c7c
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=jms_v185 ida=0x76bcc6
// packet-audit:verify packet=pet/serverbound/PetDropPickUp version=gms_v84 ida=0x722672
func TestDropPickUpRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// Use dropId=14 (14%13!=0) to exercise the extended fields path
			input := DropPickUp{petId: 12345, fieldKey: 1, updateTime: 100, x: 50, y: -30, dropId: 14, crc: 999, bPickupOthers: true, bSweepForDrop: false, bLongRange: true, ownerX: 10, ownerY: 20, posCrc: 111, rectCrc: 222}
			output := DropPickUp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if output.DropId() != input.DropId() {
				t.Errorf("dropId: got %v, want %v", output.DropId(), input.DropId())
			}
			if output.BPickupOthers() != input.BPickupOthers() {
				t.Errorf("bPickupOthers: got %v, want %v", output.BPickupOthers(), input.BPickupOthers())
			}
		})
	}
}

func TestDropPickUpDivisibleByThirteenRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// Use dropId=26 (26%13==0) to exercise the non-extended path
			input := DropPickUp{petId: 12345, fieldKey: 1, updateTime: 100, x: 50, y: -30, dropId: 26, crc: 999, bPickupOthers: false, bSweepForDrop: true, bLongRange: false}
			output := DropPickUp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.DropId() != input.DropId() {
				t.Errorf("dropId: got %v, want %v", output.DropId(), input.DropId())
			}
			if output.BSweepForDrop() != input.BSweepForDrop() {
				t.Errorf("bSweepForDrop: got %v, want %v", output.BSweepForDrop(), input.BSweepForDrop())
			}
		})
	}
}
