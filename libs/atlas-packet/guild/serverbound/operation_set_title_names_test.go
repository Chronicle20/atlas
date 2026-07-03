package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CField::SendSetGradeNameMsg: COutPacket(GUILD_OPERATION)+Encode1(0xD=SET_TITLE_NAMES)+5×EncodeStr(title).
// Body = 5×EncodeStr. IDA-verified: v83@0x530e1e, v84@0x53d097, v87@0x558638, v95@0x534fe0.
// v48 CField::SendSetGradeNameMsg @0x4c624a (sub_4C624A): COutPacket(96=GUILD_OPERATION)+Encode1(0xD=SET_TITLE_NAMES)+5xEncodeStr(title). Body=5xEncodeStr, == v83.
// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=gms_v48 ida=0x4c624a
// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=gms_v79 ida=0x51c411
// v72 CField::SendSetGradeNameMsg @0x515372: COutPacket(124)+Encode1(0xD=SET_TITLE_NAMES)
// +5×EncodeStr(title). Body = 5×EncodeStr, == v79.
// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=gms_v72 ida=0x515372
// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=gms_v95 ida=0x534fe0
// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=jms_v185 ida=ABSENT
// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=gms_v83 ida=0x530e1e
// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=gms_v84 ida=0x53d097
// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=gms_v87 ida=0x558638
// v61 COutPacket(114)+Encode1(13=SET_GRADE_NAME)+5xEncodeStr; body=5xEncodeStr, == v72/v83 (CField::SendSetGradeNameMsg @0x4e99fb).
// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=gms_v61 ida=0x4e99fb
func TestSetTitleNamesRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SetTitleNames{titles: []string{"Master", "Jr. Master", "Member", "Rookie", "Intern"}}
			output := SetTitleNames{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Titles()) != len(input.Titles()) {
				t.Fatalf("titles length: got %v, want %v", len(output.Titles()), len(input.Titles()))
			}
			for i, title := range output.Titles() {
				if title != input.Titles()[i] {
					t.Errorf("titles[%d]: got %v, want %v", i, title, input.Titles()[i])
				}
			}
		})
	}
}
