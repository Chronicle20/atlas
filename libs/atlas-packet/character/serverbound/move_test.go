package serverbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestCharacterMove(t *testing.T) {
	p := Move{}
	p.dr0 = 100
	p.dr1 = 200
	p.fieldKey = 42
	p.dr2 = 300
	p.dr3 = 400
	p.crc = 500
	p.dwKey = 600
	p.crc32 = 700

	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, p.Encode, p.Decode, nil)

			if p.FieldKey() != 42 {
				t.Errorf("expected fieldKey 42, got %d", p.FieldKey())
			}
			// IDA JMS v185 CVecCtrlUser::EndUpdateActive@0xaaa076: no dr0/dr1/dr2/dr3/dwKey/crc32.
			// JMS movement is GMS v83-equivalent (fieldKey only before CMovePath).
			if v.Region == "GMS" && v.MajorVersion > 83 {
				if p.Dr0() != 100 {
					t.Errorf("expected dr0 100, got %d", p.Dr0())
				}
				if p.Dr1() != 200 {
					t.Errorf("expected dr1 200, got %d", p.Dr1())
				}
				if p.Dr2() != 300 {
					t.Errorf("expected dr2 300, got %d", p.Dr2())
				}
				if p.Dr3() != 400 {
					t.Errorf("expected dr3 400, got %d", p.Dr3())
				}
				if p.DwKey() != 600 {
					t.Errorf("expected dwKey 600, got %d", p.DwKey())
				}
				if p.Crc32() != 700 {
					t.Errorf("expected crc32 700, got %d", p.Crc32())
				}
			}
			if v.Region == "GMS" && v.MajorVersion > 28 && p.Crc() != 500 {
				t.Errorf("expected crc 500, got %d", p.Crc())
			}
		})
	}
}

func TestCharacterMoveOperationString(t *testing.T) {
	p := Move{}
	if p.Operation() != CharacterMoveHandle {
		t.Errorf("expected operation %s, got %s", CharacterMoveHandle, p.Operation())
	}
	if p.String() == "" {
		t.Error("expected non-empty string")
	}
}
