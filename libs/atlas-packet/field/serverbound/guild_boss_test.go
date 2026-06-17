package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldGuildBoss version=gms_v83 ida=0x558b45
// packet-audit:verify packet=field/serverbound/FieldGuildBoss version=gms_v84 ida=0x5655e8
// packet-audit:verify packet=field/serverbound/FieldGuildBoss version=gms_v87 ida=0x58319f
// packet-audit:verify packet=field/serverbound/FieldGuildBoss version=gms_v95 ida=0x5517d0
// packet-audit:verify packet=field/serverbound/FieldGuildBoss version=jms_v185 ida=0x59f885
func TestGuildBossGolden(t *testing.T) {
	input := NewGuildBoss()
	ctx := pt.CreateContext("GMS", 83, 1)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("golden mismatch: got %v want empty", actual)
	}
}

func TestGuildBossRoundTrip(t *testing.T) {
	input := NewGuildBoss()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := GuildBoss{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
