package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildJoin version=gms_v95 ida=0x0
// packet-audit:verify packet=guild/serverbound/GuildJoin version=jms_v185 ida=ABSENT
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
