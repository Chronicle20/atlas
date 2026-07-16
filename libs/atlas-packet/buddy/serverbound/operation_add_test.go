package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=buddy/serverbound/BuddyOperationAdd version=gms_v95 ida=0x535240
// packet-audit:verify packet=buddy/serverbound/BuddyOperationAdd version=jms_v185 ida=0x56e41d
// packet-audit:verify packet=buddy/serverbound/BuddyOperationAdd version=gms_v87 ida=0x558844
func TestOperationAddRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			// The trailing buddy-group name is only on the wire for GMS v72+
			// (and JMS); v28/v48/v61 omit it (IDA-verified, see operation_add.go).
			// Only set/assert group where the version actually carries it, so the
			// legacy variants round-trip cleanly.
			hasGroup := v.MajorVersion > 61
			input := OperationAdd{name: "TestBuddy"}
			if hasGroup {
				input.group = "Default Group"
			}
			output := OperationAdd{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Group() != input.Group() {
				t.Errorf("group: got %v, want %v", output.Group(), input.Group())
			}
		})
	}
}
