package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldGuildBossPulleyStateChange version=gms_v79 ida=0x54198b
// packet-audit:verify packet=field/clientbound/FieldGuildBossPulleyStateChange version=gms_v83 ida=0x558c3b
// packet-audit:verify packet=field/clientbound/FieldGuildBossPulleyStateChange version=gms_v84 ida=0x5656de
// packet-audit:verify packet=field/clientbound/FieldGuildBossPulleyStateChange version=gms_v87 ida=0x583295
// packet-audit:verify packet=field/clientbound/FieldGuildBossPulleyStateChange version=gms_v95 ida=0x551540
// packet-audit:verify packet=field/clientbound/FieldGuildBossPulleyStateChange version=jms_v185 ida=0x59f97b
func TestGuildBossPulleyStateChangeGolden(t *testing.T) {
	input := NewGuildBossPulleyStateChange(0x01)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestGuildBossPulleyStateChangeByteOutputV79 pins the gms_v79
// FIELD_GUILD_BOSS_PULLEY_STATE_CHANGE clientbound read. IDA:
// CField_GuildBoss::OnPulleyStateChange @0x54198b (GMS_v79_1_DEVM.exe). Body
// is byte-identical to the v83 golden.
func TestGuildBossPulleyStateChangeByteOutputV79(t *testing.T) {
	input := NewGuildBossPulleyStateChange(0x01)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{0x01}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}

func TestGuildBossPulleyStateChangeRoundTrip(t *testing.T) {
	input := NewGuildBossPulleyStateChange(0x01)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
