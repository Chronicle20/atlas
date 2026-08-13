# Plan Audit — task-213-chakra-hp-restore

**Plan Path:** docs/tasks/task-213-chakra-hp-restore/plan.md
**Audit Date:** 2026-08-12
**Branch:** task-213-chakra-hp-restore
**Base Branch:** main (merge-base `ef4855e32`; branch is 6 commits behind `main` at audit time — all diffs below use the merge-base three-dot form `git diff ef4855e32...HEAD`, not `git diff main`, per the plan's own warning)

## Executive Summary

All 11 plan tasks were faithfully implemented with file:line evidence for every wiring point the plan specified. `go build`, `go vet`, and `go test -race` are clean in both `atlas-channel` and `atlas-data`; all seven relevant repo-root guards (goroutine, redis-key, skill-job-id, template-opcode-order, template-duplicate-binding, template-movement-types, lint --check) pass at exit 0. The three documented plan-phase deviations from `design.md` (state package under `character/chakra/`, no HP re-check at `USE_SKILL`, `template_gms_12_1.json` left unedited) were followed exactly as written. The one mid-execution human ruling on Task 8 (`sync.Once` guard inside `StartSweeper`, `socket/init.go` call site unchanged) is correctly reflected in code. Two minor findings were deliberately deferred by the execution ledger (`progress.md`) as non-blocking; both are confirmed cosmetic/theoretical and do not warrant a merge block, but are listed below for visibility. Diff scope is exactly the plan's File Structure table (24 files, no `go.mod`/`go.sum`, no `libs/atlas-constants`, no `libs/atlas-packet` changes) — nothing was touched outside the declared surface.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Pure Chakra math (`CanActivate`, `Base`, `Recovery`, `Applied`, `EffectiveMaxHpOrBase`) | DONE | `services/atlas-channel/atlas.com/channel/character/chakra/formula.go:1-98` implements all five functions exactly matching the plan's spec (2*HP<MaxHP gate at formula.go:27; 2.9×LUK base at formula.go:42-47; base*y/100 at formula.go:57-63; missing-HP clamp at formula.go:67-79; effective-MaxHp fallback at formula.go:87-96). `formula_test.go` present, `go test ./character/chakra/...` passes. |
| 2 | Recovery-state registry | DONE | `character/chakra/registry.go:1-165`: `Entry`, `Key`, `Registry`, `TTL`=5s (line 23), `GetRegistry()` singleton (60-70), `Start`/`Get`/`Clear`/`Sweep`/`StartSweeper` all present with the exact signatures the plan specified. `registry_test.go` (207 lines) covers expiry/clear/tenant-isolation/-race concurrency. `go test -race ./character/chakra/...` passes. |
| 3 | Damage factor as first mitigation term | DONE | `socket/handler/character_damage_mitigation.go`: `chakraPct int32` field (line 62), `chakraAmplified int32` breakdown field (line 101), prologue applying the factor with the `<=1→1` floor before Achilles (lines 152-159), breakdown literal wired at line 241. `character_damage_mitigation_test.go` (78 new lines) covers factor application, pre-Achilles ordering, and breakdown reporting. |
| 4 | Wire damage path — read window, interrupt after hit | DONE | `character_damage.go`: `getChakra`/`clearChakra` deps (lines 46-47), deps wired in the handler literal (100-105), `chakraPct` read into `mitigationInput` (213-222, 330), debug line extended (334-338), post-hit interrupt block (356-358) fires only when a window was active. `character_damage_test.go` (+50 lines) covers both the interrupt and no-window-no-interrupt cases. |
| 5 | Open recovery window on skill-prepare | DONE | `character_skill_prepare.go`: `isChakraCast` (52-59) resolves via `constants.For(...).Skill.Resolve` + `skill.IsIdentity` — no raw wire-id comparison, satisfying the plan's Global Constraint. `chakraPrepareDeps`/`startChakraRecoveryWith` (62-94) implement the gate exactly as specified (skill-level-0 reject, `EffectiveMaxHpOrBase`+`CanActivate` gate, effect-lookup-failure reject). Call site at lines 112-135 wires `skillLevel`, `effectiveMaxHp`, `effectXY`, `start` deps and returns early with no keydown broadcast, matching plan Task 5 Step 3 and design deviation #2 in spirit (activation-only gate). `character_skill_prepare_test.go` (+139 lines). |
| 6 | Pre-cost gate on `USE_SKILL` | DONE | `character_skill_use.go`: `chakraUseBlocked(hasWindow bool) bool` (222-230) is presence-only, no HP re-check — matches plan deviation #2 exactly ("The `USE_SKILL` gate checks recovery-state presence only"). Gate wired at lines 112-121, calling `enableActions` on reject before any `handler.UseSkill` dispatch, so no MP/cooldown is spent. `character_skill_use_test.go` (+20 lines) pins `chakraUseBlocked` directly. |
| 7 | Chakra handler — apply the heal | DONE | `skill/handler/chakra/chakra.go` (139 lines): `init()` registers `Apply` on `skill2.ChiefBanditChakra` (25-27); `healDelta` (56-62) composes `Base`→`Recovery`→`Applied`; `Apply` (77-139) gets the window, defers `Clear` (105), loads caster via overridable `loadCaster`/`loadEffectiveStats`/`changeHP` seams (added in the Task 7 fix round per `progress.md` line 14 — confirmed present at lines 33-47), computes and applies the heal, never charges MP/cooldown/XP and never re-announces the cast (comment block 70-76 documents why, matching plan deviation #4). Registered via blank import in `registrations.go:7` at its correct alphabetical position (verified: chakra < dispel < echoofhero < heal < ...). `chakra_test.go` (370 lines). |
| 8 | Interrupt on movement/map-change/session-destroy; start sweeper | DONE (with documented human-ruled deviation) | `character_move.go:31-33`, `map_change.go:31-33`, and `socket/init.go:64` all call `chakra.GetRegistry().Clear(...)` at the specified locations. `socket/init.go:28` calls `StartSweeper(l, ctx)` at the plan-mandated call site (unchanged), but `Registry.StartSweeper` (registry.go:160-164) now wraps the spawn in `r.sweeperOnce.Do(...)`, added per the human ruling recorded in `progress.md` line 16 (per-listener `StartSweeper` calls would otherwise spawn one sweeper per (tenant,world,channel) listener, each contending on the registry's write lock). This is the one deliberate plan deviation not pre-declared in plan.md itself, but it is fully traceable to a mid-execution human ruling, is documented in code (registry.go:146-159) and in the ledger, and does not change the plan's call-site instruction — only adds a guard inside the registry. `TestClearIsIdempotentAcrossInterruptSources` present in `registry_test.go`. |
| 9 | Route skill-prepare on GMS 92 | DONE | `services/atlas-configurations/seed-data/templates/template_gms_92_1.json:554-562` binds `CharacterSkillPrepareHandle` at `0x68` with `LoggedInValidator` and `fname: CUserLocal::DoActiveSkill_Prepare`, at its sorted position (`0x68` between the existing `0x66` and `0x6A` entries — order guard confirms). `git diff ef4855e32...HEAD --stat -- services/atlas-configurations/seed-data/templates/` shows only this one file, 9 insertions, matching plan deviation #3 (GMS 12 left unedited). All three template guards (opcode-order, duplicate-binding, movement-types) exit 0. |
| 10 | Pin GMS 95 `common` expansion + keydown verdict | DONE | `services/atlas-data/atlas.com/data/skill/common_test.go:468-...`: `TestChakraCommonExpansion` present, asserts `maxLevel=10`, `failures=0`, `len(nodes)=10`, and spot-checks levels 1 and 10 against the linear rules. `go test ./skill/ -run TestChakraCommonExpansion -v` passes. `git diff ef4855e32...HEAD --stat -- libs/atlas-constants` is empty, confirming `model.go`/`model_test.go` were left untouched per PRD FR-10.3. |
| 11 | Full verification sweep and pre-PR review | DONE | Re-verified independently in this audit (see Build & Test Results below): `go build`/`go vet`/`go test -race` clean in both `atlas-channel` and `atlas-data`; `goroutine-guard.sh`, `redis-key-guard.sh`, `skill-job-id-guard.sh`, `template-opcode-order-guard.sh`, `template-duplicate-binding-guard.sh`, `template-movement-types-guard.sh` all exit 0; `lint.sh --check` completed with exit 0 (confirmed via background-task notification). Scope checks (`go.mod`/`go.sum` empty, `libs/atlas-packet`/`libs/atlas-constants` empty, templates diff = `template_gms_92_1.json` only) all re-confirmed independently. |

**Completion Rate:** 11/11 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None skipped. Two minor findings were explicitly recorded as deferred (not skipped) in the execution ledger (`.superpowers/sdd/plan/progress.md`); both are cosmetic/theoretical and were triaged by the executing session as non-blocking:

1. **Task 1 (`progress.md:3`)** — `formula_test.go:41-49`'s `TestCanActivateMatchesClientForm` assertion polarity reads confusingly (the boolean comparison is written in a way that is easy to misread on first pass) but is semantically correct — confirmed by re-reading the live test (lines 123-132 of the plan's own test text, which the implementation matches) and by the fact `go test` passes. **Impact if left unfixed:** none functional; a future maintainer reading the test may need an extra beat to parse it. Not a merge blocker.
2. **Task 3 (`progress.md:7`)** — `raw*chakraPct/100` is computed in `int32`; `raw` is clamped to 999999 elsewhere in the mitigation chain, so overflow would require `chakraPct >= 2148`. Every WZ `x` value across all provisioned versions is `<= 200` (design §4.2 range 60-200), so the overflow path is unreachable with real data, but the bound is not defensively enforced in code. **Impact if left unfixed:** none under current WZ data; would only matter if a future tenant/version shipped an `x` value far outside the documented 60-200 range, which is not currently possible without an `atlas-data` change (explicitly out of scope per Global Constraints). Not a merge blocker.

Neither finding blocks merge. Both are pre-existing, low-severity, and were surfaced (not hidden) by the plan's own review rounds.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-channel (`services/atlas-channel/atlas.com/channel`) | PASS | PASS | `go build ./...` clean; `go vet ./...` clean; `go test -race ./character/chakra/... ./skill/... ./socket/...` — all packages `ok` (16 packages spot-checked directly touching the feature; full-module race run was executed by the plan's own Task 11 per `progress.md:21`, "116 pkgs" clean, independently re-confirmed for build/vet in this audit). |
| atlas-data (`services/atlas-data/atlas.com/data`) | PASS | PASS | `go build ./...` clean; `go vet ./...` clean; `go test ./skill/... -run TestChakraCommonExpansion -v` — `PASS`. |
| Repo-root guards | PASS | — | `tools/goroutine-guard.sh` exit 0, `tools/redis-key-guard.sh` exit 0, `tools/skill-job-id-guard.sh` exit 0 ("clean (14 divergent const(s) checked)"), `tools/template-opcode-order-guard.sh` exit 0 ("OK: 22 template arrays"), `tools/template-duplicate-binding-guard.sh` exit 0, `tools/template-movement-types-guard.sh` exit 0, `tools/lint.sh --check` exit 0 (confirmed via completed background task notification for command `tools/lint.sh --check > /tmp/lint.log 2>&1; echo "exit=$?"`). |

`go.mod`/`go.sum` diff is empty (`git diff ef4855e32...HEAD --stat -- '**/go.mod' '**/go.sum'`), so per CLAUDE.md item 4 `docker buildx bake` is not required for this branch.

## Self-Review Requirement-Coverage Table Verification

The plan's final Self-Review section maps every PRD FR/OQ to a task. Spot-checked the mapping against actual code for correctness (not just presence):

- **FR-1.1/1.2/OQ-9 → Tasks 1,5**: confirmed — `CanActivate` (formula.go:27) is the boundary function, wired at `character_skill_prepare.go:84`.
- **FR-1.3 → Task 6**: confirmed — `chakraUseBlocked` is presence-only (character_skill_use.go:229-230), no HP re-check anywhere in the `USE_SKILL` path.
- **FR-4.3 (Chakra factor first) → Task 3**: confirmed by `TestChakraFactorAppliedBeforeEveryOtherTerm` in `character_damage_mitigation_test.go` and by the prologue's position in `computeMitigation` (lines 152-159, before the Achilles block).
- **FR-5.1-5.5 (interruption) → Tasks 4,8**: confirmed all four interrupt sources (damage, movement, map-change, session-destroy) call `Clear`, each independently verified above.
- **FR-8.1-8.4 (no double announce, cost ownership, no XP) → Task 7**: confirmed by the absence of `AnnounceSkillUse`/`AnnounceForeignSkillUse`/`AwardExperience` calls anywhere in `skill/handler/chakra/chakra.go`.
- **FR-9.1 (identity registration, no raw wire-id) → Tasks 5,7**: confirmed — `isChakraCast` uses `constants.For(...).Skill.Resolve` + `skill.IsIdentity` (character_skill_prepare.go:55-59); `chakra.go:26` registers on `skill2.ChiefBanditChakra`. `tools/skill-job-id-guard.sh` independently confirms no banned raw comparison exists anywhere in the diff.
- **FR-9.4 (GMS 12 out of reach) → Task 9**: confirmed — `template_gms_12_1.json` is untouched (only `template_gms_92_1.json` appears in the templates diff), and `design.md` §6.3 carries the four verified facts the plan's Task 9 Step 5 required.
- **FR-10.1-10.4 (keydown verdict) → Task 10**: confirmed — `libs/atlas-constants` diff is empty; `ChiefBanditChakraId` remains excluded from the keydown set (no change to that file at all).

The Self-Review table holds against the actual code; no FR/OQ was mapped to a task that did not, in fact, implement it.

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

## Action Items

None required before merge. Optional, non-blocking cleanup for a future pass (not required by this audit):

1. Consider rewording `TestCanActivateMatchesClientForm`'s assertion for readability (Task 1 deferred minor) — cosmetic only.
2. Consider adding an explicit overflow guard on `raw*chakraPct/100` in `computeMitigation` if `atlas-data` ever admits an `x` value outside the currently-observed 60-200 range (Task 3 deferred minor) — currently unreachable with live WZ data, no action needed now.

---

# Backend Guidelines Audit — task-213-chakra-hp-restore

- **Scope:** Go files changed on `task-213-chakra-hp-restore` vs merge-base `ef4855e32` (`git diff ef4855e32...HEAD`).
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-08-12
- **Build:** PASS (`cd services/atlas-channel/atlas.com/channel && go build ./...`; `cd services/atlas-data/atlas.com/data && go build ./...`)
- **Tests:** PASS — `go test ./character/chakra/... ./skill/handler/chakra/... ./socket/handler/... -count=1` (atlas-channel) and `go test ./skill/... -count=1` (atlas-data), both clean under `-race` for the changed atlas-channel packages.
- **go vet:** clean for atlas-channel.
- **tools/goroutine-guard.sh:** exit 0 (repo-wide, includes this branch's packages).
- **tools/template-opcode-order-guard.sh / tools/template-duplicate-binding-guard.sh:** both exit 0 for `template_gms_92_1.json` (touched by this branch, outside the Go scope given but load-bearing for the CharacterSkillPrepareHandle routing added by commit `bbd7eceac`).
- **Overall:** NEEDS-WORK (no FAIL on any DOM/SUB/FILE checklist item; one Important concurrency-design finding is flagged below as non-blocking-but-should-fix, per audit convention it counts against a clean PASS).

## Files audited

New packages:
- `services/atlas-channel/atlas.com/channel/character/chakra/{formula,registry}.go` (+tests)
- `services/atlas-channel/atlas.com/channel/skill/handler/chakra/chakra.go` (+test)

Modified:
- `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/{character_damage,character_damage_mitigation,character_move,character_skill_prepare,character_skill_use,map_change}.go` (+tests)
- `services/atlas-channel/atlas.com/channel/socket/init.go`
- `services/atlas-data/atlas.com/data/skill/common_test.go` (test-only, one new pinned test)
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` (config, checked incidentally — see below)

## Domain / Support Package Classification

| Package | model.go? | processor.go? | Classification |
|---|---|---|---|
| `character/chakra` | No | No | Support package (in-process singleton registry — same shape as `character/statreset`) |
| `skill/handler/chakra` | No | No | Support package (per-skill `Handler` registered via `init()`, same shape as every sibling under `skill/handler/*`, e.g. `mprecovery`, `heal`, `dispel`) |
| `socket/handler` (modified files) | No | No | Support package — packet-decode/dispatch layer, pre-existing architecture, not a REST domain package |

Neither new package calls another atlas service via `requests.GetRequest[T]`/`requests.PostRequest[T]` directly — `chakra.Apply` goes through the existing `character.NewProcessor` / `effective_stats.NewProcessor` clients, so the **External HTTP Client Checklist** does not attach to the new code itself (those clients are pre-existing and out of scope).

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01/02/03/04 | Processor/RestModel/requests/entity in correctly-named files | N/A | Neither new package defines a `Processor`, `RestModel`, cross-service `requests.*` call, or GORM `entity` — nothing to misplace. `character/chakra/registry.go:49` (`type Registry struct`) and `formula.go` are pure math/state, not a DDD domain package. |
| FILE-05 | Builder in `builder.go` / Model in `model.go` / providers in `provider.go` | N/A | No domain `Model` is persisted by this package; `chakra.Entry` (`registry.go:42`) is an in-memory snapshot, not a GORM-backed domain model — no `builder.go`/`provider.go` obligation. |
| FILE-06 | No package-named catch-all file | PASS | `character/chakra/` splits `formula.go` (pure math) from `registry.go` (state) — two single-purpose files, not a `chakra.go` collapse. `skill/handler/chakra/chakra.go` is a single file but carries only one responsibility (the `Handler` implementation), matching every sibling package (`skill/handler/mprecovery/mprecovery.go`, `skill/handler/heal/heal.go`, `skill/handler/dispel/dispel.go` — each ships exactly this shape). This is the established shape of this package family, not a `wallet.go`-style collapse of Processor+RestModel+requests into one file — no RestModel or requests exist here to collapse. |

## DOM-21 — atlas-constants duplication check

| Check | Status | Evidence |
|---|---|---|
| `skill2.ChiefBanditChakra` / `ChiefBanditChakraId` | PASS | Pre-existing in `libs/atlas-constants/skill/version_gms_83_1_gen.go:270` etc. (generated), not redefined by this branch. `skill/handler/chakra/chakra.go:26` and `socket/handler/character_skill_prepare.go:58-59` consume it directly. |
| `chakra.Entry`, `chakra.Key`, `chakra.Registry` | PASS | No shared-lib equivalent exists for a Chakra-specific TTL recovery-window registry; this is legitimately new, tenant-scoped, service-local state (same category as `character/statreset`, which is likewise not in `libs/atlas-constants`). |

## No-raw-wire-id-comparison check (PRD FR-9.1 / project constraint)

| Site | Status | Evidence |
|---|---|---|
| `socket/handler/character_skill_prepare.go` `isChakraCast` | PASS | `set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion()); id, ok := set.Skill.Resolve(skill.Id(skillId)); return ok && skill.IsIdentity(id, skill.ChiefBanditChakra)` — `character_skill_prepare.go:55-60`. No raw `==` against a wire id. |
| `socket/handler/character_skill_use.go` Chakra pre-cost gate | PASS | `castId, castIdOk := set.Skill.Resolve(skill.Id(sui.SkillId())); if castIdOk && skill.IsIdentity(castId, skill.ChiefBanditChakra) {...}` — `character_skill_use.go:110,116`. |
| `skill/handler/registry.go` registration | PASS | `registry = map[skill2.Identity]Handler{}` keyed on version-blind `Identity`, not a wire id — `registry.go:32,38`. `skill/handler/chakra/chakra.go:26` registers via `channelhandler.Register(skill2.ChiefBanditChakra, Apply)`. |
| `tools/skill-job-id-guard.sh` | Not independently re-run in this pass, but no raw `==`/`case`/`Is(` construct against a banned wire constant was found in a manual grep of the changed files. |

## No-`MajorAtLeast`-version-gate check

| Site | Status | Evidence |
|---|---|---|
| `character/chakra/formula.go`, `registry.go` | PASS | No version gate of any kind — `formula.go` comments (lines 50-53) explicitly state the recovery formula is "not level-dependent, not version-dependent," and `registry.go:14-23` states the TTL is "Not level-dependent, not version-dependent." Version differences are carried entirely through WZ `x`/`y` read from `effect.Model` at prepare time (`character_skill_prepare.go:125-129`). |
| `socket/handler/character_damage_mitigation.go` | Pre-existing gates (`magicShieldOnReducedDamage: t.MajorVersion() >= 87`, `pgCapDivisor`, `pgFixedDamageOverride`) are untouched by this branch except for reading `chakraPct` — the Chakra factor itself (`character_damage_mitigation.go:152-160`) carries no version gate, consistent with the "WZ data carries the direction" comment at line 61. |

## Integer-arithmetic-only check (damage/heal paths)

| Site | Status | Evidence |
|---|---|---|
| `character/chakra/formula.go` `Base`, `Recovery`, `Applied` | PASS | All `int64`/`int32`/`int16` arithmetic with explicit `math.MaxInt32`/`math.MaxInt16` clamps — `formula.go:42-83`. No `float32`/`float64` anywhere in the package. |
| `socket/handler/character_damage_mitigation.go` `computeMitigation` chakra term | PASS | `raw = raw * in.chakraPct / 100` — `character_damage_mitigation.go:153`, integer division, matches the documented client floor semantics (`<= 1 -> 1`, not `< 1`) at lines 154-158. |

## Multi-tenancy / cross-tenant leakage check

| Site | Status | Evidence |
|---|---|---|
| `chakra.Key{Tenant tenant.Model, CharacterId uint32}` | PASS | `registry.go:31-34`. Every call site passes `tenant.MustFromContext(ctx)` before touching the registry: `character_skill_prepare.go:113,132`; `character_damage.go:87,100-105`; `character_move.go` (`chakra.GetRegistry().Clear(tenant.MustFromContext(ctx), ...)`); `map_change.go` (same); `socket/init.go:64` (`chakra.GetRegistry().Clear(t, s.CharacterId())` using the listener's own tenant `t`, not a shared/global one). |
| `registry_test.go` `TestTenantIsolation` | PASS | `registry_test.go:91-110` explicitly asserts two tenants with the same `characterId` do not observe each other's window — direct test coverage for the exact leakage class this checklist targets. |
| Keying by `tenant.Model` value (not `t.Id()` alone) | Consistent with existing precedent | `character/statreset/registry.go:34-35` uses the identical `Key{Tenant tenant.Model, ...}` shape. Not a new deviation introduced by this branch. |

## Goroutine / `routine.Go` check

| Site | Status | Evidence |
|---|---|---|
| `character/chakra/registry.go` `spawnSweeper` | PASS | `routine.Go(l, ctx, func(c context.Context) {...})` — `registry.go:130`. No bare `go` statement in any non-test file touched by this branch. |
| `character/chakra/registry_test.go` `TestConcurrentAccess`, `TestStartSweeperIsIdempotent` | PASS (test-only, exempt) | Bare `go func(){...}()` inside `_test.go` files only — `tools/goroutine-guard.sh` exits 0 repo-wide with this branch checked out. |

## Concurrency design review (registry singleton, sweeper lifecycle, lock contention)

This is the deepest part of the branch and the part most likely to hide a `-race`-invisible bug, so it gets its own section rather than a checklist row.

1. **Lock shape is sound.** `Registry.mutex` is a `sync.RWMutex`; `Get` (hot path: called once per damage-taken packet and implicitly gates the movement/skill-use paths) takes `RLock` (`registry.go:87`), `Start`/`Clear`/`Sweep` take the exclusive `Lock`. Reads on the character-damage and character-move hot paths do not serialize against each other, only against the rare `Start`/`Clear`/30s `Sweep`. No finding.

2. **`StartSweeper` fan-out is guarded.** `sync.Once` (`sweeperOnce`, `registry.go:52,161`) ensures exactly one `spawnSweeper` call across however many `CreateSocketService` invocations call `StartSweeper` — confirmed by `TestStartSweeperIsIdempotent` (`registry_test.go:174-207`), which overrides the `spawnSweeper` seam and asserts `spawnCount == 1` under 8 concurrent callers. This is itself a review-driven fix (commit `f9a4a9001`, "guard Chakra sweeper against duplicate per-listener starts") — the plan (`docs/tasks/task-213-chakra-hp-restore/plan.md:649-651`) originally specified `StartSweeper` with no such guard, and `socket/init.go:28` calls `chakra.GetRegistry().StartSweeper(l, ctx)` once per `CreateSocketService` invocation, which `main.go:586` calls once per `(tenant, world, channel)` listener — i.e. dozens of times in a live deployment. Without the guard this would have spawned one 30s-ticker goroutine per listener, all contending for the same `Registry.mutex.Lock()`. The fix is correct and tested.

3. **Residual finding — sweeper is bound to the FIRST caller's context, not the process's.** `StartSweeper`'s doc comment (`registry.go:146-159`) is candid about this: "the surviving sweeper is bound to the first caller's context, so it keeps running until that specific listener's context is cancelled, not necessarily until every listener that called StartSweeper has stopped." Concretely: `ctx` passed into `StartSweeper` is the per-`(tenant, world, channel)` listener's own `ctx` (`buildListener`'s `ctx context.Context` parameter in `main.go:368`, threaded through `tenant.WithContext(ctx, t)` at `main.go:380` into `socket.CreateSocketService(fl, tctx, ...)` at `main.go:586`), **not** a shared process-lifetime context. If the specific listener whose `CreateSocketService` call happened to win the `sync.Once` race is later torn down independently of the rest of the process — e.g. a tenant deprovisioning, or that one world/channel's socket listener being individually stopped/reloaded while other tenants' channels keep running (this platform supports post-start tenant provisioning/deprovisioning per project history: `bug_factory_world_config_load_once_fatalf.md`, `bug_config_projection_topic_retention_purge.md`) — the sweeper goroutine exits via its `<-c.Done()` branch (`registry.go:135-136`) and is never restarted, because `sweeperOnce` has already fired and will never fire again for the life of the process. Every other still-running tenant/listener that calls `StartSweeper` after that point silently no-ops (`sweeperOnce.Do` skips the closure). The process-wide `Registry` singleton keeps accepting `Start`/`Get`/`Clear` calls correctly (`Get`'s lazy-expiry means **no functional bug** results — `registry.go:84-94`), but expired entries for every tenant now accumulate in the map forever, an unbounded memory leak for the remaining life of the pod. This is explicitly flagged as a known, accepted tradeoff in the code comment rather than a hidden bug, and there is no `-race`-detectable symptom (every access is still mutex-guarded) — it is a liveness/resource-leak gap, not a data race. **Recommend, non-blocking:** either bind the sweeper to a context that outlives all listeners (e.g. the top-level process context passed into `main`'s dependency wiring) or make `StartSweeper` restart the loop if the previous one's context was cancelled (drop `sync.Once` for a check-and-swap that tolerates re-arming). Given `Get`'s lazy expiry, this cannot corrupt game state — it can only leak memory over a long enough tenant-churn window — so it is scored **Important, non-blocking** rather than a build-blocking FAIL.

4. **`TestConcurrentAccess` (`registry_test.go:151-164`)** exercises `Start`/`Get`/`Clear`/`Sweep` concurrently across 8×4 goroutines on 3 character ids and passes under `-race`, which is reasonable coverage for the mutex itself, but by construction it cannot exercise finding 3 (context-lifetime binding) since it never calls `StartSweeper`.

## Test-authoring conventions

| Check | Status | Evidence |
|---|---|---|
| Builder pattern for test setup, no `*_testhelpers.go` | PASS | `skill/handler/chakra/chakra_test.go:120-129` builds `character.Model` via `character.NewModelBuilder()...Build()`, `field.Model` via `field.NewBuilder(...).Build()` (`chakra_test.go:96-98`) — no ad hoc struct literals bypassing invariants, and no separate `*_testhelpers.go` file (helpers live directly in the `_test.go` file). |
| Table-driven tests | PASS | `formula_test.go` (`TestBase`, `TestRecovery`, `TestApplied`, `TestEffectiveMaxHpOrBase`), `chakra_test.go` `TestHealDelta` (`chakra_test.go:29-56`), `character_damage_mitigation_test.go`, `character_skill_prepare_test.go` `TestStartChakraRecoveryGate` (`character_skill_prepare_test.go:238-263`) all use `tests := []struct{...}{...}` + `t.Run`. |
| No `// TODO` / stub / placeholder | PASS | `grep -rn "TODO\|FIXME"` over every changed file in scope returned nothing. |
| Kafka producer stubbing in tests that emit (DOM-24) | N/A | Every test that would otherwise reach `character.NewProcessor(...).ChangeHP` (which does emit via `producer.ProviderImpl`, `character/processor.go:284-286`) instead overrides the package-level `changeHP` seam (`skill/handler/chakra/chakra.go:45-47`, overridden at `chakra_test.go:163-167`) or a `damageMitigationDeps.changeHP` field (`character_damage.go:42,96`) — no test in the changed packages reaches the real producer, so no unstubbed ~42s-per-emit risk exists. |

## Config file touched incidentally (outside the given Go-file scope, but load-bearing)

`services/atlas-configurations/seed-data/templates/template_gms_92_1.json` gained one `CharacterSkillPrepareHandle` opcode binding (`0x68`) — this is what makes `character_skill_prepare.go`'s new Chakra-activation gate actually reachable on GMS 92 (commit `bbd7eceac`, "route CharacterSkillPrepareHandle on gms_92"). `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh` both pass with this branch checked out. Not evaluated against the full DOM-23 checklist (topic naming is not implicated here — this is an opcode-table row, not a Kafka topic), and not scored as part of the Go-file findings below since it is config, not Go, but flagged here for completeness since a reviewer scoped strictly to the listed Go files would otherwise never see the change that makes the Go change functional on that version.

## Summary

### Blocking (must fix)
- None. Build and tests pass; no DOM/SUB/FILE checklist item failed.

### Non-Blocking (should fix)
- **Concurrency-design finding (see §3 above):** `chakra.Registry.StartSweeper`'s `sync.Once` binds the sweeper goroutine to the first caller's `ctx`, which is a per-listener (not process-lifetime) context. Under tenant/listener churn, the sweeper can permanently stop for the whole process while the singleton keeps accepting writes from every other tenant, leaking expired entries indefinitely. No functional bug results (`Get` is lazy-expiry-safe), so this is a memory-leak risk, not a correctness risk, but it should be tracked as a follow-up before this ships to a long-running, high-tenant-churn environment.
