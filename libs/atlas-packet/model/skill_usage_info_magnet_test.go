package model

import (
	"encoding/binary"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// decodeMagnetBody runs the shared decoder over buf under the given tenant
// version and returns the model plus the reader's unconsumed byte count. A
// non-zero remainder means the layout is wrong, which is the single most
// valuable assertion these tests make.
func decodeMagnetBody(t *testing.T, region string, major, minor uint16, buf []byte) (*SkillUsageInfo, int) {
	t.Helper()
	req := request.Request(buf)
	reader := request.NewRequestReader(&req, 0)
	m := &SkillUsageInfo{}
	m.Decode(nil, pt.CreateContext(region, major, minor))(&reader, nil)
	return m, reader.Available()
}

// modernMagnetBody builds a gms_61+/jms magnet body: uint32 grab count,
// (objectId uint32, grabbed byte) per entry, trailing direction byte, NO delay.
func modernMagnetBody(skillId skill.Id, level byte, grabs []MagnetGrab, left bool) []byte {
	buf := make([]byte, 0, 13+5*len(grabs))
	buf = binary.LittleEndian.AppendUint32(buf, 12345)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skillId))
	buf = append(buf, level)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(grabs)))
	for _, g := range grabs {
		buf = binary.LittleEndian.AppendUint32(buf, g.ObjectId())
		if g.Grabbed() {
			buf = append(buf, 1)
		} else {
			buf = append(buf, 0)
		}
	}
	if left {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}
	return buf
}

// legacyMagnetBody builds the gms_48 magnet body: byte entry count, uint32
// object ids with entry[0] = the CASTER's own object id, trailing delay short,
// NO per-entry result and NO direction byte.
func legacyMagnetBody(skillId skill.Id, level byte, casterObjectId uint32, mobIds []uint32, delay uint16) []byte {
	buf := make([]byte, 0, 12+4*(len(mobIds)+1))
	buf = binary.LittleEndian.AppendUint32(buf, 12345)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skillId))
	buf = append(buf, level)
	buf = append(buf, byte(len(mobIds)+1))
	buf = binary.LittleEndian.AppendUint32(buf, casterObjectId)
	for _, id := range mobIds {
		buf = binary.LittleEndian.AppendUint32(buf, id)
	}
	buf = binary.LittleEndian.AppendUint16(buf, delay)
	return buf
}

func TestDecodeMagnetModern_MixedGrabResults(t *testing.T) {
	want := []MagnetGrab{
		NewMagnetGrab(1001, true),
		NewMagnetGrab(1002, false),
		NewMagnetGrab(0, true), // released slot — id 0 is legitimate on the wire
	}
	m, left := decodeMagnetBody(t, "GMS", 83, 1,
		modernMagnetBody(skill.HeroMonsterMagnetId, 30, want, true))

	if left != 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", left)
	}
	got := m.MagnetGrabs()
	if len(got) != len(want) {
		t.Fatalf("MagnetGrabs len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ObjectId() != want[i].ObjectId() || got[i].Grabbed() != want[i].Grabbed() {
			t.Fatalf("MagnetGrabs[%d] = (%d,%v), want (%d,%v)",
				i, got[i].ObjectId(), got[i].Grabbed(), want[i].ObjectId(), want[i].Grabbed())
		}
	}
	if !m.Direction() {
		t.Fatal("Direction = false, want true (stance&1 == 1 means facing left)")
	}
	if m.Delay() != 0 {
		t.Fatalf("Delay = %d, want 0 — the modern magnet body carries no delay short", m.Delay())
	}
}

func TestDecodeMagnetModern_EmptyGrabTable(t *testing.T) {
	// PaladinMonsterMagnetId (1221001) has no entry in the gms_95.1 generated
	// skill set (libs/atlas-constants/skill/version_gms_95_1_gen.go) — unlike
	// every other provisioned GMS version, which all map it — so
	// constants.For(...).Skill.Resolve fails and isMonsterMagnet correctly
	// reports false for it there. Using HeroMonsterMagnetId instead, which IS
	// mapped at v95, keeps this test's point (the modern branch handles an
	// empty grab table) without depending on that gap in the constants data.
	m, left := decodeMagnetBody(t, "GMS", 95, 1,
		modernMagnetBody(skill.HeroMonsterMagnetId, 1, nil, false))

	if left != 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", left)
	}
	if len(m.MagnetGrabs()) != 0 {
		t.Fatalf("MagnetGrabs = %v, want empty", m.MagnetGrabs())
	}
	if m.Direction() {
		t.Fatal("Direction = true, want false")
	}
}

func TestDecodeMagnetModern_JmsTakesModernBranch(t *testing.T) {
	m, left := decodeMagnetBody(t, "JMS", 185, 1,
		modernMagnetBody(skill.DarkKnightMonsterMagnetId, 20,
			[]MagnetGrab{NewMagnetGrab(7, true)}, true))

	if left != 0 {
		t.Fatalf("jms_185 must take the modern branch; %d unconsumed bytes", left)
	}
	if len(m.MagnetGrabs()) != 1 || m.MagnetGrabs()[0].ObjectId() != 7 {
		t.Fatalf("MagnetGrabs = %v, want [(7,true)]", m.MagnetGrabs())
	}
}

func TestDecodeMagnetLegacy_DiscardsLeadingCasterEntry(t *testing.T) {
	m, left := decodeMagnetBody(t, "GMS", 48, 1,
		legacyMagnetBody(skill.HeroMonsterMagnetId, 15, 900001, []uint32{2001, 2002}, 750))

	if left != 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", left)
	}
	got := m.MagnetGrabs()
	if len(got) != 2 {
		t.Fatalf("MagnetGrabs len = %d, want 2 — entry[0] is the caster and must be discarded", len(got))
	}
	if got[0].ObjectId() != 2001 || got[1].ObjectId() != 2002 {
		t.Fatalf("MagnetGrabs ids = [%d %d], want [2001 2002]", got[0].ObjectId(), got[1].ObjectId())
	}
	for i, g := range got {
		if !g.Grabbed() {
			t.Fatalf("MagnetGrabs[%d].Grabbed = false; v48 sends no per-entry result, every surviving entry is an unconditional grab", i)
		}
	}
	if m.Delay() != 750 {
		t.Fatalf("Delay = %d, want 750", m.Delay())
	}
	if m.Direction() {
		t.Fatal("Direction = true; v48 sends no direction byte")
	}
}

func TestDecodeMagnetLegacy_CasterOnlyYieldsNoGrabs(t *testing.T) {
	m, left := decodeMagnetBody(t, "GMS", 48, 1,
		legacyMagnetBody(skill.PaladinMonsterMagnetId, 1, 900001, nil, 600))

	if left != 0 {
		t.Fatalf("reader has %d unconsumed bytes after decode", left)
	}
	if len(m.MagnetGrabs()) != 0 {
		t.Fatalf("MagnetGrabs = %v, want empty (only the caster entry was sent)", m.MagnetGrabs())
	}
	if m.Delay() != 600 {
		t.Fatalf("Delay = %d, want 600", m.Delay())
	}
}

// TestDecodeMagnetHostileCountDoesNotSpin pins the allocation/loop bound. The
// shared reader returns 0 WITHOUT advancing pos once exhausted
// (atlas-socket/request/reader.go), so an unbounded loop over a client-supplied
// uint32 count would spin ~4 billion times on the channel's packet goroutine.
func TestDecodeMagnetHostileCountDoesNotSpin(t *testing.T) {
	buf := make([]byte, 0, 17)
	buf = binary.LittleEndian.AppendUint32(buf, 1)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.HeroMonsterMagnetId))
	buf = append(buf, 30)
	buf = binary.LittleEndian.AppendUint32(buf, 0xFFFFFFFF) // hostile grabCount
	buf = binary.LittleEndian.AppendUint32(buf, 1001)       // one real entry
	buf = append(buf, 1)

	m, _ := decodeMagnetBody(t, "GMS", 83, 1, buf)
	if len(m.MagnetGrabs()) > 4 {
		t.Fatalf("MagnetGrabs len = %d; the loop must be bounded by the bytes actually available", len(m.MagnetGrabs()))
	}
}

// TestDecodeMagnetDoesNotDisturbOtherSkills is the FR-1.3 / backward-compat
// guard: a magnet id must not be reachable through the mob-affecting, party or
// anti-repeat lists, and the representative non-magnet arms must decode
// byte-identically with the magnet branch present.
func TestDecodeMagnetDoesNotDisturbOtherSkills(t *testing.T) {
	for _, id := range []skill.Id{
		skill.HeroMonsterMagnetId,
		skill.PaladinMonsterMagnetId,
		skill.DarkKnightMonsterMagnetId,
	} {
		if isMobAffectingBuff(id) {
			t.Fatalf("skill [%d] must not be in isMobAffectingBuff", id)
		}
		if isPartyBuff(id) {
			t.Fatalf("skill [%d] must not be in isPartyBuff", id)
		}
		if isAntiRepeatBuffSkill(id) {
			t.Fatalf("skill [%d] must not be in isAntiRepeatBuffSkill", id)
		}
	}

	// Shadow Stars: updateTime(4) skillId(4) slv(1) javelinItemId(4).
	buf := make([]byte, 0, 13)
	buf = binary.LittleEndian.AppendUint32(buf, 999)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.NightLordShadowStarsId))
	buf = append(buf, 30)
	buf = binary.LittleEndian.AppendUint32(buf, 2070006)
	m, left := decodeMagnetBody(t, "GMS", 83, 1, buf)
	if left != 0 || m.SpiritJavelinItemId() != 2070006 {
		t.Fatalf("Shadow Stars decode regressed: javelin=%d, %d bytes left", m.SpiritJavelinItemId(), left)
	}
	if len(m.MagnetGrabs()) != 0 {
		t.Fatalf("non-magnet cast populated MagnetGrabs = %v", m.MagnetGrabs())
	}

	// Bishop Resurrection: updateTime(4) skillId(4) slv(1) bitmap(1) delay(2),
	// but per the existing TestDecodeBishopResurrectionReadsPartyBitmap
	// (skill_usage_info_test.go) the trailing delay short is present on the
	// wire yet deliberately left UNREAD by Decode (only skill.PriestDispelId
	// triggers the delay read inside the isPartyBuff arm) — pre-existing
	// behavior, unrelated to the magnet branch. This asserts that behavior is
	// unchanged, not that it is correct.
	buf = buf[:0]
	buf = binary.LittleEndian.AppendUint32(buf, 999)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(skill.BishopResurrectionId))
	buf = append(buf, 30)
	buf = append(buf, 0b101)
	buf = binary.LittleEndian.AppendUint16(buf, 400)
	m, left = decodeMagnetBody(t, "GMS", 83, 1, buf)
	if left != 2 || m.AffectedPartyMemberBitmap() != 0b101 || m.Delay() != 0 {
		t.Fatalf("Resurrection decode regressed: bitmap=%#b delay=%d, %d bytes left",
			m.AffectedPartyMemberBitmap(), m.Delay(), left)
	}
}

func TestMagnetBuilderSetters(t *testing.T) {
	grabs := []MagnetGrab{NewMagnetGrab(42, true)}
	m := NewSkillUsageInfoBuilder().
		SetSkillId(uint32(skill.HeroMonsterMagnetId)).
		SetMagnetGrabs(grabs).
		SetDirection(true).
		Build()

	if len(m.MagnetGrabs()) != 1 || m.MagnetGrabs()[0].ObjectId() != 42 {
		t.Fatalf("MagnetGrabs = %v, want [(42,true)]", m.MagnetGrabs())
	}
	if !m.Direction() {
		t.Fatal("Direction = false, want true")
	}
}
