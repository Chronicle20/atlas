package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=buddy/serverbound/BuddyOperationAccept version=gms_v95 ida=0x52f290
// packet-audit:verify packet=buddy/serverbound/BuddyOperationAccept version=jms_v185 ida=0x56e66c
// packet-audit:verify packet=buddy/serverbound/BuddyOperationAccept version=gms_v87 ida=0x558a93
// packet-audit:verify packet=buddy/serverbound/BuddyOperationAccept version=gms_v83 ida=0x531270
func TestOperationAcceptRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationAccept{fromCharacterId: 12345}
			output := OperationAccept{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.FromCharacterId() != input.FromCharacterId() {
				t.Errorf("fromCharacterId: got %v, want %v", output.FromCharacterId(), input.FromCharacterId())
			}
		})
	}
}
