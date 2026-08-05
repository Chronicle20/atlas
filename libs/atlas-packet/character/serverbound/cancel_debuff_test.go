package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CANCEL_DEBUFF is CWvsContext::CheckTemporaryStatDuration: the client finds a
// locally-expired temporary stat via SecondaryStat::CheckByTime, computes the
// expired mask — and then does NOT transmit it. Every client examined
// constructs COutPacket(opcode) and calls SendPacket with no intervening
// encode calls, so the body is empty on all ten versions and there is nothing
// to version-gate. Evidence: docs/tasks/task-190-disease-duration-cancel-debuff/
// investigation.md §8.1/§8.2.
//
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v48 ida=0x71b126
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v61 ida=0x84374e
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v72 ida=0x91914f
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v79 ida=0x96ad48
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v83 ida=0xa20935
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v84 ida=0xa6bd3a
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v87 ida=0xab7fd7
// packet-audit:verify packet=character/serverbound/CancelDebuff version=gms_v95 ida=0x9f2d30
// packet-audit:verify packet=character/serverbound/CancelDebuff version=jms_v185 ida=0xb0783e
func TestCancelDebuffRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CancelDebuff{}
			output := CancelDebuff{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestCancelDebuffEmptyBodyAllVersions pins the empty body per version. The
// v48 client builds the packet with the three-argument
// COutPacket::COutPacket(v5, 78, 0) constructor; that third argument is a
// client-side construction detail with no wire consequence, so v48 encodes the
// same zero bytes as every other version.
func TestCancelDebuffEmptyBodyAllVersions(t *testing.T) {
	versions := []struct {
		name   string
		region string
		major  uint16
		minor  uint16
	}{
		{"gms_v48", "GMS", 48, 1},
		{"gms_v61", "GMS", 61, 1},
		{"gms_v72", "GMS", 72, 1},
		{"gms_v79", "GMS", 79, 1},
		{"gms_v83", "GMS", 83, 1},
		{"gms_v84", "GMS", 84, 1},
		{"gms_v87", "GMS", 87, 1},
		{"gms_v92", "GMS", 92, 1},
		{"gms_v95", "GMS", 95, 1},
		{"jms_v185", "JMS", 185, 1},
	}
	for _, v := range versions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			got := pt.Encode(t, ctx, CancelDebuff{}.Encode, nil)
			if len(got) != 0 {
				t.Errorf("expected empty body, got % x", got)
			}
		})
	}
}

// TestCancelDebuffOperation pins the handler-map key. atlas-channel binds the
// handler by NAME through tenant socket config, never by opcode — 0x63 means
// CANCEL_DEBUFF at v83/v84 but is the calc-damage-stat request at v61, so a
// hard-coded opcode would mis-route (FR-2.3.2, DOM-25).
func TestCancelDebuffOperation(t *testing.T) {
	if got := (CancelDebuff{}).Operation(); got != CancelDebuffHandle {
		t.Errorf("operation: got %q, want %q", got, CancelDebuffHandle)
	}
}
