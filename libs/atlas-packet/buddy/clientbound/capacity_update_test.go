package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=buddy/clientbound/BuddyCapacityUpdate version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyCapacityUpdate version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyCapacityUpdate version=gms_v95 ida=0xa12630
func TestBuddyCapacityUpdateRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuddyCapacityUpdate(15, 50)
			output := CapacityUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Capacity() != input.Capacity() {
				t.Errorf("capacity: got %v, want %v", output.Capacity(), input.Capacity())
			}
		})
	}
}
