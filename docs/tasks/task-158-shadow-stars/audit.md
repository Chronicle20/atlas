# Plan Audit — task-158-shadow-stars

**Plan Path:** docs/tasks/task-158-shadow-stars/plan.md
**Audit Date:** 2026-07-17
**Branch:** task-158-shadow-stars
**Base commit range:** c9490b724..81c108d91 (docs commits `e71370cb8`, `9af1e2a7b`, `c1a5ec9e1` + 6 code commits)

## Plan Adherence

### Executive Summary

All 6 plan tasks were faithfully implemented and are traceable to specific commits with matching file:line evidence. Task 4's code and the plan's Global Constraints statement about `reader.go` being functionally unchanged were both superseded by a human-approved mid-execution fix (commit `67c6e8ec4`) after discovering the plan's `0`-amount premise was factually wrong (`produceBuffStatAmount`'s `if value != 0` guard silently drops the SHADOW_CLAW placeholder). Per the audit brief, this deviation is expected and not scored as a defect — both the append-if-absent statup fix (atlas-channel) and the nonzero-placeholder fix (atlas-data) are present and tested. No `// TODO`, stub, or 501 was introduced by this branch's diff (all pre-existing `TODO`s in touched files predate this work and sit outside the changed lines). Working tree is clean; branch is `task-158-shadow-stars` as expected.

### Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | `SkillUsageInfo.SpiritJavelinItemId()` getter (FR-1) | DONE | `libs/atlas-packet/model/skill_usage_info.go:61-63` adds the getter exactly as specified. Byte-fixture decode test `TestSkillUsageInfoDecodeSpiritJavelinItemId` at `libs/atlas-packet/model/skill_usage_info_test.go:48-71` matches the plan's fixture verbatim, asserting decoded value and zero leftover bytes. Commit `77c4aedec`. |
| 2 | `effect.Model.BulletCount()` getter | DONE | `services/atlas-channel/atlas.com/channel/data/skill/effect/model.go:109-114` adds the getter with doc comment distinguishing it from `BulletConsume()`. Test `TestModelBulletCount` present in `model_test.go`. Commit `d93cb4cfb`. |
| 3 | Projectile gate — claw + `SHADOW_CLAW` carve-out (FR-3) | DONE | `character_attack_projectile.go:209-219` adds `projectileConsumptionSkipped(weaponType, buffs)` exactly per plan; `Plan()` call site at line 107 rewired to use it. Test `TestProjectileConsumptionSkipped` in `character_attack_projectile_test.go` covers all 8 plan cases (bow/crossbow+soulArrow skip, claw+shadowClaw skip, claw+no-buff/expired/soulArrow consume, bow/gun+shadowClaw consume). Commit `2fb37a693`. |
| 4 | Shadow Stars pure helpers (FR-2, FR-4 plan, FR-5) | DONE (amended, approved) | `shadow_stars.go` implements `StarDraw`, `validateShadowStar`, `resolveStarConsume`, `rewriteShadowClawStatups`, `resolveShadowStarsCast` per plan (commit `fb09f86bf`), then amended by commit `67c6e8ec4`: `rewriteShadowClawStatups` (lines 85-100) now appends a SHADOW_CLAW entry when absent (mirrors `mount.go:tamedMountStatups`), and `validateShadowStar` (line 35) uses the shared `item.IsThrowingStar` predicate instead of an inline classification check. New tests `TestRewriteShadowClawStatups_AppendsWhenAbsent` and `TestResolveShadowStarsCast_NoShadowClawInInput` cover the append-when-absent path. This is the human-approved Fix A described in the task brief — not a plan deviation to flag. |
| 5 | Consume emit + wire into `UseSkill` (FR-2, FR-4, FR-5) | DONE | `shadow_stars.go:115-167` adds `loadCasterInventoryFunc` seam and `emitStarConsume`/`reservedStarToConsume` matching the plan's reservation→consume pattern (commit `fb09f86bf`/`67c6e8ec4`). `common.go` wiring (commit `81c108d91`): pre-flight block at top of `UseSkill` (lines 73-96 of the diff) validates/aborts before HP/MP/cooldown exactly as specified, `statupsToApply` replaces `e.StatUps()` at the buff-apply gate (`len(statupsToApply) > 0`, not the plan's literal `len(e.StatUps()) > 0`— correctly adapted since after the Fix-B reader change `e.StatUps()` now also carries a real SHADOW_CLAW entry, so this still gates correctly), and `emitStarConsume` fires after `buff.Apply`. Matches plan's wiring note precisely, including the instruction not to introduce a second `skillId` variable (confirmed — `skill2.Id(info.SkillId())` is used inline, `common.go:99`'s existing `skillId` declaration untouched). |
| 6 | Full verification (CLAUDE.md build gates) | DONE (adapted, approved) | Controller ran all gates directly (not via subagent) per the task brief's note. Evidence: `go build ./...`, `go vet ./...`, `go test -race ./...` all rc=0 across `libs/atlas-packet`, `services/atlas-channel`, `services/atlas-data`; `tools/redis-key-guard.sh` clean; `tools/goroutine-guard.sh` clean (plan omitted this gate — CLAUDE.md gate 6 requires it, correctly added); `docker buildx bake atlas-channel atlas-data` succeeded (plan only listed `atlas-channel`; `atlas-data` correctly added since Fix B touched its `go.mod`-scoped module). Explicit file staging used for the fix commit instead of the plan's `git add -A` (project convention bans `git add -A` — correct substitution). No `go.mod`/`go.sum` changed, consistent with "no new lib" expectation. |

**Completion Rate:** 6/6 tasks (100%)
**Skipped without approval:** 0
**Partial implementations:** 0

### Skipped / Deferred Tasks

None. The only deviations from the plan's literal text (Task 4's code, the "No reader.go functional change" Global Constraint, Task 6's subagent-vs-controller execution and `git add -A`-vs-explicit-staging, and the added `goroutine-guard.sh` gate) are all pre-approved per the audit brief and are documented above as DONE (amended/adapted), not as gaps.

### Self-Review Spec-Coverage Table Verification (plan.md lines 738-753)

| Requirement | Plan's claimed coverage | Verified? |
|---|---|---|
| FR-1 getter + decode test | Task 1 | Yes — `skill_usage_info.go:61-63`, test at `skill_usage_info_test.go:48-71`. |
| FR-2 SHADOW_CLAW amount == chosen star id | Task 4 + 5 | Yes — `rewriteShadowClawStatups` sets amount to `int32(starItemId)` for both override and append paths; wired via `statupsToApply` in `common.go`. Covered by `TestRewriteShadowClawStatups`, `TestRewriteShadowClawStatups_AppendsWhenAbsent`, `TestResolveShadowStarsCast`, `TestResolveShadowStarsCast_NoShadowClawInInput`. |
| FR-3 claw+SHADOW_CLAW skip; inactive-safe regression | Task 3 | Yes — `TestProjectileConsumptionSkipped` includes both the skip case and the "claw + expired shadow claw -> consume" / "claw + no buff -> consume" regression cases. |
| FR-4 charge bulletCount; multi-slot; shortfall posture | Task 2 + 4 + 5 | Yes — `BulletCount()` getter, `resolveStarConsume` (single-slot + multi-slot-with-shortfall tests), `emitStarConsume` reserve→consume wiring in `common.go`. |
| FR-5 validate classification + ownership; warn+abort | Task 4 + 5 | Yes — `validateShadowStar` (now via `item.IsThrowingStar`), abort path in `common.go` (`if !ok { l.Warnf(...); return nil }`) runs before `e.HPConsume()`/`e.MPConsume()`/cooldown — confirmed by reading the pre-flight block placement at the top of `UseSkill`, before the pre-existing HP/MP block. |
| AC: byte-fixture decode test | Task 1 | Yes. |
| AC: buff statup value == star id (not 0) | Task 4 | Yes — and strengthened by the Fix commit's append-when-absent tests, which specifically guard against the regression the plan's original code would have shipped. |
| AC: claw+SHADOW_CLAW -> no consume plan | Task 3 | Yes. |
| AC: claw inactive still consumes (regression) | Task 3 | Yes. |
| AC: consume targets chosen item id + quantity | Task 4 | Yes — `TestResolveStarConsume_SingleSlot`, `TestResolveStarConsume_MultiSlotAndShortfall`. |
| AC: bogus/unowned id rejected + warn, no consume | Task 4 + 5 | Yes — `resolveShadowStarsCast` returns `ok=false` for invalid stars (tested); `common.go` warn-logs and returns before any HP/MP/consume. |
| AC: build/vet/test/bake/redis-guard | Task 6 | Yes, plus goroutine-guard (CLAUDE.md-required, correctly added beyond plan). |

No gaps found in the Self-Review table; all claimed coverage lines up with real code and real tests.

### Build & Test Results

Per the task brief, these gates were already run by the controller with real exit codes (all rc=0) and are not re-run by this audit: `go build ./...`, `go vet ./...`, `go test -race ./...` across `libs/atlas-packet`, `services/atlas-channel`, `services/atlas-data`; `tools/redis-key-guard.sh`; `tools/goroutine-guard.sh`; `docker buildx bake atlas-channel atlas-data`. This audit independently confirms via `git status --porcelain` that the worktree is clean (no stray uncommitted changes from those runs) and via source inspection that the code those gates exercised matches the plan's intended shape.

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-packet | PASS (controller-evidenced) | PASS (controller-evidenced) | Not re-run per instructions; getter + decode test read and confirmed correct. |
| services/atlas-channel | PASS (controller-evidenced) | PASS (controller-evidenced) | Not re-run; all new/changed functions read and confirmed to match tests. |
| services/atlas-data | PASS (controller-evidenced) | PASS (controller-evidenced) | Not re-run; reader.go fix + reader_test.go addition read and confirmed correct, reuses existing `findStatup` helper. |

### Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE

### Action Items

None. All 6 tasks are implemented with matching tests; the one substantive plan defect (the incorrect "no reader.go change" premise and the resulting under-specified Task 4 code) was caught during execution and fixed with an approved belt-and-braces patch (nonzero placeholder in atlas-data + append-if-absent in atlas-channel), with regression tests added on both sides. No TODOs, stubs, or 501s were introduced. No further work identified before merge.

---

## Backend Guidelines (DOM-*)

- **Service Path(s):** `libs/atlas-packet` · `services/atlas-channel/atlas.com/channel` · `services/atlas-data/atlas.com/data`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-17
- **Build/Test:** Not re-run (per audit brief) — controller-evidenced rc=0 for `go build ./...`, `go vet ./...`, `go test -race ./...` across all three modules, plus `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `docker buildx bake atlas-channel atlas-data`.
- **Overall:** PASS (no FAIL findings; two Minor observations, non-blocking)

### Scope Classification

None of the six changed files sit in a "domain package" as `file-responsibilities.md`/DOM-* define it (a package that owns its own persistence — `entity.go` + `administrator.go` + `builder.go` + `provider.go`). Concretely:

| File | Package | Classification | Reasoning |
|---|---|---|---|
| `libs/atlas-packet/model/skill_usage_info.go` | `model` (atlas-packet) | Shared packet-codec model | Wire-format struct with `Decode` + private-field/getter/Builder shape; not a service domain package, file-responsibilities.md's model/entity/processor split doesn't apply. |
| `services/atlas-channel/.../data/skill/effect/model.go` | `effect` | Nested value object | `effect.Model` is a sub-struct of the parent `data/skill.Model` (`services/atlas-channel/atlas.com/channel/data/skill/model.go:12`) — the parent package owns `processor.go`/`requests.go`/`rest.go` (`services/atlas-channel/atlas.com/channel/data/skill/{processor,requests,rest}.go`); `effect` (alongside sibling `point.go`) is a decomposed VO, same shape as `statup/model.go`. Confirmed: `effect/` has no `processor.go`/`requests.go`/`entity.go`/`builder.go` before or after this diff — this is not new debt, it's the established nested-VO pattern. |
| `services/atlas-channel/.../socket/handler/character_attack_projectile.go` | `handler` (socket) | Support / feature-orchestration file | Pre-existing file (this diff only touches lines 107-111, 209-221); package has no `model.go`/`resource.go` at package root — it's a packet-dispatch orchestration package, not a domain package. |
| `services/atlas-channel/.../skill/handler/shadow_stars.go` | `handler` (skill) | Support / feature-orchestration file (NEW) | Same shape as sibling `mount.go` in the same directory — free functions, no `type Processor interface`/`ProcessorImpl`/`NewProcessor(` declared (grep-verified absent), so FILE-01 is not mechanically triggered. |
| `services/atlas-channel/.../skill/handler/common.go` | `handler` (skill) | Support / feature-orchestration file | Pre-existing dispatcher (`UseSkill`), FR-5 pre-flight block inserted at `common.go:73-96`. |
| `services/atlas-data/.../skill/reader.go` | `skill` (atlas-data) | Domain package (has `model.go`? — checked: package has `processor.go`/`resource.go`/`rest.go`/`registry.go`/`string_registry.go`, no `model.go`/`entity.go`/`builder.go`/`administrator.go`) | `reader.go` is a WZ-XML-ingest transform (`Read`/`produceSkill`/`getEffect`), one-way into `RestModel` — not a DB-backed CRUD domain (atlas-data has no local persistence for skills; it's an in-memory/XML-backed registry via `registry.go`). Diff only changes one placeholder value inside a large pre-existing `if/else if` chain (`reader.go:297-304`). |

None of the six files are new "domain packages" introduced by this branch, so the full DOM-01..20 CRUD checklist (builder/entity/administrator/provider/resource) is not applicable to any of them — applying it wholesale would be grading against a shape these packages never had and were never meant to have (REST-mirror / packet-codec / WZ-ingest packages, not owned-persistence domains). This is a scope determination, not an exemption granted for repo-wide prevalence: I checked each package's actual file inventory (`ls`) before classifying it, per the "grade against the table" instruction.

### FILE-* (File Responsibilities) — all six changed files

| ID | Check | File | Status | Evidence |
|---|---|---|---|---|
| FILE-01 | Processor logic in `processor.go` | `shadow_stars.go` | PASS (not triggered) | `grep -n "type Processor interface\|type ProcessorImpl\|func NewProcessor(" shadow_stars.go` → no matches. `emitStarConsume`/`reservedStarToConsume` are free functions calling `compartment.NewProcessor(l, ctx)` (an existing processor from another package), matching sibling `mount.go`'s shape in the same directory. |
| FILE-01 | Processor logic in `processor.go` | `character_attack_projectile.go` | Minor (pre-existing, not introduced by this diff) | `type ProjectileProcessor interface` (line 43), `ProjectileProcessorImpl` (line 48), `func NewProjectileProcessor(` (line 55) live in a file named for the feature, not `processor.go`/`processor_<group>.go`. This predates the branch (file already existed at `c9490b724`; diff only touches lines 107-111 and 209-221) — flagged for completeness per the "grade the file, not the prevalence" rule, but it is pre-existing debt this branch did not introduce and did not worsen. |
| FILE-02 | RestModel + Transform in `rest.go` | `effect/model.go` | N/A | No `RestModel`/`Transform`/`GetName` symbols added; `effect.RestModel` and its `Extract` already live in `effect/rest.go` (unchanged by this diff). |
| FILE-04 | Entity + Migration in `entity.go` | (none of the six files) | N/A | None of the six files touch GORM entities. |
| FILE-06 | No package-named catch-all file | `shadow_stars.go` | PASS | File holds exactly one cohesive feature's pure decision functions + one emit orchestrator (no RestModel, no entity, no requests.go-shaped REST client code bundled in) — not a `wallet.go`-style collapse of unrelated responsibility types. |

### DOM-21 — atlas-constants reuse (explicit focus item)

| Symbol used | File:line | Verified against `libs/atlas-constants` | Status |
|---|---|---|---|
| `item.IsThrowingStar` / `item.ClassificationConsumableThrowingStar` (=207) | `shadow_stars.go:35`, `character_attack_projectile.go:230` | `libs/atlas-constants/item/constants.go:35,175-176` | PASS |
| `skill.NightLordShadowStarsId` (=4121006) | `skill_usage_info.go:32`, `common.go:80`, `reader.go:297` | `libs/atlas-constants/skill/constants.go:1102,2613,3163` | PASS |
| `character.TemporaryStatTypeShadowClaw` (="SHADOW_CLAW") | `common.go:89,97`, `character_attack_projectile.go:217`, `reader.go:304` | `libs/atlas-constants/character/temporary_stat.go:46` | PASS |
| `inventory.TypeValueUse` | `shadow_stars.go:145,152` | `libs/atlas-constants/inventory/constants.go:13` | PASS |
| `StarDraw{Slot int16; ItemId uint32; Quantity int16}` (new local type, `shadow_stars.go:26-30`) | — | No equivalent shared type exists in `libs/atlas-constants` (this is a service-local consume-plan DTO, structurally identical to the pre-existing `ProjectileSlotDraw` in `character_attack_projectile.go:26-30`, which is itself local, not shared) | PASS — no shared-lib duplication; this is a small local plan struct, not a redeclared constants-package type. |

No redeclared classification/enum/numeric-constant type found anywhere in the diff. DOM-21 is clean across all six files.

### DOM-25 — client-interpreted wire values (explicit focus item)

Assessed both candidate values named in the brief:

1. **The SHADOW_CLAW statup amount carrying the star's item id** (`common.go:90,97`, `shadow_stars.go:90,97`) — **not a DOM-25 violation.** DOM-25/the anti-patterns.md rule targets values the client resolves through its own lookup switch to a *semantic meaning* (dispatcher mode bytes, notice/fail-reason codes) — e.g. byte `5` meaning "banned". The SHADOW_CLAW amount is not such a code: it is the literal throwing-star **item id**, which the client reads directly to look up the star's icon/sprite for the buff display — a foreign-key-shaped data value, not an enumerated wire code requiring tenant-table resolution. This is architecturally identical to the pre-existing MONSTER_RIDING statup carrying a vehicle item id for tamed mounts (`reader.go:226-231`, `mount.go`'s `tamedMountStatups`), which is not treated as a DOM-25 case anywhere else in the codebase. No tenant writer-options table applies to "which item id is this" the way one applies to "which of N enumerated notice reasons is this."
2. **The `1` placeholder in `reader.go:304`** — **not a DOM-25 violation.** It is not a value the client ever observes or interprets: atlas-channel unconditionally overwrites it at cast time via `rewriteShadowClawStatups` before the buff is ever applied/serialized to the client (`common.go:94`, confirmed by `TestResolveShadowStarsCast_NoShadowClawInInput` and `TestRewriteShadowClawStatups_AppendsWhenAbsent` in `shadow_stars_test.go`). It exists solely to survive `produceBuffStatAmount`'s `if value != 0` guard (`reader.go:447`) inside the WZ-ingest pipeline — an internal plumbing sentinel, not a wire value.

### SEC-* — client-supplied star id (explicit focus item)

The star id (`SpiritJavelinItemId`, decoded at `skill_usage_info.go:33`) is attacker-controlled and drives a real inventory consume. Reviewed as a targeted authorization/anti-cheat gate rather than the JWT-oriented SEC-01..04 checklist (atlas-channel is not a token-issuing/auth service):

| Risk | Gate | Evidence | Status |
|---|---|---|---|
| Client sends a non-throwing-star item id (e.g. a weapon or potion id) to smuggle an arbitrary item id into the buff/consume path | `item.IsThrowingStar(item.Id(starItemId))` classification check | `shadow_stars.go:35` (`validateShadowStar`), unit-tested by `TestValidateShadowStar` (`shadow_stars_test.go:26-40`, the `notAStar` case) | PASS |
| Client sends a valid throwing-star id the character does not own | Ownership loop over `assets` requiring `TemplateId() == starItemId && Quantity() > 0` | `shadow_stars.go:38-43`, tested by `TestValidateShadowStar` (the `starSubi`-unowned case) | PASS |
| Cast proceeds (HP/MP/cooldown spent, buff applied) before validation fails | Pre-flight block runs `resolveShadowStarsCast` and returns on `!ok` **before** the `e.HPConsume()`/`e.MPConsume()`/cooldown/buff-apply block | `common.go:73-96` precedes `common.go:98` (`if e.HPConsume() > 0`); confirmed by reading the full `UseSkill` body — the Shadow Stars pre-flight is the first statement in the closure | PASS |
| `characterId` itself is attacker-controlled (could target another character's inventory) | `characterId` is taken from the server-authoritative session, not from packet content | `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go:102` passes `s.CharacterId()` (session), not a packet field | PASS |
| TOCTOU: quantity checked at validation time no longer holds at consume time (e.g., item traded/dropped between validate and reserve) | Consume goes through the same atomic reservation pipeline as the existing projectile system: `once.ReservationValidator(txId, draw.ItemId)` + a slot-scoped one-time Kafka consume handler — a reservation that no longer matches fails cleanly rather than over-consuming | `shadow_stars.go:144-155`, doc comment at `shadow_stars.go:130-131` ("Reservation atomicity means a slot that no longer holds the item fails cleanly without over-consuming") | PASS — same mechanism as the pre-existing, already-shipped projectile consume path (`character_attack_projectile.go:150-175`) |
| Shortfall (owns fewer stars than `bulletCount`) silently drains to 0 without warning | Logged via `l.Warnf(...)` before consuming what's available | `common.go:91-93` | PASS — matches the explicit, PRD-approved shortfall posture (`docs/tasks/task-158-shadow-stars/design.md:199`), not a silent degradation. |

No SEC findings. The abort-before-spend ordering is the single most important property here and it holds.

### Minor / Non-Blocking Observations

1. **FILE-01 (pre-existing, not introduced by this branch):** `character_attack_projectile.go` declares `ProjectileProcessor`/`ProjectileProcessorImpl`/`NewProjectileProcessor` in a feature-named file rather than `processor.go`/`processor_<group>.go`. This predates `c9490b724` and this diff does not add to or worsen the violation (it only touches `Plan()`'s body and adds `projectileConsumptionSkipped`, both already inside the offending file). Not blocking this branch; flagged for whoever next touches package-wide structure in `socket/handler`.
2. **Test style (previously triaged, re-confirmed, not re-raised as new):** `shadow_stars_test.go` and `effect/model_test.go` use direct assertions / struct literals rather than table-driven `t.Run` subtests for every case. DOM-20 formally applies only to "domain packages with `model.go`" per Phase 2's classification, and neither `handler` (skill) nor the nested-VO `effect` package meets that bar (see Scope Classification above) — this is not a FAIL, and matches sibling test files in the same directories (`mount_test.go`, `registry_test.go`).

### Blocking (must fix)

None.

### Non-Blocking (should fix)

- FILE-01: `character_attack_projectile.go`'s `ProjectileProcessor`/`ProjectileProcessorImpl`/`NewProjectileProcessor` live outside a `processor.go`-named file (pre-existing debt, not introduced by this branch).

### Overall Verdict

**PASS.** Zero blocking findings. DOM-21 (constants reuse) is clean. DOM-25 (client wire values) does not apply to either candidate value assessed. The client-supplied star id is gated by classification + ownership + session-authoritative characterId + atomic reservation, with abort-before-spend ordering verified by direct code reading. The one Minor/Non-Blocking finding is pre-existing structural debt this branch did not introduce.
