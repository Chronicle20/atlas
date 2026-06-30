package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestChangeByteOutputV79 pins the gms_v79 CHANGE_MAP (op 0x24) serverbound wire.
//
// IDA: CField::SendTransferFieldRequest @0x51b950 (GMS_v79_1_DEVM.exe) —
//
//	COutPacket(36)                  @0x51b98d → opcode 0x24 (matches registry).
//	Encode1(get_field()+276)        @0x51b9b2 → fieldKey byte.
//	Encode4(a2)                     @0x51b9bd → targetId (int32 LE).
//	EncodeStr(Src)                  @0x51b9e2 → portalName.
//	if (Src) Encode2(x)/Encode2(y)  @0x51ba00/@0x51ba1a → target x/y (only with a portal name).
//	Encode1(0)                      @0x51ba23 → unused byte.
//	Encode1(a4)                     @0x51ba2e → premium byte.
//	Encode1(dword_B0D450)           @0x51ba3e → chase flag (present from v79; the legacy
//	                                            codec gate was >=83 and wrongly dropped it).
//	if (chase) Encode4(targetX)/Encode4(targetY) — omitted here (chase=false).
//
// WriteAsciiString = uint16-LE len + bytes; WriteInt = uint32-LE; WriteInt16 = uint16-LE.
func TestChangeByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := Change{fieldKey: 1, targetId: 100000000, portalName: "west00", x: 100, y: 200, unused: 0, premium: 0}
	expected := []byte{
		0x01,                   // fieldKey @0x51b9b2
		0x00, 0xE1, 0xF5, 0x05, // targetId 100000000 @0x51b9bd
		0x06, 0x00, 0x77, 0x65, 0x73, 0x74, 0x30, 0x30, // EncodeStr("west00") @0x51b9e2
		0x64, 0x00, // x=100 @0x51ba00
		0xC8, 0x00, // y=200 @0x51ba1a
		0x00, // unused @0x51ba23
		0x00, // premium @0x51ba2e
		0x00, // chase=false @0x51ba3e
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 change golden mismatch: got %v want %v", actual, expected)
	}
}

// TestChangeWithPortalRoundTrip covers a portal-named transfer. Per the v95
// client (CField::SendTransferFieldRequest @0x5345c0) a non-empty portal name
// carries the target x/y pair, so x/y participate in the round-trip here.
// packet-audit:verify packet=field/serverbound/FieldChange version=gms_v79 ida=0x51b950
// packet-audit:verify packet=field/serverbound/FieldChange version=gms_v95 ida=0x5345c0
// packet-audit:verify packet=field/serverbound/FieldChange version=gms_v83 ida=0x53035d
// packet-audit:verify packet=field/serverbound/FieldChange version=gms_v87 ida=0x557b5a
// packet-audit:verify packet=field/serverbound/FieldChange version=jms_v185 ida=0x56d75a
// packet-audit:verify packet=field/serverbound/FieldChange version=gms_v84 ida=0x53c5b9
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
