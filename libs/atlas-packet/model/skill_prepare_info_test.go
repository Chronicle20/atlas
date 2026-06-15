package model

import (
	"encoding/binary"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// sampleSkillPrepareInfo builds a representative serverbound skill prepare request.
// Uses Hurricane (3121004) — a standard keydown skill — so the swallowMobId branch
// (skillId == 33101005) stays quiet.
func sampleSkillPrepareInfo() *SkillPrepareInfo {
	m := NewSkillPrepareInfo()
	m.SetSkillId(3121004) // Hurricane — keydown, no swallowMobId on any version
	m.SetLevel(10)
	m.SetAction(0x0142) // (oneTimeAction & 0x7FFF) | (moveAction << 15)
	m.SetActionSpeed(4)
	return m
}

// sampleSkillPrepareInfoSwallow builds a prepare request for Dragon Knight swallow
// (skillId 33101005), which triggers the swallowMobId field on v95/jms185.
func sampleSkillPrepareInfoSwallow() *SkillPrepareInfo {
	m := NewSkillPrepareInfo()
	m.SetSkillId(33101005) // Dragon Knight swallow — writes swallowMobId on v95/jms185
	m.SetLevel(1)
	m.SetAction(0x0020)
	m.SetActionSpeed(6)
	m.SetSwallowMobId(9876543)
	return m
}

// sampleSkillCancelInfo builds a representative serverbound skill cancel request.
func sampleSkillCancelInfo() *SkillCancelInfo {
	m := NewSkillCancelInfo()
	m.SetSkillId(3121004)
	return m
}

// TestSkillPrepareInfoRoundTrip exercises Encode/Decode symmetry for SkillPrepareInfo
// across all tenant variants (standard keydown skill — swallowMobId branch inactive).
func TestSkillPrepareInfoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := sampleSkillPrepareInfo()
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}

// TestSkillPrepareInfoSwallowRoundTrip exercises the swallowMobId conditional branch
// (skillId == 33101005). On v95 and jms185 the field must be written and consumed;
// on all other versions the field must be absent (no unconsumed bytes).
func TestSkillPrepareInfoSwallowRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := sampleSkillPrepareInfoSwallow()
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}

// TestSkillCancelInfoRoundTrip exercises Encode/Decode symmetry for SkillCancelInfo
// across all tenant variants. The body is a single u32 on every version.
func TestSkillCancelInfoRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			m := sampleSkillCancelInfo()
			pt.RoundTrip(t, ctx, m.Encode, m.Decode, nil)
		})
	}
}

// TestSkillPrepareInfoByteFixture asserts the exact encoded bytes match the wire-spec
// field order: skillId u32 LE, level u8, action u16 LE, actionSpeed u8 [, swallowMobId u32 LE].
// Covers one GMS pre-v95 variant (v83), GMS v95, and JMS v185 to exercise the
// swallowMobId branch delta.
//
// packet-audit:verify
func TestSkillPrepareInfoByteFixture(t *testing.T) {
	// Fields for the standard case (Hurricane 3121004, no swallowMobId):
	//   skillId=3121004 (0x002F9F6C LE = 6C 9F 2F 00), level=10, action=0x0142, actionSpeed=4
	// Expected encoding (all LE):
	//   6C 9F 2F 00  0A  42 01  04
	standardExpected := []byte{
		0x6C, 0x9F, 0x2F, 0x00, // skillId=3121004 LE
		0x0A,       // level=10
		0x42, 0x01, // action=0x0142 LE
		0x04,       // actionSpeed=4
	}

	// Fields for the swallow case (skillId=33101005, swallowMobId=9876543):
	//   skillId=33101005 (0x01F914CD LE = CD 14 F9 01), level=1, action=0x0020, actionSpeed=6
	//   swallowMobId=9876543 (0x0096B43F LE = 3F B4 96 00)
	// Expected base (all LE):
	//   CD 14 F9 01  01  20 00  06
	// Plus swallowMobId on v95/jms185:
	//   3F B4 96 00
	swallowBase := []byte{
		0xCD, 0x14, 0xF9, 0x01, // skillId=33101005 LE
		0x01,       // level=1
		0x20, 0x00, // action=0x0020 LE
		0x06,       // actionSpeed=6
	}
	var swallowMobIdBytes [4]byte
	binary.LittleEndian.PutUint32(swallowMobIdBytes[:], 9876543)
	swallowWithMobId := append(append([]byte(nil), swallowBase...), swallowMobIdBytes[:]...)

	cases := []struct {
		name     string
		region   string
		major    uint16
		m        *SkillPrepareInfo
		expected []byte
	}{
		// GMS v83: standard fields, no swallowMobId even for 33101005
		{
			name:     "GMS v83 standard (no swallowMobId)",
			region:   "GMS",
			major:    83,
			m:        sampleSkillPrepareInfo(),
			expected: standardExpected,
		},
		// GMS v83 with swallow skill — no swallowMobId field on v83
		{
			name:     "GMS v83 swallow skill (no swallowMobId field)",
			region:   "GMS",
			major:    83,
			m:        sampleSkillPrepareInfoSwallow(),
			expected: swallowBase,
		},
		// GMS v95: swallow skill triggers swallowMobId field
		{
			name:     "GMS v95 swallow skill (with swallowMobId field)",
			region:   "GMS",
			major:    95,
			m:        sampleSkillPrepareInfoSwallow(),
			expected: swallowWithMobId,
		},
		// GMS v95: standard skill still has no swallowMobId
		{
			name:     "GMS v95 standard (no swallowMobId)",
			region:   "GMS",
			major:    95,
			m:        sampleSkillPrepareInfo(),
			expected: standardExpected,
		},
		// JMS v185: swallow skill triggers swallowMobId field
		{
			name:     "JMS v185 swallow skill (with swallowMobId field)",
			region:   "JMS",
			major:    185,
			m:        sampleSkillPrepareInfoSwallow(),
			expected: swallowWithMobId,
		},
		// JMS v185: standard skill still has no swallowMobId
		{
			name:     "JMS v185 standard (no swallowMobId)",
			region:   "JMS",
			major:    185,
			m:        sampleSkillPrepareInfo(),
			expected: standardExpected,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.region, tc.major, 1)
			got := pt.Encode(t, ctx, tc.m.Encode, nil)
			if len(got) != len(tc.expected) {
				t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X",
					len(got), len(tc.expected), got, tc.expected)
			}
			for i := range tc.expected {
				if got[i] != tc.expected[i] {
					t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X",
						i, got[i], tc.expected[i], got, tc.expected)
					break
				}
			}
		})
	}
}

// TestSkillCancelInfoByteFixture asserts the exact encoded bytes for SkillCancelInfo.
// The body is always skillId u32 LE, identical on every version.
//
// packet-audit:verify
func TestSkillCancelInfoByteFixture(t *testing.T) {
	// skillId=3121004 (0x002F9F6C LE = 6C 9F 2F 00)
	expected := []byte{0x6C, 0x9F, 0x2F, 0x00}

	versions := []struct {
		name   string
		region string
		major  uint16
	}{
		{"GMS v83", "GMS", 83},
		{"GMS v95", "GMS", 95},
		{"JMS v185", "JMS", 185},
	}

	for _, v := range versions {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, 1)
			m := sampleSkillCancelInfo()
			got := pt.Encode(t, ctx, m.Encode, nil)
			if len(got) != len(expected) {
				t.Fatalf("byte length mismatch: got %d want %d; got: %X", len(got), len(expected), got)
			}
			for i := range expected {
				if got[i] != expected[i] {
					t.Errorf("byte[%d] = %02X, want %02X; got: %X", i, got[i], expected[i], got)
					break
				}
			}
		})
	}
}
