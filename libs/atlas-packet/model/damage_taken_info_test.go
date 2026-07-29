package model

import (
	"testing"

	"github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// legacyLayout reports whether a variant takes the pre-v61 GMS wire layout
// (design §2a): no nMagicElemAttr byte and no characterX/characterY pair in
// the reflect extension. mobHit's shared fixture must be zeroed on those
// three fields for these variants so the equality assertions hold; the
// gates themselves (see Decode/Encode) are not weakened.
func legacyLayout(v pt.TenantVariant) bool {
	return v.Region == "GMS" && v.MajorVersion < 61
}

// mobHit builds a mob-sourced DamageTakenInfo. withExt controls the optional
// 14-byte reflect extension (present on the wire iff the client set
// bKnockback or nX — modeled by hasReflectExtension).
func mobHit(withExt bool, blockByte byte, reflect byte) DamageTakenInfo {
	m := DamageTakenInfo{
		characterId:         100,
		updateTime:          12345,
		nAttackIdx:          DamageTypePhysical,
		nMagicElemAttr:      DamageElementTypeFire,
		damage:              500,
		monsterTemplateId:   200100,
		monsterId:           42,
		left:                true,
		reflect:             reflect,
		guard:               true,
		blockByte:           blockByte,
		hasReflectExtension: withExt,
		stanceFlags:         1,
	}
	if withExt {
		m.isPowerGuard = true
		m.reflectTargetMobId = 42
		m.hitAction = 3
		m.hitX = 100
		m.hitY = 200
		m.characterX = 110
		m.characterY = 210
	}
	return m
}

// mobHitForVariant is mobHit adjusted for the target variant's wire layout:
// pre-v61 GMS variants carry neither nMagicElemAttr nor characterX/Y, so
// those fields are zeroed on the input to keep the round-trip equality
// assertions meaningful for the fields the wire actually carries.
func mobHitForVariant(v pt.TenantVariant, withExt bool, blockByte byte, reflect byte) DamageTakenInfo {
	m := mobHit(withExt, blockByte, reflect)
	if legacyLayout(v) {
		m.nMagicElemAttr = DamageElementTypeNone
		m.characterX = 0
		m.characterY = 0
	}
	return m
}

func assertCommon(t *testing.T, in, out DamageTakenInfo) {
	t.Helper()
	if out.UpdateTime() != in.UpdateTime() {
		t.Errorf("updateTime: got %v, want %v", out.UpdateTime(), in.UpdateTime())
	}
	if out.AttackIdx() != in.AttackIdx() {
		t.Errorf("nAttackIdx: got %v, want %v", out.AttackIdx(), in.AttackIdx())
	}
	if out.MagicElemAttr() != in.MagicElemAttr() {
		t.Errorf("nMagicElemAttr: got %v, want %v", out.MagicElemAttr(), in.MagicElemAttr())
	}
	if out.Damage() != in.Damage() {
		t.Errorf("damage: got %v, want %v", out.Damage(), in.Damage())
	}
}

func assertMob(t *testing.T, in, out DamageTakenInfo) {
	t.Helper()
	assertCommon(t, in, out)
	if out.MonsterTemplateId() != in.MonsterTemplateId() {
		t.Errorf("monsterTemplateId: got %v, want %v", out.MonsterTemplateId(), in.MonsterTemplateId())
	}
	if out.MonsterId() != in.MonsterId() {
		t.Errorf("monsterId: got %v, want %v", out.MonsterId(), in.MonsterId())
	}
	if out.Left() != in.Left() {
		t.Errorf("left: got %v, want %v", out.Left(), in.Left())
	}
	if out.Reflect() != in.Reflect() {
		t.Errorf("reflect: got %v, want %v", out.Reflect(), in.Reflect())
	}
	if out.BlockByte() != in.BlockByte() {
		t.Errorf("blockByte: got %v, want %v", out.BlockByte(), in.BlockByte())
	}
	if out.HasReflectExtension() != in.HasReflectExtension() {
		t.Errorf("hasReflectExtension: got %v, want %v", out.HasReflectExtension(), in.HasReflectExtension())
	}
	if out.StanceFlags() != in.StanceFlags() {
		t.Errorf("stanceFlags: got %v, want %v", out.StanceFlags(), in.StanceFlags())
	}
	if in.HasReflectExtension() {
		if out.IsPowerGuard() != in.IsPowerGuard() {
			t.Errorf("isPowerGuard: got %v, want %v", out.IsPowerGuard(), in.IsPowerGuard())
		}
		if out.ReflectTargetMobId() != in.ReflectTargetMobId() {
			t.Errorf("reflectTargetMobId: got %v, want %v", out.ReflectTargetMobId(), in.ReflectTargetMobId())
		}
		if out.HitAction() != in.HitAction() {
			t.Errorf("hitAction: got %v, want %v", out.HitAction(), in.HitAction())
		}
		if out.HitX() != in.HitX() || out.HitY() != in.HitY() {
			t.Errorf("hit point: got (%d,%d), want (%d,%d)", out.HitX(), out.HitY(), in.HitX(), in.HitY())
		}
		if out.CharacterX() != in.CharacterX() || out.CharacterY() != in.CharacterY() {
			t.Errorf("character point: got (%d,%d), want (%d,%d)", out.CharacterX(), out.CharacterY(), in.CharacterX(), in.CharacterY())
		}
	}
}

// Plain hit: no reflect, no block. The old decoder read the 14-byte
// extension unconditionally here and over-ran the packet; RoundTrip's
// Available()==0 assertion is the regression guard.
func TestDamageTakenInfoPlainHitRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := mobHitForVariant(v, false, 0, 0)
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertMob(t, input, output)
		})
	}
}

// Guardian block of a mob SKILL attack: blockByte==1 (blocked, no
// knockback) with NO extension. A decoder keyed on blockByte!=0 would
// over-read 14 bytes here.
func TestDamageTakenInfoBlockWithoutKnockbackRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := mobHitForVariant(v, false, 1, 0)
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertMob(t, input, output)
		})
	}
}

func TestDamageTakenInfoReflectExtensionRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := mobHitForVariant(v, true, 2, 30)
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertMob(t, input, output)
			if v.Region == "GMS" && v.MajorVersion >= 95 {
				if output.Guard() != input.Guard() {
					t.Errorf("guard: got %v, want %v", output.Guard(), input.Guard())
				}
			}
		})
	}
}

// Mob skill attacks carry the attack slot index (>= 1) — the old decoder
// misrouted these into the obstacle branch.
func TestDamageTakenInfoMobSkillAttackIdxRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := mobHitForVariant(v, false, 0, 0)
			input.nAttackIdx = DamageType(1)
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			assertMob(t, input, output)
		})
	}
}

func TestDamageTakenInfoNonMobRoundTrip(t *testing.T) {
	for _, attackIdx := range []DamageType{DamageTypeCounter, DamageTypeObstacle, DamageTypeStat} {
		for _, v := range pt.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				input := DamageTakenInfo{
					characterId:    100,
					updateTime:     999,
					nAttackIdx:     attackIdx,
					nMagicElemAttr: DamageElementTypeNone,
					damage:         120,
					obstacleData:   7,
					stanceFlags:    0,
				}
				output := DamageTakenInfo{}
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				assertCommon(t, input, output)
				if output.ObstacleData() != input.ObstacleData() {
					t.Errorf("obstacleData: got %v, want %v", output.ObstacleData(), input.ObstacleData())
				}
			})
		}
	}
}

// Sentinel −1 damage (Guardian/Fake block) must decode unchanged.
func TestDamageTakenInfoSentinelDamageRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := mobHitForVariant(v, false, 1, 0)
			input.damage = -1
			output := DamageTakenInfo{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Damage() != -1 {
				t.Errorf("damage: got %v, want -1", output.Damage())
			}
		})
	}
}

// Raw byte fixtures pin the exact v83 wire layout (no bGuard byte,
// little-endian): a 22-byte plain hit and a 36-byte extension hit.
func TestDamageTakenInfoV83ByteFixtures(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	l, _ := test.NewNullLogger()

	plain := []byte{
		0x39, 0x30, 0x00, 0x00, // updateTime 12345
		0xFF,                   // nAttackIdx -1 (touch)
		0x02,                   // nMagicElemAttr fire
		0xF4, 0x01, 0x00, 0x00, // damage 500
		0x44, 0x0D, 0x03, 0x00, // mobTemplateId 200004
		0x2A, 0x00, 0x00, 0x00, // mobId 42
		0x01, // left
		0x00, // reflect
		0x01, // blockByte 1 (block, no knockback -> NO extension)
		0x05, // stanceFlags
	}
	req := request.Request(plain)
	reader := request.NewRequestReader(&req, 0)
	m := DamageTakenInfo{}
	m.Decode(l, ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Fatalf("plain fixture: %d unconsumed bytes", reader.Available())
	}
	if m.Damage() != 500 || m.MonsterTemplateId() != 200004 || m.MonsterId() != 42 ||
		m.BlockByte() != 1 || m.HasReflectExtension() || m.StanceFlags() != 5 {
		t.Fatalf("plain fixture decoded wrong: %s", m.String())
	}

	ext := []byte{
		0x39, 0x30, 0x00, 0x00,
		0xFF,
		0x00,
		0xF4, 0x01, 0x00, 0x00,
		0x44, 0x0D, 0x03, 0x00,
		0x2A, 0x00, 0x00, 0x00,
		0x01,
		0x1E,                   // reflect 30 (Power Guard percent echo)
		0x00,                   // blockByte 0
		0x01,                   // isPowerGuard
		0x2A, 0x00, 0x00, 0x00, // reflectTargetMobId 42
		0x03,       // hitAction
		0x64, 0x00, // hitX 100
		0xC8, 0x00, // hitY 200
		0x6E, 0x00, // characterX 110
		0xD2, 0x00, // characterY 210
		0x00, // stanceFlags
	}
	req2 := request.Request(ext)
	reader2 := request.NewRequestReader(&req2, 0)
	m2 := DamageTakenInfo{}
	m2.Decode(l, ctx)(&reader2, nil)
	if reader2.Available() != 0 {
		t.Fatalf("ext fixture: %d unconsumed bytes", reader2.Available())
	}
	if !m2.HasReflectExtension() || !m2.IsPowerGuard() || m2.ReflectTargetMobId() != 42 ||
		m2.Reflect() != 30 || m2.HitAction() != 3 || m2.HitX() != 100 || m2.CharacterY() != 210 {
		t.Fatalf("ext fixture decoded wrong: %s", m2.String())
	}
}

// Raw byte fixtures pin the DIVERGENT v48 wire layout (design §2a): no
// nMagicElemAttr byte, a 10-byte reflect extension (no charX/charY), and a
// non-mob branch with no trailing stanceFlags byte. These are the fixtures
// that would silently pass on a v83-only decoder and fail on the real wire.
func TestDamageTakenInfoV48ByteFixtures(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	l, _ := test.NewNullLogger()

	// Mob-hit with reflect extension: 31 bytes (v83's equivalent is 34 —
	// v48 drops the 1-byte magicElemAttr and the 4-byte charX/charY pair).
	ext := []byte{
		0x39, 0x30, 0x00, 0x00, // updateTime 12345
		0xFF, // nAttackIdx -1 (touch)
		// (no nMagicElemAttr byte on v48)
		0xF4, 0x01, 0x00, 0x00, // damage 500
		0x44, 0x0D, 0x03, 0x00, // mobTemplateId 200004
		0x2A, 0x00, 0x00, 0x00, // mobId 42
		0x01,                   // left
		0x1E,                   // reflect 30
		0x00,                   // blockByte 0
		0x01,                   // isPowerGuard
		0x2A, 0x00, 0x00, 0x00, // reflectTargetMobId 42
		0x03,       // hitAction
		0x64, 0x00, // hitX 100
		0xC8, 0x00, // hitY 200
		// (no charX/charY on v48)
		0x05, // stanceFlags (mob branch)
	}
	req := request.Request(ext)
	reader := request.NewRequestReader(&req, 0)
	m := DamageTakenInfo{}
	m.Decode(l, ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Fatalf("v48 ext fixture: %d unconsumed bytes", reader.Available())
	}
	if m.Damage() != 500 || m.MonsterTemplateId() != 200004 || !m.HasReflectExtension() ||
		!m.IsPowerGuard() || m.HitY() != 200 || m.CharacterX() != 0 || m.StanceFlags() != 5 {
		t.Fatalf("v48 ext fixture decoded wrong: %s", m.String())
	}

	// Obstacle/non-mob: 11 bytes, no trailing stance (design §2a).
	obstacle := []byte{
		0x39, 0x30, 0x00, 0x00, // updateTime
		0xFE, // nAttackIdx -2 (non-mob sentinel)
		// (no nMagicElemAttr byte on v48)
		0xF4, 0x01, 0x00, 0x00, // damage 500
		0x07, 0x00, // obstacleData 7
		// (no trailing stanceFlags on pre-v83 non-mob)
	}
	req2 := request.Request(obstacle)
	reader2 := request.NewRequestReader(&req2, 0)
	m2 := DamageTakenInfo{}
	m2.Decode(l, ctx)(&reader2, nil)
	if reader2.Available() != 0 {
		t.Fatalf("v48 obstacle fixture: %d unconsumed bytes", reader2.Available())
	}
	if m2.Damage() != 500 || m2.ObstacleData() != 7 || m2.HasReflectExtension() {
		t.Fatalf("v48 obstacle fixture decoded wrong: %s", m2.String())
	}
}
