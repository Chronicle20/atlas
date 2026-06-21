package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// JOIN sent from CUIFadeYesNo::OnButtonClicked case 8 (guild-invite YES): COutPacket(GUILD_OPERATION)
// +Encode1(6=JOIN)+Encode4(guildId)+Encode4(characterId). Body = Encode4(guildId)+Encode4(characterId).
// IDA-verified (OnButtonClicked fn entry, case 8 body): v83@0x522585, v84@0x52dc20, v87@0x548098, v95@0x529c60.
// packet-audit:verify packet=guild/serverbound/GuildJoin version=gms_v95 ida=0x529c60
// packet-audit:verify packet=guild/serverbound/GuildJoin version=jms_v185 ida=0x5599d6
// packet-audit:verify packet=guild/serverbound/GuildJoin version=gms_v83 ida=0x522585
// packet-audit:verify packet=guild/serverbound/GuildJoin version=gms_v84 ida=0x52dc20
// packet-audit:verify packet=guild/serverbound/GuildJoin version=gms_v87 ida=0x548098
func TestJoinRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Join{guildId: 100, characterId: 200}
			output := Join{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.GuildId() != input.GuildId() {
				t.Errorf("guildId: got %v, want %v", output.GuildId(), input.GuildId())
			}
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
		})
	}
}
