package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldGuildBoss version=gms_v79 ida=0x541895
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

// TestGuildBossByteOutputV79 pins the gms_v79 GUILD_BOSS (op 0xCF) serverbound
// wire. IDA: CField_GuildBoss::BasicActionAttack @0x541895 (GMS_v79_1_DEVM.exe) —
// COutPacket(207) @0x54191c then SendPacket with NO Encode* calls: empty body.
func TestGuildBossByteOutputV79(t *testing.T) {
	input := NewGuildBoss()
	ctx := pt.CreateContext("GMS", 79, 1)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("v79 guild_boss golden mismatch: got %v want empty", actual)
	}
}

// TestGuildBossByteOutputV72 pins the gms_v72 GUILD_BOSS (op 0xCD = 205)
// serverbound wire. IDA: CField_GuildBoss::BasicActionAttack @0x531d91
// (GMS_v72.1_U_DEVM.exe) — COutPacket(205) @0x531e18 then SendPacket @0x531e2b
// with NO Encode* calls: empty body (header only) — identical to the v79 golden
// (op 207).
// packet-audit:verify packet=field/serverbound/FieldGuildBoss version=gms_v72 ida=0x531d91
func TestGuildBossByteOutputV72(t *testing.T) {
	input := NewGuildBoss()
	ctx := pt.CreateContext("GMS", 72, 1)
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("v72 guild_boss golden mismatch: got %v want empty", actual)
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
