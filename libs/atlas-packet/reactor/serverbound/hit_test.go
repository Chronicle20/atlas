package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
			if output.IsSkill() != input.IsSkill() {
				t.Errorf("isSkill: got %v, want %v", output.IsSkill(), input.IsSkill())
			}
			if output.DwHitOption() != input.DwHitOption() {
				t.Errorf("dwHitOption: got %v, want %v", output.DwHitOption(), input.DwHitOption())
			}
			if output.Delay() != input.Delay() {
				t.Errorf("delay: got %v, want %v", output.Delay(), input.Delay())
			}
			if output.SkillId() != input.SkillId() {
				t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
			}
		})
	}
}
