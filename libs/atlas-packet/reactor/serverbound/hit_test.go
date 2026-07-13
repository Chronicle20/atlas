package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// hasIsSkill / hasSkillId mirror the version gates in hit.go: isSkill is on the
// wire from GMS v72+ (and JMS), the trailing skillId from GMS v79+ (and JMS).
// IDA-confirmed CReactorPool::FindHitReactor send layouts: v48 @0x5a5d1a and v61
// @0x633ac7 = oid+dwHitOption+delay (3 fields); v72 @0x6928bc adds isSkill;
// v79 @0x6b8077 and v83 @0x7356c7 add skillId.
func hasIsSkill(region string, major uint16) bool {
	return (region == "GMS" && major >= 72) || region == "JMS"
}

func hasSkillId(region string, major uint16) bool {
	return (region == "GMS" && major >= 79) || region == "JMS"
}

// packet-audit:verify packet=reactor/serverbound/ReactorHitRequest version=gms_v83 ida=0x7356c7
// packet-audit:verify packet=reactor/serverbound/ReactorHitRequest version=gms_v87 ida=0x77b5eb
// packet-audit:verify packet=reactor/serverbound/ReactorHitRequest version=gms_v95 ida=0x6cd4e0
// packet-audit:verify packet=reactor/serverbound/ReactorHitRequest version=jms_v185 ida=0x79ea6a
// packet-audit:verify packet=reactor/serverbound/ReactorHitRequest version=gms_v84 ida=0x752cbc
func TestHitRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := HitRequest{oid: 100, isSkill: true, dwHitOption: 3, delay: 50, skillId: 1001004}
			output := HitRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Oid() != input.Oid() {
				t.Errorf("oid: got %v, want %v", output.Oid(), input.Oid())
			}
			// isSkill / skillId are version-gated (see hit.go); legacy variants
			// (GMS v28 here) do not carry them on the wire, so they round-trip to
			// the zero value rather than the input value.
			if hasIsSkill(v.Region, v.MajorVersion) {
				if output.IsSkill() != input.IsSkill() {
					t.Errorf("isSkill: got %v, want %v", output.IsSkill(), input.IsSkill())
				}
			} else if output.IsSkill() {
				t.Errorf("isSkill: legacy variant carried it on the wire, want false")
			}
			if output.DwHitOption() != input.DwHitOption() {
				t.Errorf("dwHitOption: got %v, want %v", output.DwHitOption(), input.DwHitOption())
			}
			if output.Delay() != input.Delay() {
				t.Errorf("delay: got %v, want %v", output.Delay(), input.Delay())
			}
			if hasSkillId(v.Region, v.MajorVersion) {
				if output.SkillId() != input.SkillId() {
					t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
				}
			} else if output.SkillId() != 0 {
				t.Errorf("skillId: legacy variant carried it on the wire, want 0")
			}
		})
	}
}

// TestHitBytesV48 pins the exact v48 DAMAGE_REACTOR (op 145) wire. The send site
// is CReactorPool::FindHitReactor @0x5a5a32; the COutPacket build block @0x5a5d1a:
//
//	COutPacket(145) @0x5a5d1a
//	Encode4 @0x5a5d2b — reactor oid (v41[0] = *v41)          -> oid
//	Encode4 @0x5a5d39 — hit-option flags (v41[13])           -> dwHitOption
//	Encode2 @0x5a5d44 — reactor stance/index (v47)           -> delay
//
// v48 (<72) writes NEITHER the isSkill Encode4 (added v72 @0x6928bc) NOR the
// trailing skillId Encode4 (added v79 @0x6b8077) — only three fields. Legacy
// gate applied in hit.go; v72+/JMS unchanged.
//
// packet-audit:verify packet=reactor/serverbound/ReactorHitRequest version=gms_v48 ida=0x5a5a32
func TestHitBytesV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	// isSkill/skillId set but gated off at v48:
	input := HitRequest{oid: 100, isSkill: true, dwHitOption: 3, delay: 50, skillId: 1001004}
	got := pt.Encode(t, ctx, input.Encode, nil)
	want := []byte{
		0x64, 0x00, 0x00, 0x00, // oid 100 (Encode4 @0x5a5d2b)
		0x03, 0x00, 0x00, 0x00, // dwHitOption 3 (Encode4 @0x5a5d39)
		0x32, 0x00, // delay 50 (Encode2 @0x5a5d44)
		// isSkill (v72+) and skillId (v79+) OMITTED at v48
	}
	if !bytes.Equal(got, want) {
		t.Errorf("v48 reactor hit bytes:\n got % x\nwant % x", got, want)
	}
}
