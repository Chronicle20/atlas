# Context — task-161 keydown-aura-broadcast-skills

Companion to `plan.md`. Key files, decisions, and dependencies an executor needs
before touching code. Everything here is verified against source at plan time.

## The change in one sentence

Add `BrawlerCorkscrewBlowId` (5101004) and `GunslingerGrenadeId` (5201002) to the
single `skill.IsKeyDownSkill` predicate. That one edit turns on the cast-aura relay
AND the attack-packet `tKeyDown` field for both skills, because every call site
already routes through that predicate.

## Why only two skills (design supersedes PRD)

The PRD named four candidates. IDA verification (design §2.1) against the v83 client's
`is_keydown_skill` switch (`?is_keydown_skill@@YAHJ@Z` @ 0x4fb08f) proved:

| Skill | ID | In client set? | Verdict |
|---|---|---|---|
| Brawler Corkscrew Blow | 5101004 (0x4DD5CC) | ✅ v83/v87/v95/jms185 | ADD |
| Gunslinger Grenade | 5201002 (0x4F5C6A) | ✅ v83/v87/v95/jms185 | ADD |
| FP Mage Explosion | 2111002 | ❌ absent everywhere | DROP (FR-1.4) |
| Chief Bandit Chakra | 4211001 | ❌ absent everywhere | DROP (FR-1.4) |

Do NOT add Explosion/Chakra — they are not keydown in the client; adding them would
render a phantom aura on observers and make `attack_info.go` over-read 4 bytes,
corrupting combat parsing. This decision is guarded by tests in Tasks 1 and 2.

## Why one predicate, not a split (design §3)

The client gates both the display aura (`OnSkillPrepare`) and the attack `tKeyDown`
field through the SAME function (`is_keydown_skill`, xref'd from
`TryDoingShootAttack`@0x955213 and `TryDoingMeleeAttack`@0x952931). The display-set
and wire-set are one set, so the PRD's proposed `BroadcastsCastAura` split (FR-3.3)
is not built. Single predicate = least churn + most faithful.

## Key files

Production (the only edit):
- `libs/atlas-constants/skill/model.go:58-74` — `IsKeyDownSkill(Id) bool`, an
  `Is(skillId, ...)` OR-list. Add the two IDs here.
- Constants already defined: `libs/atlas-constants/skill/constants.go:3193`
  (`BrawlerCorkscrewBlowId = Id(5101004)`), `:3215` (`GunslingerGrenadeId = Id(5201002)`).

Downstream call sites (NO edits — they pick up the change automatically):
- `libs/atlas-packet/model/attack_info.go:113` (encode) / `:232` (decode) — writes/reads
  `keyDown` u32 when `IsKeyDownSkill(m.skillId)` OR `NeedsCharging(m.skillId)`.
  Both new skills are `charge:false`, so pre-change they carried NO keyDown field;
  the predicate edit is a latent attack-parse fix, not just cosmetics (design §5).
- `services/atlas-channel/.../socket/handler/character_skill_prepare.go:26-33` —
  `shouldBroadcastKeydown(skills, skillId)` = owns skill (level>0) && `IsKeyDownSkill`.
  Gates the `AnnounceForeignSkillPrepare` relay.
- `services/atlas-channel/.../socket/handler/character_buff_cancel.go:33-39` — key-up
  relay via `AnnounceForeignSkillCancel`, gated by the same predicate + ownership.

Test files:
- Create `libs/atlas-constants/skill/model_test.go` (package `skill`, none exists yet).
- Extend `libs/atlas-packet/model/attack_info_test.go` (package `model`; helper
  `sampleAttackInfo(AttackType)` sets skillId=0; setters `SetSkillId`, `SetKeydown` exist).
- Extend `libs/atlas-packet/character/clientbound/skill_prepare_foreign_test.go` and
  `skill_cancel_foreign_test.go` (existing byte-fixture pattern uses charId=1001,
  `pt.Encode`, `pt.CreateContext`).
- Extend `services/atlas-channel/.../socket/handler/character_skill_prepare_test.go` —
  `TestShouldBroadcastKeydown` and helper `buildSkillModel(t, skill.Id, byte)` ALREADY
  exist; add table cases (do not create a new helper or `*_testhelpers.go`).

## Packet structures (skill-agnostic — no branch on our IDs)

- `SkillPrepareForeign` (`skill_prepare_foreign.go`): charId u32, skillId u32, level u8,
  action u16, actionSpeed u8. Identical across all 5 versions.
- `SkillCancelForeign` (`skill_cancel_foreign.go`): charId u32, skillId u32.
- `SkillPrepareInfo` (`model/skill_prepare_info.go`): the only skill-id branch is a
  trailing swallowMobId gated on `skillId == 33101005` (GMS v95+/JMS) — unrelated to our IDs.
- `attack_info.go` skill-id branches: `IsKeyDownSkill`, `NeedsCharging`, and equals-checks
  for `NightWalkerStage3PoisonBombId` / `ThunderBreakerStage3SparkId` only — none of our
  two IDs hit the latter, so the byte-length delta for our IDs is purely the keyDown u32.

## Test harness facts

- `pt.Variants` (in `libs/atlas-packet/test`) iterates the five tenant variants:
  GMS v83, v84, v87, v95, JMS v185. Each has `.Name`, `.Region`, `.MajorVersion`, `.MinorVersion`.
- `pt.CreateContext(region, major, minor)` builds a tenant context;
  `pt.Encode(t, ctx, encodeFn, options)` returns encoded bytes.
- `atlas-channel` skill model built via `skill2.Extract(skill2.RestModel{Id, Level})`
  (concrete processors need live tenant/REST context, so `shouldBroadcastKeydown` is
  tested as a pure function directly).

## Verified wire bytes (little-endian)

- 5101004 = 0x4DD5CC → `CC D5 4D 00`
- 5201002 = 0x4F5C6A → `6A 5C 4F 00`
- 1001 (test charId) = 0x3E9 → `E9 03 00 00`

## Cross-version safety (design §2.4)

Both survivors are keydown in v83/v87/v95/jms185 (v84 bracketed as byte-identical to
v83 per task-083). The Go predicate is version-agnostic, so this introduces NO new
wire divergence for any supported tenant. `TestAttackInfoKeydownField` runs across all
`pt.Variants` to enforce this in CI. (Aside: v95 already drops the three Monster Magnet
IDs Atlas still lists — a PRE-EXISTING divergence this task neither creates nor widens.)

## Verification gate (CLAUDE.md + design §7)

Changed modules: `libs/atlas-constants`, `libs/atlas-packet`, `services/atlas-channel`.
Run `go test -race ./...`, `go vet ./...`, `go build ./...` (channel), plus
`docker buildx bake atlas-channel` from the worktree root and `tools/redis-key-guard.sh`
from repo root. See plan.md Task 5.

## Pitfalls / project memory

- No `go work sync` for go.sum drift (rewrites all 153 modules) — use `go mod tidy` in
  the one module if needed.
- Run guard scripts from repo root WITHOUT a global `GOWORK=off` prefix (false FAIL).
- Read files before editing (tool requirement).
- Related prior work: task-099 built the prepare/cancel relay handlers and the
  clientbound foreign packets this task reuses.
