# Keydown Cast-Aura Broadcast (Corkscrew Blow + Grenade) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the two IDA-verified keydown skills — Brawler Corkscrew Blow (`5101004`) and Gunslinger Grenade (`5201002`) — to `skill.IsKeyDownSkill` so observers see their cast/wind-up aura and the shared attack codec reads/writes the client's `tKeyDown` field for them, exactly as the v83 client does.

**Architecture:** The v61/v72/v79/v83/v84/v87/v95/jms185 clients gate BOTH the cast-aura relay (`DoActiveSkill_Prepare`/`OnSkillPrepare`) AND the attack-packet `tKeyDown` field through one client function, `is_keydown_skill`, whose set INCLUDES both survivors (design §2.3, §2.4). The design (§2.3) proves the display-keydown set and the wire-keydown set are one set. Therefore the change is a single 2-line edit to the one existing `IsKeyDownSkill` predicate — no split, no handler edit, no packet-model edit. Every downstream call site (`character_skill_prepare.go`, `character_buff_cancel.go`, `attack_info.go`) already routes through that predicate and picks up the two IDs automatically. The bulk of the work is byte-level regression coverage that pins the wire behavior and guards against future broadening (re-adding the PRD-dropped Explosion/Chakra).

**Post-merge note (legacy versions).** After merging `main` (2026-07-27), `pt.Variants` holds **11** tenant versions — the design's original five (v83/v84/v87/v95/jms185) PLUS v28, v48, v61, v72, v79, v86. Every version-swept test in this plan (Task 2 especially) now iterates all 11 automatically. Design §2.4 was re-run against the full set: v61/v72/v79 each carry both survivors in their real `is_keydown_skill` switch (IDA-verified, addresses in §2.4), and the pre-pirate v28/v48 clients never cast either skill (v48 has no `is_keydown_skill` at all and no Pirate class), so the predicate change is **inert** there. No plan step or the core edit changes because of the merge — only the swept-version count grows.

**Tech Stack:** Go, `libs/atlas-constants`, `libs/atlas-packet` (packet codecs + `pt` test harness with per-version `Variants`), `services/atlas-channel` (socket handlers).

> **Path convention:** all shell steps assume you are inside the task worktree.
> `cd "$(git rev-parse --show-toplevel)"` resolves to the worktree root without a
> literal path. Do all work on branch `task-161-keydown-aura-broadcast-skills`.

## Global Constraints

- **Only production edit is `libs/atlas-constants/skill/model.go`.** Do NOT edit `attack_info.go`, `character_skill_prepare.go`, `character_buff_cancel.go`, or any packet model. (design §4.2)
- **Add exactly two IDs**, `BrawlerCorkscrewBlowId` and `GunslingerGrenadeId`. Do NOT add `FirePoisonMagicianExplosionId` (2111002) or `ChiefBanditChakraId` (4211001) — they are NOT keydown in the v83 client and adding them corrupts attack parsing. (design §1, §3 alt-1)
- **No split predicate.** Do NOT introduce `BroadcastsCastAura` or any second predicate — the client makes no such distinction. (design §3, OQ-3 resolved)
- **Both constants already exist**: `BrawlerCorkscrewBlowId = Id(5101004)` (constants.go:3193), `GunslingerGrenadeId = Id(5201002)` (constants.go:3215). Do not redefine them.
- **Verified LE wire bytes** (from `int_convert`, design §2.1): `5101004 = 0x4DD5CC` → `CC D5 4D 00`; `5201002 = 0x4F5C6A` → `6A 5C 4F 00`.
- **Test conventions (CLAUDE.md):** use the project's Builder/Extract path for test model setup; do NOT create `*_testhelpers.go` files.
- **Changed modules for final verification:** `libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-channel`.
- **`pt.Variants` is now 11 versions** (post-merge): v28, v48, v61, v72, v79, v83, v84, v86, v87, v95, jms185. Version-swept tests iterate the whole set; do NOT re-scope any loop to a hand-listed subset. The two survivors are keydown wherever they exist (v61+) and absent-hence-inert in v28/v48 (design §2.4) — so `IsKeyDownSkill` staying version-agnostic is correct for all 11.
- **Worktree:** all work happens in the task worktree on branch `task-161-keydown-aura-broadcast-skills`. Verify the branch after each commit.

---

### Task 1: Add the two skills to `IsKeyDownSkill` (predicate change, red→green)

This is the only behavioral change. Genuine TDD: the membership test fails before the edit and passes after. Single module (`libs/atlas-constants`).

**Files:**
- Create: `libs/atlas-constants/skill/model_test.go`
- Modify: `libs/atlas-constants/skill/model.go:58-74` (the `IsKeyDownSkill` body)

**Interfaces:**
- Consumes: existing exported symbols `IsKeyDownSkill(Id) bool`, `Id`, and the skill-ID constants (all in `libs/atlas-constants/skill`).
- Produces: `IsKeyDownSkill` returns `true` for `BrawlerCorkscrewBlowId` and `GunslingerGrenadeId` — relied on by Tasks 2 and 3.

- [ ] **Step 1: Write the failing membership test**

Create `libs/atlas-constants/skill/model_test.go`:

```go
package skill

import "testing"

// TestIsKeyDownSkill pins the exact membership of the keydown predicate. The two
// task-161 additions (Corkscrew Blow, Grenade) are IDA-verified keydown in the v83
// client; the two PRD-dropped skills (Explosion, Chakra) are NOT keydown in any
// version and must never be re-added (adding them broadcasts a phantom aura and
// makes attack_info.go over-read a tKeyDown field the client never sends).
func TestIsKeyDownSkill(t *testing.T) {
	keydown := []Id{
		FirePoisonArchMagicianBigBangId,
		IceLightningArchMagicianBigBangId,
		BishopBigBangId,
		HeroMonsterMagnetId,
		PaladinMonsterMagnetId,
		DarkKnightMonsterMagnetId,
		BowmasterHurricaneId,
		MarksmanPiercingArrowId,
		CorsairRapidFireId,
		NightWalkerStage3PoisonBombId,
		WindArcherStage3HurricaneId,
		ThunderBreakerStage2CorkscrewBlowId,
		EvanStage4IceBreathId,
		EvanStage7FireBreathId,
		BrawlerCorkscrewBlowId, // 5101004 — added task-161 (IDA-verified keydown v61/v72/v79/v83/v87/v95/jms185)
		GunslingerGrenadeId,    // 5201002 — added task-161 (IDA-verified keydown v61/v72/v79/v83/v87/v95/jms185)
	}
	for _, id := range keydown {
		if !IsKeyDownSkill(id) {
			t.Errorf("IsKeyDownSkill(%d) = false, want true", uint32(id))
		}
	}

	notKeydown := []Id{
		FirePoisonMagicianExplosionId, // 2111002 — DROPPED (FR-1.4), not keydown in client
		ChiefBanditChakraId,           // 4211001 — DROPPED (FR-1.4), not keydown in client
		Id(1100003),                   // Hero Combo Attack — plain non-keydown control
	}
	for _, id := range notKeydown {
		if IsKeyDownSkill(id) {
			t.Errorf("IsKeyDownSkill(%d) = true, want false", uint32(id))
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd "$(git rev-parse --show-toplevel)/libs/atlas-constants" && go test ./skill/ -run TestIsKeyDownSkill -v`
Expected: FAIL — `IsKeyDownSkill(5101004) = false, want true` and `IsKeyDownSkill(5201002) = false, want true` (the other 14 members and the 3 negatives already pass).

- [ ] **Step 3: Add the two IDs to the predicate**

In `libs/atlas-constants/skill/model.go`, change the `IsKeyDownSkill` body (currently ending at `EvanStage7FireBreathId)`):

```go
func IsKeyDownSkill(skillId Id) bool {
	return Is(skillId,
		FirePoisonArchMagicianBigBangId,
		IceLightningArchMagicianBigBangId,
		BishopBigBangId,
		HeroMonsterMagnetId,
		PaladinMonsterMagnetId,
		DarkKnightMonsterMagnetId,
		BowmasterHurricaneId,
		MarksmanPiercingArrowId,
		CorsairRapidFireId,
		NightWalkerStage3PoisonBombId,
		WindArcherStage3HurricaneId,
		ThunderBreakerStage2CorkscrewBlowId,
		EvanStage4IceBreathId,
		EvanStage7FireBreathId,
		BrawlerCorkscrewBlowId, // 5101004 — IDA-verified keydown v61/v72/v79/v83/v87/v95/jms185 (task-161)
		GunslingerGrenadeId)    // 5201002 — IDA-verified keydown v61/v72/v79/v83/v87/v95/jms185 (task-161)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd "$(git rev-parse --show-toplevel)/libs/atlas-constants" && go test ./skill/ -run TestIsKeyDownSkill -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add libs/atlas-constants/skill/model.go libs/atlas-constants/skill/model_test.go
git commit -m "feat(task-161): add Corkscrew Blow + Grenade to keydown predicate"
git branch --show-current   # must print task-161-keydown-aura-broadcast-skills
```

---

### Task 2: Attack-codec `tKeyDown` byte-layout regression test

After Task 1, `attack_info.go` already reads/writes the `tKeyDown` u32 for the two skills (it was `else if NeedsCharging` before, and both skills are `charge:false`, so pre-Task-1 they carried NO keyDown field — this is the latent attack-parse fix, design §5). These tests are **regression guards**: they pin that a keydown skill's encoded attack is exactly 4 bytes longer than a non-keydown one, across all 11 tenant variants (the post-merge `pt.Variants` set), so any future broadening/narrowing of `IsKeyDownSkill` (e.g. re-adding Explosion/Chakra, or dropping Corkscrew) surfaces as a byte-count failure. They pass on write because Task 1 already landed the change. Single module (`libs/atlas-packet`).

**Files:**
- Modify: `libs/atlas-packet/model/attack_info_test.go` (add one test function + one import)

**Interfaces:**
- Consumes: `IsKeyDownSkill` behavior from Task 1; existing `sampleAttackInfo(AttackType) *AttackInfo`, `(*AttackInfo).SetSkillId`, `(*AttackInfo).SetKeydown`, `(*AttackInfo).Encode`, `pt.Variants`, `pt.CreateContext`, `pt.Encode`.
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Add the `skill` import to the test file**

At the top of `libs/atlas-packet/model/attack_info_test.go`, extend the import block so it reads:

```go
import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)
```

- [ ] **Step 2: Append the byte-layout regression test**

Add to `libs/atlas-packet/model/attack_info_test.go`:

```go
// TestAttackInfoKeydownField pins which skills cause the attack codec to emit the
// extra tKeyDown u32 (attack_info.go:113 encode / :232 decode). A keydown skill's
// encoded attack is exactly 4 bytes longer than the same attack with a non-keydown
// skillId — the field the v83 client writes at 0x955223 (design §2.3). This guards
// skill.IsKeyDownSkill against accidental broadening (Explosion/Chakra) or narrowing
// (dropping Corkscrew/Grenade), and runs across every tenant variant (all 11 in
// pt.Variants: v28/v48/v61/v72/v79/v83/v84/v86/v87/v95/jms185) so the cross-version
// safety of design §2.4 is enforced in CI. The baseline (skillId 0) and the two
// DROPPED skills are charge:false and non-keydown, so they carry NO keyDown field;
// Hurricane/Corkscrew/Grenade are keydown and add exactly 4 bytes. NOTE: this asserts
// Atlas's own version-agnostic self-consistency (skillId N adds 4 bytes iff
// IsKeyDownSkill(N)), not per-version client faithfulness — for the pre-pirate v28/v48
// contexts the survivors are never actually cast (design §2.4), so the +4 there is a
// harmless unreachable path, and the assertion's value is guarding the reachable versions.
func TestAttackInfoKeydownField(t *testing.T) {
	encLen := func(t *testing.T, region string, major uint16, skillId uint32) int {
		ctx := pt.CreateContext(region, major, 1)
		ai := sampleAttackInfo(AttackTypeMelee)
		ai.SetSkillId(skillId)
		ai.SetKeydown(0xAABBCCDD)
		return len(pt.Encode(t, ctx, ai.Encode, nil))
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			base := encLen(t, v.Region, v.MajorVersion, 0) // skillId 0: not keydown, not charging

			// Non-keydown skills must NOT add a keyDown field (== base length).
			for _, id := range []uint32{
				uint32(skill.FirePoisonMagicianExplosionId), // 2111002 — DROPPED, not keydown
				uint32(skill.ChiefBanditChakraId),           // 4211001 — DROPPED, not keydown
			} {
				if got := encLen(t, v.Region, v.MajorVersion, id); got != base {
					t.Errorf("skill %d: encoded len %d, want %d (non-keydown must carry no tKeyDown field)", id, got, base)
				}
			}

			// Keydown skills add exactly 4 bytes (the tKeyDown u32).
			for _, id := range []uint32{
				uint32(skill.BowmasterHurricaneId),   // 3121004 — pre-existing keydown (guards field didn't move)
				uint32(skill.BrawlerCorkscrewBlowId), // 5101004 — NEW keydown (task-161)
				uint32(skill.GunslingerGrenadeId),    // 5201002 — NEW keydown (task-161)
			} {
				if got := encLen(t, v.Region, v.MajorVersion, id); got != base+4 {
					t.Errorf("skill %d: encoded len %d, want %d (keydown must add a tKeyDown u32 = base+4)", id, got, base+4)
				}
			}
		})
	}
}
```

- [ ] **Step 3: Run the test to verify it passes (regression guard)**

Run: `cd "$(git rev-parse --show-toplevel)/libs/atlas-packet" && go test ./model/ -run TestAttackInfoKeydownField -v`
Expected: PASS for every variant — all 11 in `pt.Variants` (GMS v28/v48/v61/v72/v79/v83/v84/v86/v87/v95, JMS v185). This passes on write because Task 1 already added the two IDs; its value is catching future regressions. (The +4 delta holds on every variant because `IsKeyDownSkill` is version-agnostic; on the pre-pirate v28/v48 contexts that path is unreachable in production but still self-consistent — design §2.4.)

Sanity check the guard actually bites (optional, do NOT commit this): temporarily remove `BrawlerCorkscrewBlowId` from `model.go`'s predicate, re-run — the test must FAIL with `skill 5101004: encoded len ... want base+4`. Restore the ID before continuing.

- [ ] **Step 4: Run the full model package to confirm no regression**

Run: `cd "$(git rev-parse --show-toplevel)/libs/atlas-packet" && go test ./model/ -v`
Expected: PASS (existing `TestAttackInfoRoundTrip`, `TestAttackInfoVersionBoundary`, and the new test all green).

- [ ] **Step 5: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add libs/atlas-packet/model/attack_info_test.go
git commit -m "test(task-161): pin attack tKeyDown field membership across variants"
git branch --show-current   # must print task-161-keydown-aura-broadcast-skills
```

---

### Task 3: Foreign prepare/cancel relay byte fixtures for the two skills

The `SkillPrepareForeign` (charId u32, skillId u32, level u8, action u16/u8, actionSpeed u8) and `SkillCancelForeign` (charId u32, skillId u32) packets are skill-agnostic — they do not branch on skillId (design §4.2). These fixtures are coverage guards proving the relay carries the correct verified skillId bytes to the observer for each of the two skills (acceptance §byte-level). Single module (`libs/atlas-packet`).

> **Post-merge caveat (legacy action-field divergence).** `main` version-gated `SkillPrepareForeign`'s `action` field: GMS **< 79** (v61/v72) encodes it as a SINGLE byte (bit7=bLeft, bits0-6=nAction); v79+ as a 2-byte short (see `TestSkillPrepareForeignV61ByteFixture`/`...V72ByteFixture`). `SkillCancelForeign` is 8 bytes on every version (no divergence). The fixtures below use a **v83 context with `action=0x0142`** (2-byte, 12 bytes total), which still matches the current `TestSkillPrepareForeignByteFixture` v83 case — so they remain correct. Because the two survivors are IDA-verified keydown specifically on the new legacy versions v61/v72/v79 (design §2.4), **Step 1b (recommended)** adds one legacy fixture (v72, 1-byte action, 11 bytes) for Corkscrew so the byte-level coverage exercises a version where these skills are actually keydown-live — not only v83.

**Files:**
- Modify: `libs/atlas-packet/character/clientbound/skill_prepare_foreign_test.go` (add one test function)
- Modify: `libs/atlas-packet/character/clientbound/skill_cancel_foreign_test.go` (add one test function)

**Interfaces:**
- Consumes: existing `NewSkillPrepareForeign(characterId uint32, skillId uint32, level byte, action uint16, actionSpeed byte)`, `NewSkillCancelForeign(characterId uint32, skillId uint32)`, `(SkillPrepareForeign).Encode`, `(SkillCancelForeign).Encode`, `pt.CreateContext`, `pt.Encode`.
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Add the prepare-relay byte fixture**

Append to `libs/atlas-packet/character/clientbound/skill_prepare_foreign_test.go`:

```go
// TestSkillPrepareForeignByteFixtureKeydownSkills pins the exact relay bytes for
// the two task-161 keydown skills. Structure is skill-independent (charId u32,
// skillId u32, level u8, action u16, actionSpeed u8), so this proves the observer
// receives the correct verified skillId for each. Skill IDs decoded via int_convert
// (design §2.1): 5101004=0x4DD5CC, 5201002=0x4F5C6A. Fixed fields mirror the
// existing fixture: charId=1001, level=10, action=0x0142, actionSpeed=4.
func TestSkillPrepareForeignByteFixtureKeydownSkills(t *testing.T) {
	cases := []struct {
		name     string
		skillId  uint32
		expected []byte
	}{
		{
			name:    "Corkscrew Blow 5101004",
			skillId: 5101004,
			expected: []byte{
				0xE9, 0x03, 0x00, 0x00, // charId=1001 LE
				0xCC, 0xD5, 0x4D, 0x00, // skillId=5101004 (0x4DD5CC) LE
				0x0A,       // level=10
				0x42, 0x01, // action=0x0142 LE
				0x04,       // actionSpeed=4
			},
		},
		{
			name:    "Grenade 5201002",
			skillId: 5201002,
			expected: []byte{
				0xE9, 0x03, 0x00, 0x00, // charId=1001 LE
				0x6A, 0x5C, 0x4F, 0x00, // skillId=5201002 (0x4F5C6A) LE
				0x0A,       // level=10
				0x42, 0x01, // action=0x0142 LE
				0x04,       // actionSpeed=4
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 83, 1)
			input := NewSkillPrepareForeign(1001, tc.skillId, 10, 0x0142, 4)
			got := pt.Encode(t, ctx, input.Encode, nil)
			if len(got) != len(tc.expected) {
				t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X",
					len(got), len(tc.expected), got, tc.expected)
			}
			for i := range tc.expected {
				if got[i] != tc.expected[i] {
					t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X",
						i, got[i], tc.expected[i], got, tc.expected)
					break
				}
			}
		})
	}
}
```

- [ ] **Step 1b (recommended): Add a legacy v72 prepare fixture for Corkscrew**

Proves the relay carries the correct skillId on a legacy version where Corkscrew is genuinely keydown-live (design §2.4), and pins the legacy 1-byte `action` layout. Append to `libs/atlas-packet/character/clientbound/skill_prepare_foreign_test.go`:

```go
// TestSkillPrepareForeignByteFixtureKeydownV72 pins the legacy GMS v72 relay bytes for
// Corkscrew Blow (5101004) — a version where the client's is_keydown_skill switch
// (@0x4e5318) includes it (design §2.4). v72 encodes the action/direction field as a
// SINGLE byte (bit7=bLeft, bits0-6=nAction), so this packet is 11 bytes, not 12. action
// stays 0x42 so bit7=0 -> the byte is exactly 0x42. Mirrors TestSkillPrepareForeignV72ByteFixture.
func TestSkillPrepareForeignByteFixtureKeydownV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := NewSkillPrepareForeign(1001, 5101004, 10, 0x42, 4)
	expected := []byte{
		0xE9, 0x03, 0x00, 0x00, // charId=1001 LE
		0xCC, 0xD5, 0x4D, 0x00, // skillId=5101004 (0x4DD5CC) LE
		0x0A, // level=10
		0x42, // action=0x42 (1 BYTE on v72)
		0x04, // actionSpeed=4
	}
	got := pt.Encode(t, ctx, input.Encode, nil)
	if len(got) != len(expected) {
		t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X", len(got), len(expected), got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X", i, got[i], expected[i], got, expected)
			break
		}
	}
}
```

- [ ] **Step 2: Add the cancel-relay byte fixture**

Append to `libs/atlas-packet/character/clientbound/skill_cancel_foreign_test.go`:

```go
// TestSkillCancelForeignByteFixtureKeydownSkills pins the exact key-up cancel relay
// bytes (charId u32, skillId u32) for the two task-161 keydown skills, so the
// observer's aura stops for the correct skill. Skill IDs per design §2.1:
// 5101004=0x4DD5CC, 5201002=0x4F5C6A. charId=1001 mirrors the existing fixture.
func TestSkillCancelForeignByteFixtureKeydownSkills(t *testing.T) {
	cases := []struct {
		name     string
		skillId  uint32
		expected []byte
	}{
		{
			name:    "Corkscrew Blow 5101004",
			skillId: 5101004,
			expected: []byte{
				0xE9, 0x03, 0x00, 0x00, // charId=1001 LE
				0xCC, 0xD5, 0x4D, 0x00, // skillId=5101004 (0x4DD5CC) LE
			},
		},
		{
			name:    "Grenade 5201002",
			skillId: 5201002,
			expected: []byte{
				0xE9, 0x03, 0x00, 0x00, // charId=1001 LE
				0x6A, 0x5C, 0x4F, 0x00, // skillId=5201002 (0x4F5C6A) LE
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := pt.CreateContext("GMS", 83, 1)
			input := NewSkillCancelForeign(1001, tc.skillId)
			got := pt.Encode(t, ctx, input.Encode, nil)
			if len(got) != len(tc.expected) {
				t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X",
					len(got), len(tc.expected), got, tc.expected)
			}
			for i := range tc.expected {
				if got[i] != tc.expected[i] {
					t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X",
						i, got[i], tc.expected[i], got, tc.expected)
					break
				}
			}
		})
	}
}
```

- [ ] **Step 3: Run the clientbound package tests to verify they pass**

Run: `cd "$(git rev-parse --show-toplevel)/libs/atlas-packet" && go test ./character/clientbound/ -run 'TestSkill(Prepare|Cancel)ForeignByteFixtureKeydown' -v`
Expected: PASS for both `Corkscrew Blow 5101004` and `Grenade 5201002` subtests in each of the Skills fixtures, plus the legacy `TestSkillPrepareForeignByteFixtureKeydownV72` (11-byte, 1-byte action) from Step 1b.

- [ ] **Step 4: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add libs/atlas-packet/character/clientbound/skill_prepare_foreign_test.go libs/atlas-packet/character/clientbound/skill_cancel_foreign_test.go
git commit -m "test(task-161): byte-fixture the foreign prepare/cancel relay for Corkscrew + Grenade"
git branch --show-current   # must print task-161-keydown-aura-broadcast-skills
```

---

### Task 4: Handler gate coverage for the two skills (`shouldBroadcastKeydown`)

`TestShouldBroadcastKeydown` and its `buildSkillModel(t, skillId, level)` helper already exist in `character_skill_prepare_test.go` (using `skill2.Extract(skill2.RestModel{...})`, per CLAUDE.md's no-`*_testhelpers.go` rule). Extend its table with the two new skills to prove the ownership + keydown gate (FR-2.3) now takes the broadcast path for them. Regression guard (green after Task 1). Single module (`services/atlas-channel`).

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare_test.go` (add cases to the existing `cases` table in `TestShouldBroadcastKeydown`)

**Interfaces:**
- Consumes: existing `shouldBroadcastKeydown([]skill2.Model, uint32) bool`, `buildSkillModel(t, skill.Id, byte) skill2.Model`, and the `skill.BrawlerCorkscrewBlowId` / `skill.GunslingerGrenadeId` constants (packages `skill` and `skill2` are already imported in this file).
- Produces: nothing consumed by later tasks.

- [ ] **Step 1: Add the four table cases**

In `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare_test.go`, add these entries to the `cases := []struct{...}{ ... }` slice inside `TestShouldBroadcastKeydown` (place them before the closing `}` of the slice literal, matching the existing field names `name`, `skills`, `skillId`, `want`):

```go
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
```

- [ ] **Step 2: Run the handler test to verify it passes**

Run: `cd "$(git rev-parse --show-toplevel)/services/atlas-channel/atlas.com/channel" && go test ./socket/handler/ -run TestShouldBroadcastKeydown -v`
Expected: PASS for all existing subtests plus the four new ones (the two owned/level>0 cases return `true`; level-0 and not-in-book return `false`).

- [ ] **Step 3: Commit**

```bash
cd "$(git rev-parse --show-toplevel)"
git add services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare_test.go
git commit -m "test(task-161): cover Corkscrew + Grenade in shouldBroadcastKeydown gate"
git branch --show-current   # must print task-161-keydown-aura-broadcast-skills
```

---

### Task 5: Full verification across all changed modules

Run the CLAUDE.md build-and-verify gate for every changed module and the design §7 acceptance commands. No code changes; this task either passes clean or surfaces a defect to fix (then re-run).

**Files:** none (verification only).

**Interfaces:** none.

- [ ] **Step 1: `go test -race` in each changed module**

```bash
cd "$(git rev-parse --show-toplevel)"
(cd libs/atlas-constants && go test -race ./...)
(cd libs/atlas-packet && go test -race ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./...)
```

Expected: all three PASS (`ok` lines, no `FAIL`).

- [ ] **Step 2: `go vet` in each changed module**

```bash
cd "$(git rev-parse --show-toplevel)"
(cd libs/atlas-constants && go vet ./...)
(cd libs/atlas-packet && go vet ./...)
(cd services/atlas-channel/atlas.com/channel && go vet ./...)
```

Expected: clean (no output, exit 0).

- [ ] **Step 3: `go build` the affected service**

```bash
cd "$(git rev-parse --show-toplevel)"
(cd services/atlas-channel/atlas.com/channel && go build ./...)
```

Expected: clean (exit 0).

- [ ] **Step 4: `docker buildx bake atlas-channel` from the worktree root**

```bash
cd "$(git rev-parse --show-toplevel)"
docker buildx bake atlas-channel
```

Expected: build succeeds (`writing image ... done` / no error). Required by design §7 / PRD §10 even though no `go.mod` changed.

- [ ] **Step 5: `tools/redis-key-guard.sh` clean from the repo root**

```bash
cd "$(git rev-parse --show-toplevel)"
tools/redis-key-guard.sh
```

Expected: clean (no banned-call findings, exit 0). Run without a global `GOWORK=off` prefix (project memory: guard scripts false-FAIL under that prefix).

- [ ] **Step 5b: `tools/lint.sh --check` and `tools/goroutine-guard.sh` clean from the repo root**

`main` added these as mandatory CI gates (CLAUDE.md Build & Verification items 6 & 8) after this plan was first written. Both apply to the changed Go files:

```bash
cd "$(git rev-parse --show-toplevel)"
tools/goroutine-guard.sh          # no bare `go` added — passes trivially
tools/lint.sh --check             # gofumpt + goimports on the new test files / predicate edit
```

Expected: both exit 0. If `lint.sh --check` reports formatting drift on a new file, run `tools/lint.sh` (no flags) to fix in place, then re-commit that file. The other new guards (`service-registration-guard.sh`, `template-opcode-order-guard.sh`) are N/A — this task touches no `services.json`, deploy/k8s, docker-bake, `go.work`, or seed templates.

- [ ] **Step 6: Confirm the tree is clean and on-branch**

```bash
cd "$(git rev-parse --show-toplevel)"
git status --short          # expect empty (all work committed)
git branch --show-current   # must print task-161-keydown-aura-broadcast-skills
git log --oneline -5
```

Expected: no uncommitted changes; branch correct; the four task commits present.

---

## Notes for the executor

- **If any verification in Task 5 fails**, treat it as a bug in the corresponding task's code (not the plan): fix in place on the branch, re-commit, and re-run Task 5 from Step 1. Do not defer, stub, or split into a follow-up.
- **Do not run `go work sync`** for any go.sum drift — project memory warns it rewrites all modules. If a `go.sum`/`go.work.sum` issue appears, run `go mod tidy` in the specific module only.
- **The design's dropped-skill decision is load-bearing.** If a reviewer or later editor suggests "also add Explosion/Chakra for completeness," the answer is no — `TestIsKeyDownSkill` (Task 1) and `TestAttackInfoKeydownField` (Task 2) will fail if either is added, by design.
