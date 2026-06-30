package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldEffectBossHp version=gms_v79 ida=0x51e577
// packet-audit:verify packet=field/clientbound/FieldEffectSummon version=gms_v79 ida=0x51e577
// packet-audit:verify packet=field/clientbound/FieldEffectTremble version=gms_v79 ida=0x51e577
// packet-audit:verify packet=field/clientbound/FieldEffectString version=gms_v79 ida=0x51e577
// packet-audit:verify packet=field/clientbound/FieldEffectRewardRullet version=gms_v79 ida=0x51e577
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
// TestFieldEffectByteOutputV79 pins every gms_v79 FIELD_EFFECT (op 0x82)
// dispatcher sub-mode so the op-cell lifts off worst-of-siblings. IDA:
// CField::OnFieldEffect @0x51e577 (GMS_v79_1_DEVM.exe) switches on Decode1(mode)
// @0x51e593; each arm matches the codec field-for-field (addresses cited inline).
func TestFieldEffectByteOutputV79(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)

	// Summon (case 0): Decode1(effect)@0x51e5af + Decode4(x)@0x51e5b9 + Decode4(y)@0x51e5c3.
	summon := NewFieldEffectSummon(0, 3, 100, 200)
	if got := test.Encode(t, ctx, summon.Encode, nil); !bytes.Equal(got, []byte{
		0x00, 0x03, 0x64, 0x00, 0x00, 0x00, 0xC8, 0x00, 0x00, 0x00,
	}) {
		t.Errorf("v79 summon: got %v", got)
	}

	// Tremble (case 1): Decode1(bHeavy)@0x51e861 + Decode4(delay)@0x51e864.
	tremble := NewFieldEffectTremble(1, true, 500)
	if got := test.Encode(t, ctx, tremble.Encode, nil); !bytes.Equal(got, []byte{
		0x01, 0x01, 0xF4, 0x01, 0x00, 0x00,
	}) {
		t.Errorf("v79 tremble: got %v", got)
	}

	// String (case 2/3/4/6): Decode1(mode) + DecodeStr(name)@0x51e67c.
	str := NewFieldEffectObject(2, "x")
	if got := test.Encode(t, ctx, str.Encode, nil); !bytes.Equal(got, []byte{
		0x02, 0x01, 0x00, 'x',
	}) {
		t.Errorf("v79 string: got %v", got)
	}

	// BossHp (case 5): Decode4(monsterId)@0x51e759 + Decode4(curHp)@0x51e763 +
	// Decode4(maxHp)@0x51e76d + Decode1(tagColor)@0x51e779 + Decode1(tagBg)@0x51e77b.
	bossHp := NewFieldEffectBossHp(5, 8500003, 50000, 100000, 6, 1)
	if got := test.Encode(t, ctx, bossHp.Encode, nil); !bytes.Equal(got, []byte{
		0x05, 0x23, 0xB3, 0x81, 0x00, 0x50, 0xC3, 0x00, 0x00, 0xA0, 0x86, 0x01, 0x00, 0x06, 0x01,
	}) {
		t.Errorf("v79 bossHp: got %v", got)
	}

	// RewardRullet (case 7): Decode4@0x51e890 + Decode4@0x51e899 + Decode4@0x51e89b.
	rullet := NewFieldEffectRewardRullet(7, 1, 2, 3)
	if got := test.Encode(t, ctx, rullet.Encode, nil); !bytes.Equal(got, []byte{
		0x07, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
	}) {
		t.Errorf("v79 rewardRullet: got %v", got)
	}
}

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
