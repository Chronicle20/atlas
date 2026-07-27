package summon

import (
	"atlas-summons/data/skill/effect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// perSkillEffectSource returns a distinct effect per skill id, so a Beholder
// spawn can resolve the cast skill (1321007), the aura skill (1320008) and the
// hex skill (1320009) independently — mirroring atlas-data's per-skill resource.
type perSkillEffectSource struct {
	bySkill map[uint32]effect.Model
}

func (s perSkillEffectSource) GetEffect(skillId uint32, _ byte) (effect.Model, error) {
	return s.bySkill[skillId], nil
}

func TestBeholderSpawnSnapshotsAuraAndHex(t *testing.T) {
	// AURA_OF_BEHOLDER(1320008): hp=200 (heal), x=4 (heal every 4s) — real WZ
	// Skill.wz/132.img shape (hp restored per tick; x = interval seconds).
	aura, _ := effect.Extract(effect.RestModel{Hp: 200, X: 4})
	// HEX_OF_BEHOLDER(1320009): periodic owner buff. Real WZ grants defensive
	// stats (pdd->WEAPON_DEFENSE etc); using WEAPON_DEFENSE here as the delta.
	// x=4 (interval seconds), duration=40000ms.
	hex, _ := effect.Extract(effect.RestModel{
		X:        4,
		Duration: 40000,
		Statups:  []effect.StatupRestModel{{Type: "WEAPON_DEFENSE", Amount: 20}},
	})
	// Beholder summon itself (1321007): x drives hp = x+1; modest value.
	beholder, _ := effect.Extract(effect.RestModel{X: 5, Duration: 200000})

	src := perSkillEffectSource{bySkill: map[uint32]effect.Model{
		uint32(skillconst.DarkKnightBeholderId):          beholder,
		uint32(skillconst.DarkKnightAuraOfTheBeholderId): aura,
		uint32(skillconst.DarkKnightHexOfTheBeholderId):  hex,
	}}

	p, ten, ctx := newSpawnProcessor(t, effect.Model{})
	_ = ten
	_ = ctx
	p.effects = src

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	before := time.Now()
	m, err := p.Spawn(f, 42, uint32(skillconst.DarkKnightBeholderId), 20, 100, -50, 7, 9)
	if err != nil {
		t.Fatalf("Spawn returned error: %v", err)
	}
	if m.SummonType() != SummonTypeBuffAura {
		t.Fatalf("expected BUFF_AURA, got %s", m.SummonType())
	}
	// Beholder hp = effect.X + 1 (unchanged behavior).
	if m.Hp() != 6 {
		t.Fatalf("expected beholder hp == effect.X+1 (6), got %d", m.Hp())
	}
	if m.HealAmount() != 200 {
		t.Fatalf("expected HealAmount 200, got %d", m.HealAmount())
	}
	if m.HealInterval() != 4*time.Second {
		t.Fatalf("expected HealInterval 4s, got %v", m.HealInterval())
	}
	if m.BuffInterval() != 4*time.Second {
		t.Fatalf("expected BuffInterval 4s, got %v", m.BuffInterval())
	}
	if m.BuffDuration() != 40000 {
		t.Fatalf("expected BuffDuration 40000ms, got %d", m.BuffDuration())
	}
	if m.BuffLevel() != 9 {
		t.Fatalf("expected BuffLevel == hexLevel (9), got %d", m.BuffLevel())
	}
	if m.BuffSourceId() != int32(skillconst.DarkKnightHexOfTheBeholderId) {
		t.Fatalf("expected BuffSourceId 1320009 (positive: client looks up the skill template for the icon; negative crashes), got %d", m.BuffSourceId())
	}
	if len(m.BuffChanges()) != 1 || m.BuffChanges()[0].Type != "WEAPON_DEFENSE" || m.BuffChanges()[0].Amount != 20 {
		t.Fatalf("expected BuffChanges [WEAPON_DEFENSE +20], got %+v", m.BuffChanges())
	}
	if m.NextHealAt().IsZero() || m.NextHealAt().Before(before) {
		t.Fatalf("expected NextHealAt set to a future time, got %v", m.NextHealAt())
	}
	if m.NextBuffAt().IsZero() || m.NextBuffAt().Before(before) {
		t.Fatalf("expected NextBuffAt set to a future time, got %v", m.NextBuffAt())
	}
}
