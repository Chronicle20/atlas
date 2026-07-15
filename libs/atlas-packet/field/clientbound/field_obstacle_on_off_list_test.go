package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=gms_v83 ida=0x533057
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=gms_v84 ida=0x53f2dd
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=gms_v87 ida=0x55a870
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=gms_v95 ida=0x535b00
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=jms_v185 ida=0x5702b9
func TestFieldObstacleOnOffListGolden(t *testing.T) {
	input := NewFieldObstacleOnOffList([]ObstacleState{NewObstacleState("x", 0x00000005)})
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x78, 0x05, 0x00, 0x00, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestFieldObstacleOnOffListByteOutputV48 pins the gms_v48 FIELD_OBSTACLE_ONOFF_LIST
// (op 0x55 = 85) clientbound wire — the legacy SINGLE-obstacle shape. IDA:
// CField::OnFieldObstacleOnOffStatus = sub_4C930A @0x4c930a (GMS_v48_1_DEVM.exe):
// Decode1(flag) @0x4c9328 + Decode4(itemId) @0x4c932e, then DecodeStr(name)
// @0x4c9558 only when itemId!=0 (GetItemInfo block) and flag==0. The flag!=0 case
// carries no name (unambiguous byte trace); the flag==0+itemId!=0 case appends the
// obstacle name.
// packet-audit:verify packet=field/clientbound/FieldFieldObstacleOnOffList version=gms_v48 ida=0x4c930a
func TestFieldObstacleOnOffListByteOutputV48(t *testing.T) {
	ctx := test.CreateContext("GMS", 48, 1)
	// flag=1 (no name): flag(1) + itemId(4 LE)
	noName := NewFieldObstacleLegacy(0x01, 0x00001234, "")
	if got := test.Encode(t, ctx, noName.Encode, nil); !bytes.Equal(got, []byte{0x01, 0x34, 0x12, 0x00, 0x00}) {
		t.Errorf("v48 obstacle (flag=1) golden mismatch: got %v", got)
	}
	// flag=0, itemId!=0 (name present): flag(1) + itemId(4 LE) + Str(name)
	named := NewFieldObstacleLegacy(0x00, 0x00001234, "rt")
	if got := test.Encode(t, ctx, named.Encode, nil); !bytes.Equal(got, []byte{0x00, 0x34, 0x12, 0x00, 0x00, 0x02, 0x00, 0x72, 0x74}) {
		t.Errorf("v48 obstacle (flag=0) golden mismatch: got %v", got)
	}
}

func TestFieldObstacleOnOffListLegacyRoundTripV48(t *testing.T) {
	ctx := test.CreateContext("GMS", 48, 1)
	in := NewFieldObstacleLegacy(0x00, 0x00001234, "rt")
	out := FieldObstacleOnOffList{}
	test.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	if out.LegacyFlag() != in.LegacyFlag() || out.LegacyItemId() != in.LegacyItemId() || out.LegacyName() != in.LegacyName() {
		t.Errorf("legacy round-trip mismatch: got flag=%d id=%d name=%q", out.LegacyFlag(), out.LegacyItemId(), out.LegacyName())
	}
}

func TestFieldObstacleOnOffListRoundTrip(t *testing.T) {
	input := NewFieldObstacleOnOffList([]ObstacleState{
		NewObstacleState("x", 0x00000005),
		NewObstacleState("obstacle2", 0x00000000),
	})
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
