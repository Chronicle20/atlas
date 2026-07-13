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
// packet-audit:verify packet=field/clientbound/FieldEffectBossHp version=gms_v72 ida=0x5174bb
// packet-audit:verify packet=field/clientbound/FieldEffectSummon version=gms_v72 ida=0x5174bb
// packet-audit:verify packet=field/clientbound/FieldEffectTremble version=gms_v72 ida=0x5174bb
// packet-audit:verify packet=field/clientbound/FieldEffectString version=gms_v72 ida=0x5174bb
// packet-audit:verify packet=field/clientbound/FieldEffectRewardRullet version=gms_v72 ida=0x5174bb
// packet-audit:verify packet=field/clientbound/FieldEffectSummon version=gms_v61 ida=0x4eb523
// packet-audit:verify packet=field/clientbound/FieldEffectTremble version=gms_v61 ida=0x4eb523
// packet-audit:verify packet=field/clientbound/FieldEffectString version=gms_v61 ida=0x4eb523
// packet-audit:verify packet=field/clientbound/FieldEffectBossHp version=gms_v61 ida=0x4eb523
// packet-audit:verify packet=field/clientbound/FieldEffectSummon version=gms_v48 ida=0x4c7b59
// packet-audit:verify packet=field/clientbound/FieldEffectTremble version=gms_v48 ida=0x4c7b59
// packet-audit:verify packet=field/clientbound/FieldEffectString version=gms_v48 ida=0x4c7b59
// packet-audit:verify packet=field/clientbound/FieldEffectBossHp version=gms_v48 ida=0x4c7b59
//
// TestFieldEffectByteOutputV61 pins the gms_v61 FIELD_EFFECT (op 0x68 = 104)
// dispatcher sub-modes. IDA: CField::OnFieldEffect = sub_4EB523 @0x4eb523
// (GMS_v61.1_U_DEVM.exe) switches on Decode1(mode): mode 0 (Summon) =
// Decode1(effect)+Decode4(x)+Decode4(y); mode 1 (Tremble) = Decode1(bHeavy)+
// Decode4(delay); modes 2/3/4/6 (String/screen/sound/BGM) = DecodeStr(name);
// mode 5 (BossHp) = Decode4(monsterId)+Decode4(curHp)+Decode4(maxHp)+
// Decode1(tagColor)+Decode1(tagBg). The switch tops out at mode 6 (BGM) — there is
// NO reward-roulette (mode 7) arm, so FieldEffectRewardRullet is version-absent in
// v61 (dispositioned n-a). All present arms match the codec field-for-field and are
// byte-identical to the v72 golden (version-invariant layout).
func TestFieldEffectByteOutputV61(t *testing.T) {
	ctx := test.CreateContext("GMS", 61, 1)

	summon := NewFieldEffectSummon(0, 3, 100, 200)
	if got := test.Encode(t, ctx, summon.Encode, nil); !bytes.Equal(got, []byte{
		0x00, 0x03, 0x64, 0x00, 0x00, 0x00, 0xC8, 0x00, 0x00, 0x00,
	}) {
		t.Errorf("v61 summon: got %v", got)
	}

	tremble := NewFieldEffectTremble(1, true, 500)
	if got := test.Encode(t, ctx, tremble.Encode, nil); !bytes.Equal(got, []byte{
		0x01, 0x01, 0xF4, 0x01, 0x00, 0x00,
	}) {
		t.Errorf("v61 tremble: got %v", got)
	}

	str := NewFieldEffectObject(2, "x")
	if got := test.Encode(t, ctx, str.Encode, nil); !bytes.Equal(got, []byte{
		0x02, 0x01, 0x00, 'x',
	}) {
		t.Errorf("v61 string: got %v", got)
	}

	bossHp := NewFieldEffectBossHp(5, 8500003, 50000, 100000, 6, 1)
	if got := test.Encode(t, ctx, bossHp.Encode, nil); !bytes.Equal(got, []byte{
		0x05, 0x23, 0xB3, 0x81, 0x00, 0x50, 0xC3, 0x00, 0x00, 0xA0, 0x86, 0x01, 0x00, 0x06, 0x01,
	}) {
		t.Errorf("v61 bossHp: got %v", got)
	}
}

// TestFieldEffectByteOutputV48 pins every gms_v48 FIELD_EFFECT (op 0x54 = 84)
// dispatcher sub-mode. IDA: CField::OnFieldEffect = sub_4C7B59 @0x4c7b59
// (GMS_v48_1_DEVM.exe) switches on Decode1(mode) @0x4c7b71: mode 0 (Summon) =
// Decode1(effect)@0x4c7f1f + Decode4(x)@0x4c7f29 + Decode4(y)@0x4c7f31; mode 1
// (Tremble) = Decode1(bHeavy)@0x4c7eef + Decode4(delay)@0x4c7ef2; modes 2/3/4/6
// (String/screen/sound/BGM) = DecodeStr(name)@0x4c7eb0; mode 5 (BossHp) =
// Decode4(monsterId)@0x4c7c55 + Decode4(curHp)@0x4c7c5e + Decode4(maxHp)@0x4c7c68 +
// Decode1(tagColor)@0x4c7c74 + Decode1(tagBg)@0x4c7c76. The switch tops out at mode
// 6 (BGM) — there is NO reward-roulette (mode 7) arm, so FieldEffectRewardRullet is
// version-absent in v48 (dispositioned n-a, mirroring v61). All present arms match
// the codec field-for-field and are byte-identical to the v61 golden.
func TestFieldEffectByteOutputV48(t *testing.T) {
	ctx := test.CreateContext("GMS", 48, 1)

	summon := NewFieldEffectSummon(0, 3, 100, 200)
	if got := test.Encode(t, ctx, summon.Encode, nil); !bytes.Equal(got, []byte{
		0x00, 0x03, 0x64, 0x00, 0x00, 0x00, 0xC8, 0x00, 0x00, 0x00,
	}) {
		t.Errorf("v48 summon: got %v", got)
	}

	tremble := NewFieldEffectTremble(1, true, 500)
	if got := test.Encode(t, ctx, tremble.Encode, nil); !bytes.Equal(got, []byte{
		0x01, 0x01, 0xF4, 0x01, 0x00, 0x00,
	}) {
		t.Errorf("v48 tremble: got %v", got)
	}

	str := NewFieldEffectObject(2, "x")
	if got := test.Encode(t, ctx, str.Encode, nil); !bytes.Equal(got, []byte{
		0x02, 0x01, 0x00, 'x',
	}) {
		t.Errorf("v48 string: got %v", got)
	}

	bossHp := NewFieldEffectBossHp(5, 8500003, 50000, 100000, 6, 1)
	if got := test.Encode(t, ctx, bossHp.Encode, nil); !bytes.Equal(got, []byte{
		0x05, 0x23, 0xB3, 0x81, 0x00, 0x50, 0xC3, 0x00, 0x00, 0xA0, 0x86, 0x01, 0x00, 0x06, 0x01,
	}) {
		t.Errorf("v48 bossHp: got %v", got)
	}
}

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

// TestFieldEffectByteOutputV72 pins every gms_v72 FIELD_EFFECT (op 0x07E)
// dispatcher sub-mode so the op-cell lifts off worst-of-siblings. IDA:
// CField::OnFieldEffect @0x5174bb (GMS_v72.1_U_DEVM.exe) switches on Decode1(mode)
// @0x5174d7 (switch @0x5174e3); each v72 arm matches the codec field-for-field and
// is byte-identical to the v79 golden (addresses cited inline).
func TestFieldEffectByteOutputV72(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)

	// Summon (case 0): Decode1(effect)@0x5174f3 + Decode4(x)@0x5174fd + Decode4(y)@0x517507.
	summon := NewFieldEffectSummon(0, 3, 100, 200)
	if got := test.Encode(t, ctx, summon.Encode, nil); !bytes.Equal(got, []byte{
		0x00, 0x03, 0x64, 0x00, 0x00, 0x00, 0xC8, 0x00, 0x00, 0x00,
	}) {
		t.Errorf("v72 summon: got %v", got)
	}

	// Tremble (case 1): Decode1(bHeavy)@0x5177a5 + Decode4(delay)@0x5177a8.
	tremble := NewFieldEffectTremble(1, true, 500)
	if got := test.Encode(t, ctx, tremble.Encode, nil); !bytes.Equal(got, []byte{
		0x01, 0x01, 0xF4, 0x01, 0x00, 0x00,
	}) {
		t.Errorf("v72 tremble: got %v", got)
	}

	// String (case 2/3/4/6): Decode1(mode) + DecodeStr(name)@0x5175c0.
	str := NewFieldEffectObject(2, "x")
	if got := test.Encode(t, ctx, str.Encode, nil); !bytes.Equal(got, []byte{
		0x02, 0x01, 0x00, 'x',
	}) {
		t.Errorf("v72 string: got %v", got)
	}

	// BossHp (case 5): Decode4(monsterId)@0x51769d + Decode4(curHp)@0x5176a7 +
	// Decode4(maxHp)@0x5176b1 + Decode1(tagColor)@0x5176bd + Decode1(tagBg)@0x5176bf.
	bossHp := NewFieldEffectBossHp(5, 8500003, 50000, 100000, 6, 1)
	if got := test.Encode(t, ctx, bossHp.Encode, nil); !bytes.Equal(got, []byte{
		0x05, 0x23, 0xB3, 0x81, 0x00, 0x50, 0xC3, 0x00, 0x00, 0xA0, 0x86, 0x01, 0x00, 0x06, 0x01,
	}) {
		t.Errorf("v72 bossHp: got %v", got)
	}

	// RewardRullet (case 7): Decode4@0x5177d4 + Decode4@0x5177dd + Decode4@0x5177df.
	rullet := NewFieldEffectRewardRullet(7, 1, 2, 3)
	if got := test.Encode(t, ctx, rullet.Encode, nil); !bytes.Equal(got, []byte{
		0x07, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
	}) {
		t.Errorf("v72 rewardRullet: got %v", got)
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
