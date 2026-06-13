package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=buddy/serverbound/BuddyOperationDelete version=gms_v95 ida=0x52f170
// packet-audit:verify packet=buddy/serverbound/BuddyOperationDelete version=jms_v185 ida=0x56e5bd
// packet-audit:verify packet=buddy/serverbound/BuddyOperationDelete version=gms_v87 ida=0x5589e4
// packet-audit:verify packet=buddy/serverbound/BuddyOperationDelete version=gms_v83 ida=0x5311c1
// packet-audit:verify packet=buddy/serverbound/BuddyOperationDelete version=gms_v84 ida=0x53d443
func TestOperationDeleteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationDelete{buddyCharacterId: 67890}
			output := OperationDelete{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.BuddyCharacterId() != input.BuddyCharacterId() {
				t.Errorf("buddyCharacterId: got %v, want %v", output.BuddyCharacterId(), input.BuddyCharacterId())
			}
		})
	}
}
