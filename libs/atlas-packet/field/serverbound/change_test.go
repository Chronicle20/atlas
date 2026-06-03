package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestChangeWithPortalRoundTrip covers a portal-named transfer. Per the v95
// client (CField::SendTransferFieldRequest @0x5345c0) a non-empty portal name
// carries the target x/y pair, so x/y participate in the round-trip here.
func TestChangeWithPortalRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Change{fieldKey: 1, targetId: 100000000, portalName: "west00", x: 100, y: 200, unused: 0, premium: 0}
			output := Change{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.FieldKey() != input.FieldKey() {
				t.Errorf("fieldKey: got %v, want %v", output.FieldKey(), input.FieldKey())
			}
			if output.TargetId() != input.TargetId() {
				t.Errorf("targetId: got %v, want %v", output.TargetId(), input.TargetId())
			}
			if output.PortalName() != input.PortalName() {
				t.Errorf("portalName: got %v, want %v", output.PortalName(), input.PortalName())
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

// TestChangeNoPortalRoundTrip covers the null-portal (Revive) path: an empty
// portal name means the client emits NO x/y coordinates.
func TestChangeNoPortalRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Change{fieldKey: 2, targetId: 240000000, portalName: "", unused: 0, premium: 1}
			output := Change{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.FieldKey() != input.FieldKey() {
				t.Errorf("fieldKey: got %v, want %v", output.FieldKey(), input.FieldKey())
			}
			if output.TargetId() != input.TargetId() {
				t.Errorf("targetId: got %v, want %v", output.TargetId(), input.TargetId())
			}
			if output.PortalName() != input.PortalName() {
				t.Errorf("portalName: got %q, want %q", output.PortalName(), input.PortalName())
			}
			if output.Premium() != input.Premium() {
				t.Errorf("premium: got %v, want %v", output.Premium(), input.Premium())
			}
		})
	}
}

// TestChangeWithChaseRoundTrip covers the chase path, which trails the target
// x/y (Encode4 each) after the chase flag.
func TestChangeWithChaseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		// The chase flag is gated on GMS && Major>=83 (s_bChase is a GMS-client
		// global); other variants never serialize it, so a chase payload is not
		// representable on their wire.
		if !(v.Region == "GMS" && v.MajorVersion >= 83) {
			continue
		}
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Change{fieldKey: 3, targetId: 100000000, portalName: "east00", x: 50, y: 75, unused: 0, premium: 0, chase: true, targetX: 1234, targetY: -5678}
			output := Change{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Chase() != input.Chase() {
				t.Errorf("chase: got %v, want %v", output.Chase(), input.Chase())
			}
			if output.Chase() {
				if output.TargetX() != input.TargetX() {
					t.Errorf("targetX: got %v, want %v", output.TargetX(), input.TargetX())
				}
				if output.TargetY() != input.TargetY() {
					t.Errorf("targetY: got %v, want %v", output.TargetY(), input.TargetY())
				}
			}
		})
	}
}

// TestChangeCashShopReturnRoundTrip covers the cash-shop return variant, which
// the client (CCashShop::SendTransferFieldPacket @0x494a20) sends as an
// empty-body opcode-41 packet.
func TestChangeCashShopReturnRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Change{cashShopReturn: true}
			output := Change{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !output.CashShopReturn() {
				t.Errorf("cashShopReturn: got %v, want true", output.CashShopReturn())
			}
		})
	}
}
