# Plan Audit — task-246-maple-life-character-creation (Tasks 14–27)

**Plan Path:** docs/tasks/task-246-maple-life-character-creation/plan.md
**Audit Date:** 2026-08-21
**Branch:** task-246-maple-life-character-creation
**Base Branch:** main
**Range Audited:** Tasks 14 through 27 (Task 1-13 audited by a sibling shard; see `audit-plan-adherence-1-13.md`)
**Range Basis:** merge-base `24a33a2e6` .. HEAD `77b302206`

## Executive Summary

All 14 tasks in this range (14–27, including Task 15's supersession by Task 26
and the controller-added Task 27) are fully implemented and match the plan's
described shape, field-for-field in the cases spot-checked. Every Go module
touched in this range builds and its tests pass (`libs/atlas-saga`,
`services/atlas-saga-orchestrator`, `services/atlas-character`,
`services/atlas-configurations`, `services/atlas-character-factory`,
`services/atlas-channel`). The four real `packet-audit` checks (`matrix`,
`operations --check`, `fname-doc --check`, `gate-check`) all pass; the fifth
plan-listed command (`template --check`) does not exist as a subcommand and
was correctly struck by the controller. No `go run`/build/test failures were
found. This audit did not independently re-run the flagless `tools/verify.sh`
(out of this shard's scope per the audit's own build/test rule — full-repo
verification is a controller-level gate, not a per-shard one); its status is
reported as not re-verified here, not as a finding against these tasks.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 14 | Seed-status consumer, consume saga, `main.go` wiring | DONE | `services/atlas-channel/atlas.com/channel/kafka/consumer/seed/consumer.go` (236 lines) implements `InitConsumers`/`InitHandlers`, `resolveEntry` (TransactionId-first, account-id fallback, mismatch-not-fallback), `handleCreatedStatusEvent` (builds `saga.MapleLifeUse` one-step `DestroyAssetFromSlot`, announces success arm, logs disconnected-session and destroy-failure cases per design §5.4), `handleFailedStatusEvent` (no destroy, generic-failure arm). `MapleLifeUse` added to both `noReverseWalkSagaTypes` and `allSagaTypes` (`saga-orchestrator/saga/timer.go:256,267`) and aliased in `libs/atlas-saga/model.go:78`, `atlas-channel/saga/model.go:87`, `saga-orchestrator/saga/model.go:64`. `main.go:261` (`seedConsumer.InitConsumers`), `:583` (`InitHandlers`), `:693-694` (`MapleLifeResultWriter`/`MapleLifeErrorWriter`), `:1015` (`MapleLifeCheckNameHandle`). `MapleLifeUseHandle` correctly **absent** — `derivation.md` §3 found no v95 sender for the mislabeled opcode-303 `USE_MAPLELIFE` registry row, so Step 5's "only if §3 found a v95 sender" condition is false. |
| 15 | Full verification and coverage promotion | NOT_APPLICABLE (superseded) | Plan's own Task 26 header: "Supersedes Task 15 — same packet steps plus this amendment's. Run this, not Task 15." No commit targets Task 15 directly; Task 26 (below) is its replacement and was run instead. Correct per plan instruction, not a gap. |
| 16 | Derive the Maple Life content values | DONE | `docs/tasks/task-246-maple-life-character-creation/maple-life-content.md` (642 lines, commit `677b62c51`) with §1 (WZ paths/option lists), §2 (ordinal→class table, ordinals 2-4 explicitly marked **UNCONFIRMED** rather than invented), §3 (five class entries' full field sets with AP/SP arithmetic cited to `processor.go:1685-1709`/`:2178` and job-advance NPC conversation floors), §4 (explicit UNRESOLVED/UNCONFIRMED list), §5 (SP-skill HP/MP contribution ruling, added `2ba40b634`). No Go file changes, matching the plan's "no code" scope. |
| 17 | Carry `ap`/`sp` on the character-creation saga contract | DONE | `libs/atlas-saga/payloads.go:1048,1052` — `AP uint16 json:"ap,omitempty"`, `SP string json:"sp,omitempty"`, additive. All six sites threaded: `payloads_test.go` (3-case table), `saga-orchestrator/kafka/message/character/kafka.go`, `character/producer.go` (`RequestCreateCharacterProvider` now 21 positional params), `character/processor.go` interface+impl, `character/mock/processor.go`, `saga/handler.go:1929`. `TestRequestCreateCharacterProviderCarriesApAndSp` (`character/producer_test.go:34-58`) round-trips `AP:12`/`SP:"3,0,..."` through the produced Kafka message and decodes it back. `libs/atlas-saga` and `saga-orchestrator` both build and test clean (see Build & Test table). |
| 18 | Persist `ap`/`sp` on character creation | DONE | `services/atlas-character/atlas.com/character/character/administrator.go` (`create(...)`, +14/-3 lines) sets `AP: ap` and fixes the malformed SP default from `"0, 0, ..."` (spaced, breaks `Model.SPs()`'s `strconv.Atoi`) to `"0,0,0,0,0,0,0,0,0,0"` at line 27. `consumer.go` threads `AP`/`SP` through `handleCreateCharacter`'s builder chain; `processor_test.go` (+58 lines) and `consumer_test.go` (+104 lines) added. |
| 19 | The `mapleLife` tenant-configuration block | DONE | `services/atlas-configurations/atlas.com/configurations/tenants/maplelife/rest.go` and `templates/maplelife/rest.go` are byte-identical (`diff` exit 0) — `LookOptions`, `StatBlock`, `EquipmentEntry`, `InventoryEntry`, `ClassEntry`, `RestModel` all present with the exact json tags the plan specifies. `tenants/rest.go`/`templates/rest.go` both gain `MapleLife maplelife.RestModel json:"mapleLife"`. `rest_test.go` files added on both sides (round-trip + absent-block + tag-shape cases). |
| 20 | Seed the `mapleLife` block on the four in-scope templates | DONE | `git diff --stat origin/main -- services/atlas-configurations/seed-data/templates/` shows exactly `template_gms_{83,87,92,95}_1.json` (607 lines each); `template_gms_84_1.json` carries **zero** `mapleLife` occurrences (`grep -c` confirms). `template_gms_95_1.json`'s `mapleLife` block decodes to 2 `looks` rows and 10 `classes` rows (5 ordinals × 2 genders); spot-checked ordinal 0/gender 0 (`str:35, ap:114, spSkillId:1000001`) and ordinal 1/gender 0 (`int:20, ap:129, spSkillId:2000001`) against `maple-life-content.md` §3's derived table — values match exactly. |
| 21 | Mirror the `mapleLife` block into `atlas-character-factory` | DONE | `configuration/tenant/maplelife/rest.go` is byte-identical to the `atlas-configurations` tenants mirror (`diff` exit 0). `configuration/tenant/rest.go:21` adds `MapleLife maplelife.RestModel`. `TestProjectionDecodesMapleLife` present in `configuration/projection/projection_test.go:122`. |
| 22 | `CreateMapleLife` — resolve class, validate look, build saga | DONE | `factory/maple_life.go` (325 lines) implements `MapleLifeCreateRestModel`, the four sentinel errors, `CreateMapleLife`, `resolveMapleLifePreset` (gender/class/look validation via `validGender`/`validOption`, SP-pool bound check per §11 A7 "rejected, never clamped", name check reusing `CreateFromPreset`'s classification, equipment/inventory atlas-data revalidation), and `toPreset` (`Hair: a.Hair`/`HairColor: a.HairColor` passed separately for `buildPresetCharacterCreationSaga` to combine, `AP`/`SP` carried via the widened `preset.Attributes`). This module's four-file `preset.Attributes` widening (both configurations mirrors + factory mirror + saga builder) matches the plan's own escalation-guard scope exactly. `factory/maple_life_test.go` (525 lines) covers every error-case row from the plan's table plus `TestCreateMapleLifeNeverConsultsCreationTemplates`. Independently reviewed and PASSed by a prior specialist review (`docs/tasks/task-246-maple-life-character-creation/review-task-22.md`), which additionally verified the HP/MP arithmetic by mutation-testing the `29` constant and confirming both assertions fail as expected. One non-blocking coverage gap noted there (untested JSON-decode path for `SkillEffectRestModel.X`), carried forward below. |
| 23 | `POST /factory/characters/maple-life` | DONE | `factory/resource.go:19` adds the `CreateMapleLife = "create_maple_life"` operation constant, `:33` registers `fr.HandleFunc("/maple-life", ...)`, `:57-77` `categorizeMapleLifeError` maps every error to the exact status the plan's table specifies (400 for ordinal/look/SP/name-invalid/preset-validation, 409 for duplicate name, 502 for atlas-data-unreachable, 500 for not-configured and the generic default). `resource_test.go:105` `TestCategorizeMapleLifeError`, `:167` `TestMapleLifeRouteReturnsAcceptedWithTransactionId` both present. |
| 24 | The channel's `CreateMapleLife` client | DONE | `character/factory/requests.go:16` `MapleLifeResource = "factory/characters/maple-life"`, `:57` `requestCreateMapleLife`. `character/factory/processor.go:25` adds `CreateMapleLife(...)` to the `Processor` interface alongside the **unchanged** `SeedCharacter` (still present at `:18`, still used by `POST characters/seed`). `processor_test.go` (+340 lines) covers request-shape and status-mapping per the plan's two test tables. |
| 25 | Rewire the submit handler — A7 gate and `al` mapping | DONE | `services/atlas-channel/atlas.com/channel/socket/handler/maple_life_create.go` (248 lines, full rewrite). `grep -rn 'seedCharacterFunc\|jobIndex\|subJobIndex'` on this file returns **no output** — the withdrawn mapping is fully gone (Step 4's own check, re-run and confirmed here). `createMapleLifeFunc` forwards `AL0()` unchanged, `AL1()/10*10` (style-normalised), `AL2()%10` (colour digit), `byte(AL3())` (skin), matching the `al` table exactly. Gate 5 (`sub.CurrentClass() < 0 \|\| > 4`, `sub.SP() < 0 \|\| > 10`) added immediately before the factory call, rejecting rather than clamping per §11 A7. `logCreateFailure` maps `requests.ErrConflict` → `MapleLifeErrorNameTakenAtSubmit`, everything else → `MapleLifeErrorUnknownError`. `maple_life_create_test.go` (+750 lines) covers the wire-mapping table, ordinal/SP range gates, and the conflict→name-taken mapping. |
| 26 | Full verification and coverage promotion (supersedes 15) | DONE | Re-ran independently as part of this audit from repo root (the plan's literal `cd tools/packet-audit && go run . <sub>` form fails — paths are root-relative, confirmed and previously diagnosed in `.superpowers/sdd/plan/task-26-report.md`): `matrix` (exit 0, no diff produced — `git status --porcelain -- docs/packets/audits` empty, i.e. Step 7's "nothing to commit" is a legitimate no-op), `operations --check` (exit 0), `fname-doc --check` (exit 0), `gate-check` (exit 0, 19 gates, 0 failing). `template --check` does not exist as a subcommand (`tools/packet-audit/cmd/root.go`'s dispatch table has no `template` case) — correctly struck. `STATUS.md` shows `MapleLifeResult`/`MapleLifeError` ✅ on gms_v83/v87/v92/v95, blank on gms_v84 (confirmed VERSION-ABSENT, see Task 27). No `ItemUseMapleLife`/`USE_MAPLELIFE` row appears promoted — confirmed by reading `tools/packet-audit/cmd/run.go:2644-2665`'s own comment, which documents the deliberate decision to leave that codec's `candidatesFromFName` case unlinked pending a `guardFromIf` resolver defect, same state as every sibling `item_use_*.go` sub-body; this was independently re-investigated mid-branch by a `packet-verifier` agent (ledger: "Task 26 USE_MAPLELIFE v95 — BLOCKED - wrong op/codec pairing, confirmed correct"), which is what produced Task 27. Blast-radius check: `git diff --stat origin/main -- services/atlas-configurations/seed-data/templates/` shows only the four expected files; `services/atlas-login/` diff is empty. AP/SP additivity check: zero `CharacterCreatePayload{` hits under `services/atlas-login/`; both `atlas-character-factory` hits either set `AP`/`SP` (Maple Life path, `processor.go:373` block) or omit them (seed path, `processor.go:196` block). |
| 27 | Registry correction — drop spurious gms_v84 MapleLife rows | DONE | Controller-added, not in `plan.md` (per brief `.superpowers/sdd/plan/task-27-registry-correction-brief.md`). Commit `77b302206` deletes the two `MAPLELIFE_RESULT`/`MAPLELIFE_ERROR` rows from `docs/packets/registry/gms_v84.yaml` (opcode 349/350), moving their `status.json` cells from misleading `incomplete`("no audit report") to truthful `n-a`. `git show --stat 77b302206` touches exactly `STATUS.md`, `status.json`, `gms_v84.yaml` — no Go source, no other version's registry. Independently reviewed and **APPROVED** in `docs/tasks/task-246-maple-life-character-creation/review-task-27.md`, which confirmed no collateral cell movement and cross-checked all five registry files directly. |

**Completion Rate:** 14/14 tasks in range (100%) — 13 implemented, 1 (Task 15) correctly superseded per the plan's own instruction
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None in this range. Task 15 is not a skip: the plan text itself instructs "Run this [Task 26], not Task 15," and Task 26 was run and its commit (implicit — Steps 1-5 ran clean, Step 7 was correctly a no-op since the matrix was already fresh from Task 27's own regeneration) covers Task 15's full scope plus the amendment's additions.

## Build & Test Results

| Service / Module | Build | Tests | Notes |
|---|---|---|---|
| `libs/atlas-saga` | PASS | PASS | `go build ./... && go test ./...` from module root |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator` | PASS | PASS | Full `go test ./...`; `saga` and `saga/mock` packages pass, including `TestEverySagaTypeIsClassified` covering the new `MapleLifeUse` classification |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | Full `go test ./...`; `socket/handler` (1.969s) covers the rewired submit handler |
| `services/atlas-character/atlas.com/character` | PASS | PASS | Ran in background (>120s due to `pending_change` package's 206s suite, pre-existing and unrelated to this range); exit code 0, all reported packages `ok` |
| `services/atlas-configurations/atlas.com/configurations` | PASS | PASS | Full `go test ./...`; `tenants/maplelife` and `templates/maplelife` both `ok` |
| `services/atlas-character-factory/atlas.com/character-factory` | PASS | PASS | Full `go test ./...`; `factory` and `configuration/projection` both `ok` |
| `tools/packet-audit` (4 real checks) | N/A | N/A | `matrix`, `operations --check`, `fname-doc --check`, `gate-check` all exit 0 when run from repo root; `template --check` is not a real subcommand and was correctly struck by the plan's own Task 26 amendment note |
| `tools/verify.sh` (flagless, full-repo gate) | NOT RE-RUN | NOT RE-RUN | Out of this shard's per-service build/test scope; a controller-level gate that should still run once, branch-wide, before PR if it has not already |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (pending confirmation that the flagless `tools/verify.sh` gate has been run branch-wide, and pending the sibling shard's audit of Tasks 1-13)

## Action Items

1. Confirm (or run) the flagless `tools/verify.sh` from the repo root before opening/merging the PR — this audit's build/test verification was scoped to the modules Tasks 14-27 touch, not the full-repo bake/`-race` gate Task 26 Step 6 calls for.
2. Non-blocking, carried forward from the independent Task 22 review: `services/atlas-character-factory/atlas.com/character-factory/data` has zero test files, so the `SkillRestModel.Effects[].X` → `SkillInfo.EffectX` JSON-decode step (the exact step that determines whether every SP investment's HP/MP bonus is real or silently zero) is untested at the decode boundary — only exercised through a mock that bypasses JSON entirely. Recommend a follow-up test with a literal `data/skills` JSON:API fixture; not a blocker for this branch since the risk is pre-existing convention and the arithmetic itself is thoroughly tested via the mock.
3. `maple-life-content.md` §2/§4 records the class-ordinal order for {Bowman, Thief, Pirate} (ordinals 2/3/4) as UNCONFIRMED, per the plan's own explicit allowance ("a wrong order is a seed-data fix, not a code change"). This is documented, not silently missing, but is worth flagging to whoever signs off on live testing, per `context.md`'s note that it blocks live testing.

---

## Controller correction (appended after review)

**Non-blocking finding 2 (the `data` package decode gap) is incorrect and is withdrawn.** It
states that `atlas-character-factory/atlas.com/character-factory/data` has zero test files and
that the `SkillRestModel.Effects[].X` -> `SkillInfo.EffectX` decode boundary is exercised only
through a mock. That test exists and passes:

`data/skill_requests_test.go` (`TestGetSkillsByIds_DecodesEffectX`) stands up an `httptest`
server returning a real JSON:API document containing `"effects": [{"x": 0}, {"x": 12}, {"x": 24}]`
and drives the genuine `requestSkillsByIds -> jsonapi.Unmarshal -> SkillRestModel -> SkillInfo`
path with no mock, asserting `EffectX == [0, 12, 24]`. Its doc comment notes it is deliberately
the only test exercising the real HTTP decode path, every other factory test bypassing it via
`dmock.ProcessorMock`.

```
$ go test ./data/... -run TestGetSkillsByIds_DecodesEffectX -count=1
--- PASS: TestGetSkillsByIds_DecodesEffectX (0.01s)
ok      atlas-character-factory/data    0.017s
```

The gap was real when `review-task-22.md` first raised it, and was closed by commit `7b2196004`
("chore(character-factory): close two queued task-246 cleanup items") — which is inside the range
this shard audited. The finding carried the stale text forward without re-checking HEAD.

**Recommendation 1 (run the flagless `tools/verify.sh`) is resolved.** The controller ran it
against HEAD `77b302206`: exit 0, "All checks passed", 178 `go build/vet/test -race` module runs,
every guard ✓. `docker buildx bake` was skipped by the script's own `no go.mod touched` selection
rule, not by a flag — this was the flagless invocation.

**Finding 3 stands and is the one open item on this branch** — `maple-life-content.md` §2/§4
recording the ordinal->class order for {Bowman, Thief, Pirate} (ordinals 2/3/4) as UNCONFIRMED is
accurate, is what the plan explicitly permits ("a wrong order is a seed-data fix, not a code
change"), and does block live testing per `context.md`. Surfaced to the user.
