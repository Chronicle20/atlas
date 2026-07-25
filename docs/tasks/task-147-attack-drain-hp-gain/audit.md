# task-147 — Plan Adherence Audit

Branch: `task-147-attack-drain-hp-gain`
Range audited: `01a0a3bb0..59f20e2df` (5 doc commits + 5 implementation commits)
Date: 2026-07-25

**Verdict: the plan was faithfully executed.** All 5 tasks implemented, all PRD §10
acceptance criteria met, all negative requirements honored. No blocking findings.

> Authorship note: this audit was produced by the controller after the dispatched
> `plan-adherence-reviewer` stalled three times waiting on a lint monitor without
> producing a report. Every claim below is backed by a command run against this
> worktree. The companion `audit-backend.md` is the `backend-guidelines-reviewer`'s
> own output (verdict: PASS).

## 1. Task-by-task evidence

Production file throughout: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go`
Test file throughout: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_drain_test.go`

| Plan task | Commit | Evidence | Status |
|---|---|---|---|
| 1 — `isDrainSkill` membership helper | `e4c887bcc` | `character_attack_common.go:83`; test `character_attack_drain_test.go:24` | ✅ implemented |
| 2 — `drainHealAmount` pure cap math | `9d2aaab20` | `character_attack_common.go:235`; table test `:47` | ✅ implemented |
| 3 — widen `onDamageApplied`, rename loader | `1ad342f0f` | field `:106` `loadEffectiveStats`, field `:111` `onDamageApplied func(monsterId uint32, totalDamage uint32)`, invocation `:205`, closure `:417`; tests `:111`, `:141`, `:160` | ✅ implemented |
| 4 — `drainTryHeal` orchestrator | `31d893f2e` | `character_attack_common.go:323`; flow tests `:207`, `:239`, `:255`, `:276`, `:300` | ✅ implemented |
| 5 — wire-up + TODO removal | `59f20e2df` | gate `character_attack_common.go:448`, call `:449` | ✅ implemented |

Ten test functions exist across the five tasks; none was skipped.

## 2. PRD §10 acceptance criteria

| Criterion | Status | Evidence |
|---|---|---|
| Four skill ids heal; other skills unaffected | ✅ | `isDrainSkill` switch over the four constants (`:83`); gate at `:448` |
| Heal = `min(monsterMaxHp, floor(dmg×X/100), effMaxHp/2)` | ✅ | `drainHealAmount` (`:235`); 15-case table test incl. FR-3 spot X values |
| Multi-target heals per damaged monster, individually capped | ✅ | hook fires per `DamageInfo` (`:205`); `TestDrainTryHeal_PerMonsterCaps` (`:300`) |
| Killed monster still contributes its heal | ✅ | fetch-then-heal order in `drainTryHeal` (`:323`); no liveness check |
| Failures produce no heal and never abort the attack | ✅ | void signature; `TestDrainTryHeal_MonsterFetchError_SkipsHeal` (`:239`), `_ZeroEffectiveStats_SkipsHeal` (`:255`), `_EmitErrorSwallowed` (`:276`) |
| Heal clamped to `int16` before `ChangeHP` | ✅ | `uint64` internals then single clamp (`:235`); pathological-input case in the table |
| Table-driven tests for math/caps/zero/floor | ✅ | `TestDrainHealAmount` (`:47`), 15 subtests |
| Drain TODO removed, adjacent TODOs untouched | ✅ | see §4 |
| No per-version branching introduced (§8.1) | ✅ | see §4 |
| Build/vet/race + guard suite clean | ✅ | see §3 |

## 3. Verification gate

All run from this worktree, to completion:

| Gate | Result |
|---|---|
| `go build ./...` (atlas-channel) | clean |
| `go vet ./...` (atlas-channel) | clean |
| `go test -race ./...` (atlas-channel) | clean, all packages `ok` |
| `tools/redis-key-guard.sh` | exit 0 |
| `tools/goroutine-guard.sh` | exit 0 |
| `tools/lint.sh --check` | `lint.sh: OK` |
| `go.mod` diff over branch range | empty — no `docker buildx bake` required |

`tools/lint.sh --check` reported "6 problems (0 errors, 6 warnings)". All six are in
atlas-ui React files this branch never touched (`ApplyPresetDialog.tsx`,
`CreateTenantDialog.tsx`, `AccountsPage.tsx`, `QuestsPage.tsx`) — the known
pre-existing warning baseline. Zero findings in any Go file. The guard itself passed.

## 4. Negative requirements

- **No version gating.** `grep -n "MajorVersion\|MajorAtLeast"` over both changed files
  returns nothing. Version applicability rests on the skill-ownership check at
  `character_attack_common.go:283-291`, per PRD §8.1.
- **Exactly one TODO removed.** `TODO increase HP from Energy Drain, Vampire, or Drain`
  is gone; the file's TODO count went 24 → 23; `// TODO Combo Drain` survives at
  `character_attack_common.go:515`.
- **`go.mod` untouched** across the branch range.

## 5. Three deviations from the plan's literal text

Each was necessary or quality-driven, applied consistently, and changed no behavior.

1. **`Build()` returns `(Model, error)`.** The plan's test snippets wrote
   `monster.NewModelBuilder(...).Build(), nil`, which does not compile —
   `monster/builder.go:117` declares `Build() (Model, error)`. The two-value form is
   used at all 6 call sites; `grep "Build(), nil"` returns nothing. **Plan defect.**
2. **`testField()` → `testDrainField()`.** The plan's helper name collided with an
   existing `testField(_map.Id)` at `mystic_door_enter_test.go:30` in the same package.
   The drain helper is `testDrainField()` (`character_attack_drain_test.go:104`); the
   colliding name is not redefined. **Plan defect.**
3. **`TestDrainTryHeal_EmitErrorSwallowed` now asserts.** The plan's version ended with
   only `// Reaching here without panic is the assertion.` — a test asserting nothing.
   Controller-directed fix: the failing `changeHP` spy is counted and asserted
   `== 1` (`character_attack_drain_test.go:294-296`), so the test fails if the error
   path stops being exercised rather than passing vacuously. **Plan quality gap.**

## 6. Deferred minors (non-blocking)

Carried from task reviews; none blocks merge.

- The `math.MaxUint32` damage-sum clamp (`character_attack_common.go:202-204`) has no
  direct test — all hook tests use small damage values. Verified by inspection only.
- The `onDamageApplied` closure wiring itself (`:444-449`) is covered only indirectly;
  exercising it end-to-end needs a live session. Every underlying unit is tested.
- `drainHealAmount`'s zero-guard omits `monsterMaxHp`, which is instead handled by the
  cap cascade. Correct and test-covered; a clarifying comment would help future readers.
- The test file passes raw skill-id literals (`4101005`, `14101006`) as `drainTryHeal`'s
  logging-only `skillId` argument where the named constants would read better. Not a
  DOM-21 violation — production code classifies on constants only.

## 7. Open item (documentation, not code)

PRD §8.1 records two matrix cells as **unverified**: whether Knights of Cygnus — and
therefore Thunder Breaker Energy Drain (15111001) and Night Walker Vampire (14101006) —
exist in GMS v72/v79. No legacy WZ dump is available locally, and an IDB string probe
was inconclusive (the control run on v83/v87, where Cygnus definitively exists, also
returned zero hits, so the probe proves nothing). This does not affect code: the
ownership gate makes both outcomes behaviourally identical. It decides only whether
those cells are labelled `applicable` or `n-a`. Settle it with an atlas-data skill-effect
probe for `15111001` under a live v72/v79 tenant.
