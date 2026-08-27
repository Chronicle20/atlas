package handler

import (
	"atlas-channel/battleship"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	"atlas-channel/character/chakra"
	"atlas-channel/data/skill/effect"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"encoding/binary"
	"io"
	"math"
	"net"
	"testing"
	"time"

	skill2 "atlas-channel/character/skill"
	monsterdata "atlas-channel/data/monster"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type emissions struct {
	hp                 []int16
	mp                 []int16
	meso               []int32
	reflects           []uint32
	reflectAttackTypes []byte
}

func fakeDeps(em *emissions, buffs []buff.Model, skills []skill2.Model, eff effect.Model, mob monster.Model, tmpl monsterdata.Model) damageMitigationDeps {
	return damageMitigationDeps{
		getBuffs:           func(uint32) ([]buff.Model, error) { return buffs, nil },
		getSkills:          func(uint32) ([]skill2.Model, error) { return skills, nil },
		getEffect:          func(uint32, byte) (effect.Model, error) { return eff, nil },
		getMonster:         func(uint32) (monster.Model, error) { return mob, nil },
		getMonsterTemplate: func(uint32) (monsterdata.Model, error) { return tmpl, nil },
		changeHP: func(_ field.Model, _ uint32, amount int16) error {
			em.hp = append(em.hp, amount)
			return nil
		},
		changeMP: func(_ field.Model, _ uint32, amount int16) error {
			em.mp = append(em.mp, amount)
			return nil
		},
		requestChangeMeso: func(_ field.Model, _ uint32, _ uint32, _ string, amount int32) error {
			em.meso = append(em.meso, amount)
			return nil
		},
		damageMonster: func(_ field.Model, _ uint32, _ uint32, damages []uint32, attackType byte) error {
			em.reflects = append(em.reflects, damages...)
			for range damages {
				em.reflectAttackTypes = append(em.reflectAttackTypes, attackType)
			}
			return nil
		},
	}
}

func activeBuff(statType charconst.TemporaryStatType, amount int32) buff.Model {
	future := time.Now().Add(time.Hour)
	return buff.NewBuff(2001002, 20, 3600, []stat.Model{stat.NewStat(string(statType), amount)}, time.Now(), future, false)
}

func testTenantModel(t *testing.T, region string, major uint16) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), region, major, 1)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func testCharacter(t *testing.T, jobId job.Id, mp uint16, meso uint32, skills []skill2.Model) character.Model {
	t.Helper()
	c, err := character.NewBuilder().
		SetId(42).
		SetJobId(jobId).
		SetHp(1000).SetMaxHp(2000).
		SetMp(mp).SetMaxMp(2000).
		SetMeso(meso).
		SetSkills(skills).
		Build()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// damagePacket builds a decoded-equivalent DamageTakenInfo via the codec:
// struct fields are package-private to packetmodel, so encode+decode through
// the real codec is the construction path.
func damagePacket(t *testing.T, tm tenant.Model, attackIdx packetmodel.DamageType, damage int32, withPGExt bool) packetmodel.DamageTakenInfo {
	t.Helper()
	return decodeDamagePacket(t, tm, attackIdx, damage, withPGExt)
}

func damageTestField() field.Model {
	return field.NewBuilder(2, 1, 100000000).Build()
}

// decodeDamagePacket produces a DamageTakenInfo as the handler would see
// it, by round-tripping raw bytes through the real decoder.
func decodeDamagePacket(t *testing.T, tm tenant.Model, attackIdx packetmodel.DamageType, damage int32, withPGExt bool) packetmodel.DamageTakenInfo {
	t.Helper()
	ctx := tenant.WithContext(context.Background(), tm)
	l, _ := test.NewNullLogger()

	w := response.NewWriter(l)
	w.WriteInt(uint32(12345))
	w.WriteInt8(int8(attackIdx))
	w.WriteInt8(0)
	w.WriteInt32(damage)
	if attackIdx >= packetmodel.DamageTypePhysical {
		w.WriteInt(uint32(200100)) // mobTemplateId
		w.WriteInt(uint32(42))     // mobId
		w.WriteBool(true)          // left
		if withPGExt {
			w.WriteByte(30) // reflect echo
		} else {
			w.WriteByte(0)
		}
		if tm.Region() == "GMS" && tm.MajorVersion() >= 95 {
			w.WriteBool(false)
		}
		w.WriteByte(0) // blockByte
		if withPGExt {
			w.WriteBool(true)      // isPowerGuard
			w.WriteInt(uint32(42)) // reflectTargetMobId == attacking mob
			w.WriteByte(3)         // hitAction
			w.WriteInt16(100)
			w.WriteInt16(200)
			w.WriteInt16(110)
			w.WriteInt16(210)
		}
	} else {
		w.WriteInt16(0) // obstacleData
	}
	w.WriteByte(0) // stanceFlags

	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	m := packetmodel.NewDamageTakenInfo(42)
	m.Decode(l, ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Fatalf("test packet under-consumed: %d bytes left", reader.Available())
	}
	return m
}

// decodeMRDamagePacket produces a DamageTakenInfo carrying a Mana
// Reflection-shaped reflect extension: a non-zero reflect echo with
// isPowerGuard=false. decodeDamagePacket's withPGExt path cannot express
// this — it hardcodes isPowerGuard=true — so this sibling helper mirrors
// the same byte layout (GMS v83, no bGuard byte) with isPowerGuard forced
// false.
func decodeMRDamagePacket(t *testing.T, tm tenant.Model, attackIdx packetmodel.DamageType, damage int32, reflectEcho byte) packetmodel.DamageTakenInfo {
	t.Helper()
	ctx := tenant.WithContext(context.Background(), tm)
	l, _ := test.NewNullLogger()

	w := response.NewWriter(l)
	w.WriteInt(uint32(12345))
	w.WriteInt8(int8(attackIdx))
	w.WriteInt8(0)
	w.WriteInt32(damage)
	w.WriteInt(uint32(200100)) // mobTemplateId
	w.WriteInt(uint32(42))     // mobId
	w.WriteBool(true)          // left
	w.WriteByte(reflectEcho)   // reflect echo (non-zero => MR signal)
	if tm.Region() == "GMS" && tm.MajorVersion() >= 95 {
		w.WriteBool(false)
	}
	w.WriteByte(0)        // blockByte
	w.WriteBool(false)    // isPowerGuard = false (Mana Reflection, not Power Guard)
	w.WriteInt(uint32(0)) // reflectTargetMobId (not consulted by the MR gate)
	w.WriteByte(3)        // hitAction
	w.WriteInt16(100)
	w.WriteInt16(200)
	w.WriteInt16(110)
	w.WriteInt16(210)
	w.WriteByte(0) // stanceFlags

	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	m := packetmodel.NewDamageTakenInfo(42)
	m.Decode(l, ctx)(&reader, nil)
	if reader.Available() != 0 {
		t.Fatalf("test packet under-consumed: %d bytes left", reader.Available())
	}
	return m
}

func TestProcessDamageTakenNoBuffAppliesFullDamage(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 500, false)
	c := testCharacter(t, job.Id(100), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -500 {
		t.Fatalf("hp emissions=%v, want [-500]", em.hp)
	}
	if len(em.mp)+len(em.meso)+len(em.reflects) != 0 {
		t.Fatalf("unexpected side effects: %+v", em)
	}
}

func TestProcessDamageTakenMagicGuardEmitsHPAndMP(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffs := []buff.Model{activeBuff(charconst.TemporaryStatTypeMagicGuard, 80)}
	deps := fakeDeps(em, buffs, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, false)
	c := testCharacter(t, job.Id(200), 2000, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -200 {
		t.Fatalf("hp=%v, want [-200]", em.hp)
	}
	if len(em.mp) != 1 || em.mp[0] != -800 {
		t.Fatalf("mp=%v, want [-800]", em.mp)
	}
}

func TestProcessDamageTakenMesoGuardEmitsMesoDeduction(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffs := []buff.Model{activeBuff(charconst.TemporaryStatTypeMesoGuard, 81)}
	deps := fakeDeps(em, buffs, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, false)
	c := testCharacter(t, job.Id(422), 100, 1000000, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -500 {
		t.Fatalf("hp=%v, want [-500]", em.hp)
	}
	if len(em.meso) != 1 || em.meso[0] != -405 {
		t.Fatalf("meso=%v, want [-405]", em.meso)
	}
}

func TestProcessDamageTakenPowerGuardReflects(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffs := []buff.Model{activeBuff(charconst.TemporaryStatTypePowerGuard, 30)}
	mob, err := monster.NewBuilder(42, damageTestField(), 200100).SetHp(50000).SetMaxHp(100000).Build()
	if err != nil {
		t.Fatal(err)
	}
	deps := fakeDeps(em, buffs, nil, effect.Model{}, mob, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, true)
	c := testCharacter(t, job.Id(110), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.reflects) != 1 || em.reflects[0] != 300 {
		t.Fatalf("reflects=%v, want [300]", em.reflects)
	}
	if len(em.hp) != 1 || em.hp[0] != -700 {
		t.Fatalf("hp=%v, want [-700]", em.hp)
	}
}

// Forged claim: isPowerGuard set on the wire but no POWER_GUARD buff —
// the claim is ignored, full damage applies, nothing reflects.
func TestProcessDamageTakenForgedPowerGuardIgnored(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, true)
	c := testCharacter(t, job.Id(110), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.reflects) != 0 {
		t.Fatalf("reflects=%v, want none", em.reflects)
	}
	if len(em.hp) != 1 || em.hp[0] != -1000 {
		t.Fatalf("hp=%v, want [-1000]", em.hp)
	}
}

// Forged oversized damage is clamped, and the int16 conversion is bounded.
func TestProcessDamageTakenForgedDamageClamped(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 50000000, false)
	c := testCharacter(t, job.Id(100), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -32767 {
		t.Fatalf("hp=%v, want [-32767] (clamped int16)", em.hp)
	}
}

// Block sentinel (-1) must not touch HP; the old handler healed +1.
func TestProcessDamageTakenSentinelNoHPChange(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, -1, false)
	c := testCharacter(t, job.HeroId, 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 0 {
		t.Fatalf("hp=%v, want none for block sentinel", em.hp)
	}
}

func TestProcessDamageTakenAchillesPassive(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	skills := []skill2.Model{buildSkillModel(t, skillconst.HeroAchillesId, 30)}
	eff, err := effect.Extract(effect.RestModel{X: 850})
	if err != nil {
		t.Fatal(err)
	}
	deps := fakeDeps(em, nil, skills, eff, monster.Model{}, monsterdata.Model{})

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 1000, false)
	c := testCharacter(t, job.HeroId, 100, 0, skills)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -850 {
		t.Fatalf("hp=%v, want [-850] (Achilles x=850)", em.hp)
	}
}

// Forged Mana Reflection claim: the wire carries an MR-shaped reflect echo
// (isPowerGuard=false, reflect echo > 0, mob magic attack) but the
// character has no active MANA_REFLECTION buff — the claim is ignored,
// full damage applies, nothing reflects.
func TestProcessDamageTakenForgedManaReflectionIgnored(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})

	p := decodeMRDamagePacket(t, tm, packetmodel.DamageTypeMagic, 1000, 30)
	c := testCharacter(t, job.Id(200), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.reflects) != 0 {
		t.Fatalf("reflects=%v, want none", em.reflects)
	}
	if len(em.hp) != 1 || em.hp[0] != -1000 {
		t.Fatalf("hp=%v, want [-1000]", em.hp)
	}
}

// Valid Mana Reflection: an active MANA_REFLECTION buff plus a mob magic
// attack reflects the server-recomputed amount (raw * effect X / 100,
// capped at maxHp/20) at the MAGIC attack type, without reducing the
// caster's own damage (FR-10.3 — MR does not self-mitigate).
func TestProcessDamageTakenManaReflectionEmitsReflect(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	buffs := []buff.Model{activeBuff(charconst.TemporaryStatTypeManaReflection, 100)}
	mob, err := monster.NewBuilder(42, damageTestField(), 200100).SetHp(50000).SetMaxHp(100000).Build()
	if err != nil {
		t.Fatal(err)
	}
	eff, err := effect.Extract(effect.RestModel{X: 30})
	if err != nil {
		t.Fatal(err)
	}
	deps := fakeDeps(em, buffs, nil, eff, mob, monsterdata.Model{})

	p := decodeMRDamagePacket(t, tm, packetmodel.DamageTypeMagic, 1000, 30)
	c := testCharacter(t, job.Id(200), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	// raw 1000 * X 30 / 100 = 300, well under the maxHp/20 = 5000 cap.
	if len(em.reflects) != 1 || em.reflects[0] != 300 {
		t.Fatalf("reflects=%v, want [300]", em.reflects)
	}
	if len(em.reflectAttackTypes) != 1 || em.reflectAttackTypes[0] != byte(packetmodel.AttackTypeMagic) {
		t.Fatalf("reflectAttackTypes=%v, want [AttackTypeMagic]", em.reflectAttackTypes)
	}
	if len(em.hp) != 1 || em.hp[0] != -1000 {
		t.Fatalf("hp=%v, want [-1000] (Mana Reflection does not self-mitigate)", em.hp)
	}
}

// TestDamageAppliesChakraFactorAndInterrupts pins PRD FR-4.5 / FR-5.2: the
// interrupting hit itself takes the Chakra factor, and the window is closed
// afterwards so the pending heal cannot fire.
func TestDamageAppliesChakraFactorAndInterrupts(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	cleared := false
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})
	deps.getChakra = func(uint32) (chakra.Entry, bool) {
		return chakra.Entry{SkillLevel: 1, X: 200, Y: 9, StartedAt: time.Now()}, true
	}
	deps.clearChakra = func(uint32) bool { cleared = true; return true }

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 500, false)
	c := testCharacter(t, job.Id(100), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -1000 {
		t.Fatalf("hp=%v, want [-1000] (500 raw x 200%%)", em.hp)
	}
	if !cleared {
		t.Fatal("Chakra window was not cleared by the damaging hit")
	}
}

// TestDamageWithoutChakraWindowDoesNotInterrupt pins that the interrupt is
// only attempted when a window is actually open.
func TestDamageWithoutChakraWindowDoesNotInterrupt(t *testing.T) {
	l, _ := test.NewNullLogger()
	tm := testTenantModel(t, "GMS", 83)
	em := &emissions{}
	cleared := false
	deps := fakeDeps(em, nil, nil, effect.Model{}, monster.Model{}, monsterdata.Model{})
	deps.getChakra = func(uint32) (chakra.Entry, bool) { return chakra.Entry{}, false }
	deps.clearChakra = func(uint32) bool { cleared = true; return true }

	p := damagePacket(t, tm, packetmodel.DamageTypePhysical, 500, false)
	c := testCharacter(t, job.Id(100), 100, 0, nil)
	processDamageTaken(l, tm, damageTestField(), p, c, deps)

	if len(em.hp) != 1 || em.hp[0] != -500 {
		t.Fatalf("hp=%v, want [-500] (unfactored)", em.hp)
	}
	if cleared {
		t.Fatal("clearChakra was called with no window open")
	}
}

func TestGaugeCooldownValue(t *testing.T) {
	tests := []struct {
		name      string
		remaining int32
		expected  uint16
	}{
		{"normal", 8500, 8500},
		{"formula max fits (v87+ arm, SLV 10 @ 200)", 29000, 29000},
		{"defensive clamp above uint16", math.MaxUint16 + 1, math.MaxUint16},
		{"defensive floor below zero", -5, 0},
		{"one", 1, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := gaugeCooldownValue(tc.remaining); got != tc.expected {
				t.Errorf("gaugeCooldownValue(%d) = %d, want %d", tc.remaining, got, tc.expected)
			}
		})
	}
}

// TestShouldAnnounceGauge covers the call-site gate in isolation. The full
// handler can't be driven end-to-end in this package's tests (the
// pre-existing, unseamed character.NewProcessor(...).GetById() call returns
// early without a live character service, out of scope for this task), so
// the predicate that stands between a correct and an incorrect announce is
// verified directly against every battleship.DrainStatus value instead.
func TestShouldAnnounceGauge(t *testing.T) {
	tests := []struct {
		name     string
		status   battleship.DrainStatus
		expected bool
	}{
		{"DrainNotRiding does not announce", battleship.DrainNotRiding, false},
		{"DrainSkipped does not announce", battleship.DrainSkipped, false},
		{"DrainDrained announces", battleship.DrainDrained, true},
		{"DrainBroke does not announce", battleship.DrainBroke, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldAnnounceGauge(tc.status); got != tc.expected {
				t.Errorf("shouldAnnounceGauge(%v) = %v, want %v", tc.status, got, tc.expected)
			}
		})
	}
}

// discardConn is a minimal net.Conn stub. announceShipHpGauge's success path
// runs the real session.Announce -> announceEncrypted -> con.Write chain;
// a nil conn (the pattern used by other handler tests that never reach that
// chain) would panic here, so this test needs a live, harmless sink instead.
// The plaintext packet body is captured earlier, from inside
// gaugeProducerRecorder's BodyFunc, so nothing needs to be read back off
// this connection or decrypted.
type discardConn struct{}

func (discardConn) Read(_ []byte) (int, error)         { return 0, io.EOF }
func (discardConn) Write(b []byte) (int, error)        { return len(b), nil }
func (discardConn) Close() error                       { return nil }
func (discardConn) LocalAddr() net.Addr                { return nil }
func (discardConn) RemoteAddr() net.Addr               { return nil }
func (discardConn) SetDeadline(_ time.Time) error      { return nil }
func (discardConn) SetReadDeadline(_ time.Time) error  { return nil }
func (discardConn) SetWriteDeadline(_ time.Time) error { return nil }

var _ net.Conn = discardConn{}

// gaugeProducerRecorder is a fake writer.Producer that records how many
// times, and with what writer name, session.Announce requested a body
// writer. It also captures the plaintext bytes the caller's encoder
// produced, from inside the returned BodyFunc, before session.Announce's own
// encrypt/write step — so a test can assert the exact wire values without
// decrypting anything. A test asserting "no packet announced" checks calls
// == 0: announceShipHpGauge must return before ever invoking the producer.
type gaugeProducerRecorder struct {
	calls    int
	lastName string
	lastBody []byte
}

func (r *gaugeProducerRecorder) producer() writer.Producer {
	return func(name string) (socketwriter.BodyFunc, error) {
		r.calls++
		r.lastName = name
		return func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(l, ctx)(nil)
				r.lastBody = b
				return b
			}
		}, nil
	}
}

// newGaugeTestSession builds a v83 GMS tenant + context + session backed by
// discardConn (same mustTenant helper as newCashItemUseTestSession in
// character_cash_item_use_test.go). announceShipHpGauge only reads
// s.CharacterId() for error logging on the (unexercised, since discardConn
// never errors) failure path, so no registry/character-id wiring is needed.
func newGaugeTestSession(t *testing.T) (session.Model, context.Context, uuid.UUID) {
	t.Helper()
	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	s := session.NewSession(uuid.New(), ten, 0, discardConn{})
	return s, ctx, ten.Id()
}

func TestAnnounceShipHpGauge(t *testing.T) {
	const gaugeId = uint32(5221999)

	t.Run("valid options resolve and announce exactly once", func(t *testing.T) {
		s, ctx, tid := newGaugeTestSession(t)
		writer.RegisterTenantWriterOptions(tid, []opcodes.WriterConfig{
			{OpCode: "0xEA", Writer: charpkt.CharacterSkillCooldownWriter, Options: map[string]interface{}{
				"skills": map[string]interface{}{"BATTLESHIP_HP_GAUGE": float64(gaugeId)},
			}},
		})
		t.Cleanup(func() { writer.EvictTenantWriterOptions(tid) })

		rec := &gaugeProducerRecorder{}
		announceShipHpGauge(discardLogger(), ctx, rec.producer(), s, 8500)

		if rec.calls != 1 {
			t.Fatalf("producer calls = %d, want 1", rec.calls)
		}
		if rec.lastName != charpkt.CharacterSkillCooldownWriter {
			t.Errorf("writer name = %q, want %q", rec.lastName, charpkt.CharacterSkillCooldownWriter)
		}
		if len(rec.lastBody) != 6 {
			t.Fatalf("body length = %d, want 6 (uint32 skillId + uint16 cooldown)", len(rec.lastBody))
		}
		if gotSkillId := binary.LittleEndian.Uint32(rec.lastBody[0:4]); gotSkillId != gaugeId {
			t.Errorf("skillId = %d, want %d", gotSkillId, gaugeId)
		}
		if gotCooldown := binary.LittleEndian.Uint16(rec.lastBody[4:6]); gotCooldown != 8500 {
			t.Errorf("cooldown = %d, want %d", gotCooldown, 8500)
		}
	})

	t.Run("no writer options registered for tenant sends nothing", func(t *testing.T) {
		s, ctx, _ := newGaugeTestSession(t)
		// Deliberately do not call writer.RegisterTenantWriterOptions for
		// this tenant at all.

		rec := &gaugeProducerRecorder{}
		announceShipHpGauge(discardLogger(), ctx, rec.producer(), s, 8500)

		if rec.calls != 0 {
			t.Fatalf("producer calls = %d, want 0 (no writer options registered)", rec.calls)
		}
	})

	t.Run("options present but gauge key missing sends nothing", func(t *testing.T) {
		s, ctx, tid := newGaugeTestSession(t)
		writer.RegisterTenantWriterOptions(tid, []opcodes.WriterConfig{
			{OpCode: "0xEA", Writer: charpkt.CharacterSkillCooldownWriter, Options: map[string]interface{}{
				"skills": map[string]interface{}{"SOME_OTHER_SKILL": float64(123)},
			}},
		})
		t.Cleanup(func() { writer.EvictTenantWriterOptions(tid) })

		rec := &gaugeProducerRecorder{}
		announceShipHpGauge(discardLogger(), ctx, rec.producer(), s, 8500)

		if rec.calls != 0 {
			t.Fatalf("producer calls = %d, want 0 (BATTLESHIP_HP_GAUGE key missing from options)", rec.calls)
		}
	})
}
