package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldEffectBossHp version=gms_v83 ida=0x5330f7
// packet-audit:verify packet=field/clientbound/FieldEffectBossHp version=gms_v87 ida=0x55aac5
// packet-audit:verify packet=field/clientbound/FieldEffectBossHp version=gms_v95 ida=0x53b9c1
// packet-audit:verify packet=field/clientbound/FieldEffectBossHp version=jms_v185 ida=0x570359
// packet-audit:verify packet=field/clientbound/FieldEffectSummon version=gms_v83 ida=0x5330f7
// packet-audit:verify packet=field/clientbound/FieldEffectTremble version=gms_v83 ida=0x5330f7
// packet-audit:verify packet=field/clientbound/FieldEffectString version=gms_v83 ida=0x5330f7
// packet-audit:verify packet=field/clientbound/FieldEffectRewardRullet version=gms_v83 ida=0x5330f7
// packet-audit:verify packet=field/clientbound/FieldEffectSummon version=gms_v87 ida=0x55a948
// packet-audit:verify packet=field/clientbound/FieldEffectTremble version=gms_v87 ida=0x55abbb
// packet-audit:verify packet=field/clientbound/FieldEffectString version=gms_v87 ida=0x55a9fb
// packet-audit:verify packet=field/clientbound/FieldEffectRewardRullet version=gms_v87 ida=0x55abea
// packet-audit:verify packet=field/clientbound/FieldEffectSummon version=gms_v95 ida=0x53b790
// packet-audit:verify packet=field/clientbound/FieldEffectTremble version=gms_v95 ida=0x53bb74
// packet-audit:verify packet=field/clientbound/FieldEffectString version=gms_v95 ida=0x53b8b3
// packet-audit:verify packet=field/clientbound/FieldEffectRewardRullet version=gms_v95 ida=0x53bba4
// packet-audit:verify packet=field/clientbound/FieldEffectSummon version=jms_v185 ida=0x570359
// packet-audit:verify packet=field/clientbound/FieldEffectTremble version=jms_v185 ida=0x570359
// packet-audit:verify packet=field/clientbound/FieldEffectString version=jms_v185 ida=0x570359
// packet-audit:verify packet=field/clientbound/FieldEffectRewardRullet version=jms_v185 ida=0x570359
// packet-audit:verify packet=field/clientbound/FieldEffectBossHp version=gms_v84 ida=0x53f37d
// packet-audit:verify packet=field/clientbound/FieldEffectRewardRullet version=gms_v84 ida=0x53f37d
// packet-audit:verify packet=field/clientbound/FieldEffectSummon version=gms_v84 ida=0x53f37d
// packet-audit:verify packet=field/clientbound/FieldEffectTremble version=gms_v84 ida=0x53f37d
// packet-audit:verify packet=field/clientbound/FieldEffectString version=gms_v84 ida=0x53f37d
func TestFieldEffectSummon(t *testing.T) {
	input := NewFieldEffectSummon(0, 3, 100, 200)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestFieldEffectTremble(t *testing.T) {
	input := NewFieldEffectTremble(1, true, 500)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestFieldEffectObject(t *testing.T) {
	input := NewFieldEffectObject(2, "Map/Effect.img/dojang/start")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestFieldEffectBossHp(t *testing.T) {
	input := NewFieldEffectBossHp(5, 8500003, 50000, 100000, 6, 1)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestFieldEffectRewardRullet(t *testing.T) {
	input := NewFieldEffectRewardRullet(7, 1, 2, 3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
