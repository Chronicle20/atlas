package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldGuildBossHealerMove version=gms_v83 ida=0x558c0c
// packet-audit:verify packet=field/clientbound/FieldGuildBossHealerMove version=gms_v84 ida=0x5656af
// packet-audit:verify packet=field/clientbound/FieldGuildBossHealerMove version=gms_v87 ida=0x583266
// packet-audit:verify packet=field/clientbound/FieldGuildBossHealerMove version=gms_v95 ida=0x551510
// packet-audit:verify packet=field/clientbound/FieldGuildBossHealerMove version=jms_v185 ida=0x59f94c
func TestGuildBossHealerMoveGolden(t *testing.T) {
	input := NewGuildBossHealerMove(0x0007)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x07, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestGuildBossHealerMoveRoundTrip(t *testing.T) {
	input := NewGuildBossHealerMove(0x0007)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
