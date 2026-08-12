package handler

import (
	skill2 "atlas-channel/character/skill"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// buildSkillModel is a test helper that constructs a skill.Model for the
// given skill id and level using the package's Extract/RestModel path. It
// avoids creating a *_testhelpers.go (per CLAUDE.md) and avoids direct struct
// literals (private fields). The builder-style helper lives here, inline.
func buildSkillModel(t *testing.T, skillId skill.Id, level byte) skill2.Model {
	t.Helper()
	m, err := skill2.Extract(skill2.RestModel{
		Id:    uint32(skillId),
		Level: level,
	})
	if err != nil {
		t.Fatalf("buildSkillModel: Extract error: %v", err)
	}
	return m
}

// testTenantCtx builds a tenant-bearing context: shouldBroadcastKeydown
// resolves the wire skill id through the tenant's version set (task-187), so
// it can no longer run against a bare context.Background().
func testTenantCtx(t *testing.T, region string, major, minor uint16) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

// TestShouldBroadcastKeydown is a table-driven unit test for the gate function
// that controls whether a prepare/cancel packet should be relayed.
//
// Testability path chosen: no handler-test harness. The handler uses concrete
// processors (character.NewProcessor) that require a live tenant context and
// REST back-end. Instead we extracted shouldBroadcastKeydown as a small pure
// package-level function and test that directly, following the same pattern as
// computeReflect in character_attack_common_test.go.
func TestShouldBroadcastKeydown(t *testing.T) {
	// BowmasterHurricane is a known keydown skill (IsKeyDownSkill = true).
	const hurricaneId = skill.BowmasterHurricaneId
	// CorsairRapidFire is another keydown skill.
	const rapidFireId = skill.CorsairRapidFireId
	// FighterFinalAttackAxe (1100003) is NOT a keydown skill.
	const fighterFinalAttackId = skill.FighterFinalAttackAxeId

	cases := []struct {
		name    string
		skills  []skill2.Model
		skillId uint32
		want    bool
	}{
		{
			name: "owned keydown skill level>0 → broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, hurricaneId, 1),
			},
			skillId: uint32(hurricaneId),
			want:    true,
		},
		{
			name: "owned keydown skill level>0 (second keydown) → broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, rapidFireId, 3),
			},
			skillId: uint32(rapidFireId),
			want:    true,
		},
		{
			name: "non-keydown skill at level>0 → no broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, fighterFinalAttackId, 5),
			},
			skillId: uint32(fighterFinalAttackId),
			want:    false,
		},
		{
			name:    "skill not in character book → no broadcast",
			skills:  []skill2.Model{},
			skillId: uint32(hurricaneId),
			want:    false,
		},
		{
			name: "keydown skill at level 0 → no broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, hurricaneId, 0),
			},
			skillId: uint32(hurricaneId),
			want:    false,
		},
		{
			name: "multiple skills in book, only keydown one matches → broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, fighterFinalAttackId, 5),
				buildSkillModel(t, hurricaneId, 2),
			},
			skillId: uint32(hurricaneId),
			want:    true,
		},
		{
			name: "multiple skills in book, queried skill is non-keydown → no broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, hurricaneId, 2),
				buildSkillModel(t, fighterFinalAttackId, 5),
			},
			skillId: uint32(fighterFinalAttackId),
			want:    false,
		},
		{
			name: "owned Corkscrew Blow (keydown) level>0 → broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, skill.BrawlerCorkscrewBlowId, 1),
			},
			skillId: uint32(skill.BrawlerCorkscrewBlowId),
			want:    true,
		},
		{
			name: "owned Grenade (keydown) level>0 → broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, skill.GunslingerGrenadeId, 2),
			},
			skillId: uint32(skill.GunslingerGrenadeId),
			want:    true,
		},
		{
			name: "Corkscrew Blow at level 0 → no broadcast",
			skills: []skill2.Model{
				buildSkillModel(t, skill.BrawlerCorkscrewBlowId, 0),
			},
			skillId: uint32(skill.BrawlerCorkscrewBlowId),
			want:    false,
		},
		{
			name:    "Grenade not in character book → no broadcast",
			skills:  []skill2.Model{},
			skillId: uint32(skill.GunslingerGrenadeId),
			want:    false,
		},
	}

	ctx := testTenantCtx(t, "GMS", 83, 1)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := shouldBroadcastKeydown(ctx, tc.skills, tc.skillId)
			if got != tc.want {
				t.Errorf("shouldBroadcastKeydown(skills, %d) = %v, want %v", tc.skillId, got, tc.want)
			}
		})
	}
}

// TestKeydown_v48HideVsV72Corkscrew is the v0.48 correctness proof for the
// keydown-broadcast gate (task-187, the PRD-motivating bug): wire 5101004
// means SuperGmHide at v0.48 (NOT a keydown skill -- Hide is a toggle, not an
// attack) and BrawlerCorkscrewBlow at v0.62+ (a keydown attack). The SAME
// owned-skill entry (wire 5101004, level 1) must be gated differently purely
// by which tenant version resolves it.
func TestKeydown_v48HideVsV72Corkscrew(t *testing.T) {
	const wireId = uint32(5101004)
	skills := []skill2.Model{buildSkillModel(t, skill.Id(wireId), 1)}

	ctx48 := testTenantCtx(t, "GMS", 48, 1)
	if shouldBroadcastKeydown(ctx48, skills, wireId) {
		t.Fatal("v48 wire 5101004 (SuperGmHide) must NOT be treated as keydown")
	}

	ctx72 := testTenantCtx(t, "GMS", 72, 1)
	if !shouldBroadcastKeydown(ctx72, skills, wireId) {
		t.Fatal("v72 wire 5101004 (BrawlerCorkscrewBlow) MUST be treated as keydown")
	}
}

// TestStartChakraRecoveryGate pins PRD FR-1.1/FR-1.2/FR-1.4: below 50% of
// EFFECTIVE max HP the window opens; at or above it nothing happens at all —
// no window, no MP, no cooldown (the client sends no USE_SKILL, and even a
// crafted one is rejected by the use-skill gate in Task 6).
func TestStartChakraRecoveryGate(t *testing.T) {
	tests := []struct {
		name         string
		hp           uint16
		baseMaxHp    uint16
		effectiveMax uint32
		wantStarted  bool
	}{
		{"49 percent of effective", 490, 500, 1000, true},
		{"exactly 50 percent of effective", 500, 500, 1000, false},
		{"51 percent of effective", 510, 500, 1000, false},
		{"effective stats unavailable falls back to base", 240, 500, 0, true},
		{"base fallback rejects at half base", 250, 500, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			started := false
			deps := chakraPrepareDeps{
				skillLevel:     func() byte { return 3 },
				effectiveMaxHp: func() uint32 { return tc.effectiveMax },
				effectXY:       func(byte) (int16, int16, error) { return 99, 92, nil },
				start:          func(byte, int16, int16) { started = true },
			}
			startChakraRecoveryWith(testLogger(), tc.hp, tc.baseMaxHp, deps)
			if started != tc.wantStarted {
				t.Fatalf("window started = %v, want %v", started, tc.wantStarted)
			}
		})
	}
}

// TestStartChakraRecoverySnapshotsWzValues pins that x and y are captured
// from the caster's REAL skill-book level (design §5.1), not from the wire.
func TestStartChakraRecoverySnapshotsWzValues(t *testing.T) {
	var gotLevel byte
	var gotX, gotY int16
	deps := chakraPrepareDeps{
		skillLevel:     func() byte { return 7 },
		effectiveMaxHp: func() uint32 { return 1000 },
		effectXY: func(level byte) (int16, int16, error) {
			if level != 7 {
				t.Fatalf("effect looked up at level %d, want the skill-book level 7", level)
			}
			return 99, 116, nil
		},
		start: func(l byte, x int16, y int16) { gotLevel, gotX, gotY = l, x, y },
	}
	startChakraRecoveryWith(testLogger(), 100, 500, deps)
	if gotLevel != 7 || gotX != 99 || gotY != 116 {
		t.Fatalf("(level,x,y) = (%d,%d,%d), want (7,99,116)", gotLevel, gotX, gotY)
	}
}

// TestStartChakraRecoveryUnknownSkill pins that a caster who does not own
// the skill opens no window.
func TestStartChakraRecoveryUnknownSkill(t *testing.T) {
	started := false
	deps := chakraPrepareDeps{
		skillLevel:     func() byte { return 0 },
		effectiveMaxHp: func() uint32 { return 1000 },
		effectXY:       func(byte) (int16, int16, error) { return 99, 92, nil },
		start:          func(byte, int16, int16) { started = true },
	}
	startChakraRecoveryWith(testLogger(), 100, 500, deps)
	if started {
		t.Fatal("window opened for a caster who does not own Chakra")
	}
}

// TestStartChakraRecoveryEffectLookupFailure pins that an atlas-data miss
// opens no window rather than opening one with zeroed x/y (a zero x would
// zero every hit the caster takes).
func TestStartChakraRecoveryEffectLookupFailure(t *testing.T) {
	started := false
	deps := chakraPrepareDeps{
		skillLevel:     func() byte { return 3 },
		effectiveMaxHp: func() uint32 { return 1000 },
		effectXY:       func(byte) (int16, int16, error) { return 0, 0, errors.New("upstream down") },
		start:          func(byte, int16, int16) { started = true },
	}
	startChakraRecoveryWith(testLogger(), 100, 500, deps)
	if started {
		t.Fatal("window opened despite an effect lookup failure")
	}
}
