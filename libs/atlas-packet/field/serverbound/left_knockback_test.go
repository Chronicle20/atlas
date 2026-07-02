package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v79 ida=0x55230d
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v83 ida=0x574df1
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v84 ida=0x58476f
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v87 ida=0x5a307b
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v95 ida=0x5612d0
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=jms_v185 ida=0x5c92fb
func TestLeftKnockbackGolden(t *testing.T) {
	input := NewLeftKnockback()
	ctx := pt.CreateContext("GMS", 83, 1)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

// TestLeftKnockbackByteOutputV79 pins the gms_v79 LEFT_KNOCKBACK (op 0xCC)
// serverbound wire. IDA: CField_SnowBall::Update @0x55230d (GMS_v79_1_DEVM.exe) —
// COutPacket(204) @0x552388 then SendPacket with NO Encode* calls: empty body.
func TestLeftKnockbackByteOutputV79(t *testing.T) {
	input := NewLeftKnockback()
	ctx := pt.CreateContext("GMS", 79, 1)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("v79 left_knockback golden mismatch: got %v want empty", actual)
	}
}

// TestLeftKnockbackByteOutputV72 pins the gms_v72 LEFT_KNOCKBACK (op 0xCA = 202)
// serverbound wire. IDA: CField_SnowBall::Update @0x53ff36
// (GMS_v72.1_U_DEVM.exe) — COutPacket(202) @0x53ffb1 then SendPacket @0x53ffc4
// with NO Encode* calls: empty body (header only) — identical to the v79 golden
// (op 204).
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v72 ida=0x53ff36
func TestLeftKnockbackByteOutputV72(t *testing.T) {
	input := NewLeftKnockback()
	ctx := pt.CreateContext("GMS", 72, 1)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("v72 left_knockback golden mismatch: got %v want empty", actual)
	}
}

// TestLeftKnockbackByteOutputV61 pins the gms_v61 LEFT_KNOCKBACK (op 0xB1 = 177)
// serverbound wire. IDA: CField_SnowBall::Update @0x50bb50 (GMS_v61.1_U_DEVM.exe) —
// COutPacket(177) then SendPacket with NO Encode* calls: empty body (header only)
// — identical to the v72 golden (op 202).
// packet-audit:verify packet=field/serverbound/FieldLeftKnockback version=gms_v61 ida=0x50bb50
func TestLeftKnockbackByteOutputV61(t *testing.T) {
	input := NewLeftKnockback()
	ctx := pt.CreateContext("GMS", 61, 1)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("v61 left_knockback golden mismatch: got %v want empty", actual)
	}
}

func TestLeftKnockbackRoundTrip(t *testing.T) {
	input := NewLeftKnockback()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := LeftKnockback{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
