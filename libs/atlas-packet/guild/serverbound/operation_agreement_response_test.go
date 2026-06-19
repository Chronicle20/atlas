package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildAgreementResponse version=gms_v95 ida=0x52d780
// packet-audit:verify packet=guild/serverbound/GuildAgreementResponse version=jms_v185 ida=0x56da47
// packet-audit:verify packet=guild/serverbound/GuildAgreementResponse version=gms_v87 ida=0x557e6e
// packet-audit:verify packet=guild/serverbound/GuildAgreementResponse version=gms_v83 ida=0x530666
// v84 SendCreateGuildAgreeMsg @0x53c8cd: COutPacket(0x82)+Encode1(0x1E)+Encode4(guildId)+Encode1(agreed) (IDA-verified).
// packet-audit:verify packet=guild/serverbound/GuildAgreementResponse version=gms_v84 ida=0x53c8cd
func TestAgreementResponseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AgreementResponse{unk: 42, agreed: true}
			output := AgreementResponse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Unk() != input.Unk() {
				t.Errorf("unk: got %v, want %v", output.Unk(), input.Unk())
			}
			if output.Agreed() != input.Agreed() {
				t.Errorf("agreed: got %v, want %v", output.Agreed(), input.Agreed())
			}
		})
	}
}
