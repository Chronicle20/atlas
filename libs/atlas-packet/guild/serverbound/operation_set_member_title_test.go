package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CField::SendSetMemberGradeMsg: COutPacket(GUILD_OPERATION)+Encode1(0xE=SET_MEMBER_TITLE)+Encode4(targetId)+Encode1(newTitle).
// Body = Encode4(targetId)+Encode1(newTitle). IDA-verified: v83@0x530dba, v84@0x53d030, v87@0x5585d1, v95@0x52d820, jms@0x56e1aa.
// packet-audit:verify packet=guild/serverbound/GuildSetMemberTitle version=gms_v79 ida=0x51c3ad
// v72 CField::SendSetMemberGradeMsg @0x51530e: COutPacket(124)+Encode1(0xE=SET_MEMBER_TITLE)
// +Encode4(targetId)+Encode1(newTitle). Body = Encode4(targetId)+Encode1(newTitle), == v79.
// packet-audit:verify packet=guild/serverbound/GuildSetMemberTitle version=gms_v72 ida=0x51530e
// packet-audit:verify packet=guild/serverbound/GuildSetMemberTitle version=jms_v185 ida=0x56e1aa
// packet-audit:verify packet=guild/serverbound/GuildSetMemberTitle version=gms_v95 ida=0x52d820
// packet-audit:verify packet=guild/serverbound/GuildSetMemberTitle version=gms_v83 ida=0x530dba
// packet-audit:verify packet=guild/serverbound/GuildSetMemberTitle version=gms_v84 ida=0x53d030
// packet-audit:verify packet=guild/serverbound/GuildSetMemberTitle version=gms_v87 ida=0x5585d1
// v61 COutPacket(114)+Encode1(14=SET_MEMBER_GRADE)+Encode4(cid)+Encode1(title); body=Encode4+Encode1, == v72/v83 (CField::SendSetMemberGradeMsg @0x4e9995).
// packet-audit:verify packet=guild/serverbound/GuildSetMemberTitle version=gms_v61 ida=0x4e9995
func TestSetMemberTitleRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SetMemberTitle{targetId: 54321, newTitle: 2}
			output := SetMemberTitle{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TargetId() != input.TargetId() {
				t.Errorf("targetId: got %v, want %v", output.TargetId(), input.TargetId())
			}
			if output.NewTitle() != input.NewTitle() {
				t.Errorf("newTitle: got %v, want %v", output.NewTitle(), input.NewTitle())
			}
		})
	}
}
