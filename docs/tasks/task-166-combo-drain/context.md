# Combo Drain (Aran) — Execution Context

Task: task-166-combo-drain
Companion to: `plan.md` (implementation plan), `design.md` (approved design),
`prd.md` (requirements)
Revised: 2026-08-07 — branch rebased onto `main` @ `e0f5bd01d`; all references
re-derived; approach changed B → C; version scope added.
Revised: 2026-08-13 — `main` merged into the branch (`f1b4e4046`); every line
number below re-derived against the merge. No decision changed.

## What this task is

The `COMBO_DRAIN` buff (Aran skill 21100005) applies and renders correctly, but
the attack pipeline never reads it, so the heal never fires. This task replaces
the `// TODO Combo Drain` in atlas-channel's `processAttack` with a
once-per-attack heal of `totalDamage * x / 100` HP (buff statup amount `x`),
clamped to `math.MaxInt16`, emitted via the existing `ChangeHP` Kafka command.
Single service: **atlas-channel**. No new REST endpoints, Kafka topics,
packets, templates, or schema — on any supported version.

## Version scope (short form)

Eleven tenant templates are provisioned. Combo Drain is **in scope** on
`gms_79_1`, `gms_83_1`, `gms_84_1`, `gms_87_1`, `gms_95_1`, `jms_185_1`;
**N/A** on `gms_12_1`, `gms_48_1`, `gms_61_1`, `gms_72_1` (no Aran in those
clients' WZ); **blocked** on `gms_92_1` (client has Aran, but the template
routes no attack handler and `gms_v92` has no packet-matrix column, so the
opcodes to route against are unverified — a version bring-up, not this task).

The implementation is version-blind on purpose: a version without Aran cannot
produce a `COMBO_DRAIN` statup, so the gate never fires. Full table + evidence
in PRD §4A and design §6.

## Key files (line numbers verified against the `main` merge `f1b4e4046`)

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` | `processAttack` — new memoized `loadBuffs` closure before `pp := NewProjectileProcessor` (:888); `Plan` call site :889; Pick Pocket call site :919-925; proc call replaces `// TODO Combo Drain` at :1117. `cp` (`character.Processor`, :777) and `s` (session) already in scope. `loadEffectiveStats` at :898-912 is the memoization idiom to mirror. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_projectile.go` | `ProjectileProcessor.Plan` (:45 interface, :66 method) gains a `getBuffs` func param; internal `bp buff.Processor` field (:53, :61) and the fetch at :99 replaced by the injected loader. `hasBuff` (:199) / `computeCount` untouched. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain.go` | NEW — pure helpers `buffStatAmount`, `attackTotalDamage`, `comboDrainHealAmount` + orchestrator `comboDrainTryProc` (+ optionally `newAttackBuffLoader`). Style mirrors `drainHealAmount`/`drainTryHeal` (:519, :607 in common.go). |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_drain_test.go` | NEW — table-driven tests, production constructors only. |
| `services/atlas-channel/atlas.com/channel/character/buff/model.go:63` | `buff.NewBuff(sourceId, level, duration, changes, createdAt, expiresAt, noExpiry)` — **seven** args. `Expired()`, `Changes() []stat.Model`. |
| `services/atlas-channel/atlas.com/channel/character/buff/stat/model.go:16` | `stat.NewStat(statType string, amount int32)`; `Type() string`, `Amount() int32`. |
| `services/atlas-channel/atlas.com/channel/character/processor.go` | `ChangeHP(f field.Model, characterId uint32, amount int16) error` — emits the Kafka command; atlas-character owns max-HP clamping. |
| `libs/atlas-constants/character/temporary_stat.go:75` | `TemporaryStatTypeComboDrain = "COMBO_DRAIN"` — the gate stat. |
| `libs/atlas-packet/model/character_temporary_stat.go:163` | `COMBO_DRAIN` bit, registered unconditionally ahead of the first version-gated slot (bit 82, `:186`) — encodes on every version. |
| `libs/atlas-packet/model/attack_info.go` | `AttackInfo.DamageInfo()`, `AttackType{Melee,Ranged,Magic,Energy}`; `DamageInfo.Damages() []uint32`. |
| `services/atlas-data/atlas.com/data/skill/reader.go:407-408` | Emits the `COMBO_DRAIN` statup with amount = effect `x`, from the tenant's WZ. Version-blind. |
| `libs/atlas-constants/gen/wzsnapshot/*.json` | Per-version skill/job availability (`PROVENANCE.md` for method) — the source of the §4A table. |
| `docs/TODO.md:151` | `- [ ] Combo Drain` line item to check off. |
| `docs/packets/audits/status.json` | Attack + `STAT_CHANGED` + `GIVE_BUFF` coverage per version. Must be **unchanged** by this branch. |

Import aliases in this package differ per file: `character_attack_common.go`
uses `charconst` for `libs/atlas-constants/character` and `skill3` for
`libs/atlas-constants/skill`; `character_attack_projectile.go` uses `ts` and
`skillConst`. The new file uses `charconst`.

## Decisions locked in design (do not re-litigate)

- **Approach C** (reversed from v1's Approach B): a per-attack memoized
  `loadBuffs` closure, injected into the projectile planner and Pick Pocket and
  called by Combo Drain. Ceiling of one buff REST read per attack; the two
  existing consumers keep their gate-before-fetch behavior so no path pays a
  read it did not need. A fetch failure is logged once and cached as "no
  buffs"; never aborts the attack.
- **Why B was dropped**: when v1 was written, `Plan` was the only buff
  consumer, so an eager hoist cost nothing. `main` now has two gate-first
  consumers, and an eager fetch would add a read to every melee/magic attack
  that currently performs none.
- **Buff-only gate**: no job / skill-ownership / attack-type / version check
  (Approach D rejected — a `COMBO_DRAIN` statup from any source must work). If
  the melee-path read cost is ever judged too high, Approach D is the lever and
  it is a PRD change, not a design change.
- **No version branch** (FR-5): no `MajorVersion`/`MajorAtLeast`/`IsRegion` in
  the diff, no template edit, no matrix cell promoted. Precedent:
  `character_attack_combo.go:37`/`:173` gates Aran combo orbs on learned
  skills, not version; and task-217's Aran combo counter
  (`character_aran_combo.go`) resolved its one version-varying value as tenant
  config (`idleResetMs`) rather than a major-version branch.
- **One heal per attack** from the plain total over all monsters and hit lines
  — Cosmic's per-monster running-total over-heal is explicitly NOT replicated.
- **Overflow discipline**: sum in `uint64`; early-saturate when
  `totalDamage >= MaxInt16*100`; clamp to `MaxInt16` before narrowing to
  `int16`. No emission when heal `<= 0`.
- **First non-expired match wins** when multiple buffs carry the stat (mirrors
  `hasBuff`). Reflected entries' damage lines still count toward the total.
- **Merge hygiene**: sibling tasks edit the same TODO block in their own
  worktrees — replace exactly the one `// TODO Combo Drain` line in place,
  touch nothing adjacent.

## Discoveries from the rebase (corrections to earlier assumptions)

- **v1's "Aran arrived in GMS v84" was wrong.** Per the WZ snapshot, `21100005`
  and jobs 2100/2110/2111/2112 are present from `gms_79_1` onward.
- **There is no `CharacterEnergyAttackHandle`.** `AttackTypeEnergy` is produced
  by `CharacterTouchAttackHandle` (`character_attack_touch.go:18`), so the
  "energy attack" acceptance criterion is exercised through the touch path.
  `gms_48_1` routes no touch handler (matrix has `TOUCH_MONSTER_ATTACK` as
  `n-a` for `gms_v48`), which is consistent and irrelevant here — v48 has no
  Aran.
- **`buff.NewBuff` gained a seventh parameter** (`noExpiry bool`). Every test
  constructor in the earlier plan was stale.
- **`processAttack` grew several siblings** since v1: `drainTryHeal`,
  `pickPocketResolveState`/`pickPocketTryProc`, `beaconTryApply`,
  `mortalBlowTryProc`, Sacrifice, `comboOrbTryUpdate`, and — after the
  2026-08-13 merge — `aranComboRefreshEligibility` (task-217),
  `energyChargeTryUpdate` (task-216) and `attackCastTryApply`. The TODO block
  moved from line 420 to 991 to 1117. **None of the post-v2 additions reads
  buffs in the attack path**, so the one-read ceiling is still set by the
  projectile gate and Pick Pocket alone.
- **`21100005` is not in `docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv`**,
  so it is not a version-divergent id — but this feature compares no skill id
  anyway, so `tools/skill-job-id-guard.sh` is not engaged either way.
- **Projectile tests exercise only pure helpers** (`computeCount`,
  `resolvePlan`, `hasBuff`, `requiredClassification`) — nothing constructs
  `ProjectileProcessorImpl` or calls `Plan`. Re-verify with
  `grep -rn "ProjectileProcessorImpl\|\.Plan(" services/atlas-channel/...`
  before assuming zero test edits; this was true at rebase time.

## Dependencies

- No cross-task/service dependencies. atlas-data already emits the
  `COMBO_DRAIN` statup; atlas-buffs REST read and atlas-character `ChangeHP`
  consumer are used as-is.
- Task order matters: Task 3 (wiring) needs Task 2's `comboDrainTryProc`, which
  needs Task 1's helpers. Task 4 (version scope) and Task 5 (gates) run last.

## Verification gates

From `services/atlas-channel/atlas.com/channel`:
`go build ./... && go vet ./... && go test -race ./...` clean.
From the worktree root: `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh`,
`tools/lint.sh --check` (needs nvm 22 on PATH) all clean; and
`docker buildx bake atlas-channel` succeeds (mandatory despite untouched
`go.mod`).
Version-scope gates: no `MajorVersion`/`MajorAtLeast`/`IsRegion` in added
lines; `git diff --stat main` empty for
`services/atlas-configurations/seed-data/templates/`, `docs/packets/` and
`libs/atlas-packet/`.
`grep -rn "TODO Combo Drain"` over `services/` and `docs/TODO.md` must be
empty; `docs/TODO.md:151` checked.
Then `superpowers:requesting-code-review` before any PR (project rule).
