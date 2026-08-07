# Backend Guidelines Audit — task-156-gm-hide-heal-dispel

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-07-17
- **Git Range:** `43894c25bfc368d743a5434c4537c112f8a85b14..d5d2e104d`
- **Build:** PASS
- **Tests:** all packages PASS (`go test -race ./... -count=1`, full module)
- **Overall:** NEEDS-WORK

Default posture for this audit is FAIL until file:line evidence proves otherwise. Nothing below was accepted on the grounds of "the rest of the service does it this way."

## Build & Test Results

```
cd services/atlas-channel/atlas.com/channel
go build ./...        # clean, no output
go vet ./...           # clean, no output
go test -race ./... -count=1   # ok for every package with test files; no failures
tools/goroutine-guard.sh (repo root)  # exit 0
```

No `go.mod`/`go.sum` changes in range → DOM-22 (Dockerfile lib sync) is N/A. No new Kafka topic env vars introduced (CANCEL_BY_TYPES reuses the existing `COMMAND_TOPIC_CHARACTER_BUFF` topic as a new `Type` value in the same envelope) → DOM-23 is N/A.

## Package Classification (Phase 2)

None of the changed/added packages have a `model.go` introduced or modified in this range, so the full DOM-01..DOM-19 domain checklist (builder/entity/rest/provider) is largely **N/A by scope** — this diff is skill-handler / Kafka-orchestration code layered on top of pre-existing domain packages (`character/buff`, `character`, `session`, `map`) that were not themselves restructured. Every touched package was still run through the File Responsibilities Checklist (mandatory for support packages) and the applicable DOM items (constants reuse, logger typing, goroutine safety, testing).

| Package | Classification |
|---|---|
| `character/buff` (hidden.go, processor.go, producer.go, +tests) | Existing domain package (model.go untouched this range) — additive predicate + processor method |
| `data/skill/effect` (model.go, model_test.go) | Existing domain package — net-zero diff vs base (accessors added then removed within range) |
| `skill/handler/healdispel` | Support / orchestrator package (skill-cast handler, no model.go, no resource.go — Kafka/socket pipeline, not REST) |
| `skill/handler/hide` | Support / orchestrator package, same shape as healdispel |
| `skill/handler` (recipients.go) | Support / shared library package (selector helpers used by multiple skill handlers) |
| `kafka/message/buff` (kafka.go) | Message-definition package (Command/StatusEvent structs + topic/type consts) |
| `kafka/consumer/map` (consumer.go) | Kafka consumer / broadcast package (pre-existing, ~800 lines; this diff adds ~100 lines: GM-hide gate + reveal path) |
| `skill/handler/registrations` | Trivial init-wiring package (blank imports only) |

## File Responsibilities Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in `processor.go` | PASS | `character/buff/processor.go:16-22` (interface incl. new `CancelByTypes`), `:66-69` (impl). No Processor code leaked into `hidden.go` or `producer.go`. |
| FILE-02 | RestModel/Transform in `rest.go` | PASS (N/A new code) | No new RestModel/Transform introduced in this diff. |
| FILE-03 | Cross-service request funcs in `requests.go` | PASS (N/A) | No new `requests.RootUrl`/`GetRequest`/`PostRequest` call sites added in this diff. |
| FILE-04 | Entity/Migration/TableName in `entity.go` | PASS (N/A) | No entity changes in this diff. |
| FILE-05 | Builder/Model/administrator/provider placement | PASS (N/A) | No builder/administrator/provider files touched. |
| FILE-06 | No package-named catch-all file | PASS | `character/buff/hidden.go` holds exactly one exported symbol (`IsGmHidden`) — single-purpose utility file, not a collapse of Processor+RestModel+requests. `kafka/message/buff/kafka.go:57-71` (new `CancelByTypesCommandBody`) is additive to an existing message-definition file, consistent with its established single responsibility (message schema only, no Processor/RestModel code). |

## Domain / Support Checklist Results

### character/buff

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | `character/buff/processor.go:30` `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor` |
| DOM-21 | No duplication of atlas-constants types | PASS | `character/buff/hidden.go:4,15` uses `skill2.SuperGmHideId` from `libs/atlas-constants/skill` (verified at `libs/atlas-constants/skill/constants.go:3253`); `hidden_test.go:8,15,22` uses `skill2.RogueDarkSightId` (`constants.go:3142`) — no service-local re-declaration of these ids. `healdispel.go:33-42`'s `diseaseTypes` list uses `charconst.TemporaryStatType*` string constants from `libs/atlas-constants/character/temporary_stat.go` (all 11 verified present: lines 16/23-26/36-38/45/57/81/122) rather than inventing local string literals. |
| DOM-24 | Kafka producer stubbed in emit-path tests | PASS (N/A) | `character/buff/producer_test.go` (`TestCancelByTypesCommandProvider`) calls only the pure `CancelByTypesCommandProvider` message-builder, never `ProcessorImpl.CancelByTypes` (the method that actually calls `producer.ProviderImpl`, `processor.go:66-69`). No test in the diff invokes a real emit path, directly or transitively (`healDispelDeps.dispel` / `hideDeps.applyHide|cancelHide` are 100% test-injected fakes — see `healdispel_test.go:39-62`, `hide_test.go:33-43`). No `TestMain`/`producertest.InstallNoop()` needed and none is required by the emit-surface actually exercised. |
| — | Thread-safety of message construction | PASS | `producer.go:67` defensively copies the caller-owned slice (`append([]string(nil), types...)`) rather than aliasing `healdispel.diseaseTypes`, which is a shared package-level `var` reused across every cast — avoids a future mutation-through-alias hazard. |

### data/skill/effect

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Dead-code cleanup after accessor removal | PASS | `git show d5d2e104d` removes `MP()`/`HpR()`/`MpR()` from `model.go` and their test from `model_test.go` in the same commit that stops referencing them; `grep` confirms zero remaining references anywhere in the service. Net diff vs. the audit's base commit is empty (accessors were added earlier in this same range by `4b411a4f3` and removed again by `d5d2e104d`) — no dead code survives to HEAD. `HP()` (a distinct, still-used accessor) correctly remains (`model.go:111`). |

### skill/handler/healdispel

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Per-recipient failure isolation (log-and-continue) | PASS | `healdispel.go:120-132`: `ChangeHP`/`ChangeMP`/`dispel` errors are each logged via `l.WithError(err).Errorf(...)` and the loop continues to the next recipient / next call; no `return err` anywhere inside the `for _, r := range recipients` body. Verified by `healdispel_test.go:118-150` (`TestPerRecipientIsolation`): a forced `ChangeHP` failure for recipient 1 does not prevent recipient 2's HP/MP/dispel from being applied, nor does it suppress the self-announce. |
| — | SuperGM job gate fails closed | PASS | `healdispel.go:100-103` and `hide.go:60-63`: `job.IsA(c.JobId(), job.SuperGmId)` is checked before any state mutation; on `loadCaster` error the function returns before the job check even runs (`healdispel.go:95-99`, `hide.go:55-59`) — no default-allow path. |
| DOM-21 | Job constant sourced from atlas-constants | PASS | `job.SuperGmId` resolves to `libs/atlas-constants/job/constants.go:1190` (`Id(910)`); `job.IsA` is the shared helper at `libs/atlas-constants/job/model.go:31`, not a local reimplementation. |
| DOM-20 | Table-driven tests | **FAIL** | `healdispel_test.go` defines 5 separate `TestXxx` functions (`TestNonSuperGmRejected`, `TestHealDispelAllRecipients`, `TestForeignSuppressedWhenHidden`, `TestPerRecipientIsolation`, `TestEffectiveMaxFallsBackToBase`), none using a `tests := []struct{...}{}` + `t.Run` table. Same pattern in `hide_test.go` (3 functions), `hidden_test.go` (1 function, 4 inline assertions), `producer_test.go` (1 function), `recipients_map_test.go` (1 function). Functionally sound and each case is independently readable, but mechanically this is not the `t.Run` table-driven shape the checklist's Pass Criteria specifies. Severity: Minor (non-blocking) — coverage is present and correct, only the structural convention deviates. |

### skill/handler/hide

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | No foreign skill-use leak on toggle | PASS | `hide.go:38-39,89-92`: comment explicitly documents "There is deliberately NO foreign-announce seam" and `applyHide` only calls `d.announceSelf`, never a foreign announce — confirmed no foreign-broadcast call exists anywhere in `hide.go`. |
| DOM-21 | DARK_SIGHT stat sourced from atlas-constants | PASS | `hide.go:17,125` `charconst.TemporaryStatTypeDarkSight` → `libs/atlas-constants/character/temporary_stat.go:16` (`"DARK_SIGHT"`). |
| — | Layering: skill handler reaching into a Kafka-consumer package | **Minor / architectural note** | `hide.go:11,131-136` imports `_mapconsumer "atlas-channel/kafka/consumer/map"` and calls its exported `DespawnCharacterInMap`/`SpawnCharacterInMap`. No circular import exists (`kafka/consumer/map/consumer.go` does not import `skill/handler/hide` — confirmed by successful build and by grep), so this is not a hard layering violation under the anti-patterns doc (which only prohibits `resource.go → provider.go` / `resource.go → entity.go`). It is nonetheless an unusual dependency direction — a skill-cast handler depending on a Kafka *consumer* package's broadcast helpers rather than on a neutral `map` package export — worth flagging as a design smell, not a blocking finding since no specific checklist item forbids it. |

### kafka/consumer/map (consumer.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Single wire-emit choke point claim | PASS (independently verified) | `grep -rn "CharacterSpawnBody("` across the whole service returns exactly one production call site: `consumer.go:439` inside `emitCharacterSpawn`. Both `spawnCharacterForSession` (`:446-473`) and `spawnCharacterForSessionRevealed` (`:497-512`) route through it — the "single choke point" claim in the doc comment (`:429-433`) is factually accurate, not just asserted. |
| DOM-26 | Goroutines via `routine.Go` | PASS | All spawn-fan-out goroutines in `SpawnForSelf` use `routine.Go(l, ctx, func(_ context.Context) {...})` (e.g. `:182,202,218,224,230...`); zero bare `go` statements in the changed file (`grep -nE '^\s*go (func|[A-Za-z_])'` = empty); `tools/goroutine-guard.sh` exits 0 for the whole repo. |
| — | Concurrency of the new GM-hide-gated buff read | PASS, no data race, but see finding below | `ForOtherSessionsInMap` (`map/processor.go:102-105`) → `session.ProcessorImpl.ForEachByCharacterId` (`session/processor.go:227-231`) runs its operator via `model.ParallelExecute()`. `spawnCharacterForSessionRevealed` (new, `consumer.go:497-512`) and the pre-existing `spawnCharacterForSession` both call `buff.NewProcessor(l, ctx).GetByCharacterId(c.Id())` per-session inside that parallel fan-out. Each call constructs a fresh `ProcessorImpl` and only reads a shared, read-only `ctx`/`logrus.FieldLogger` — no shared mutable state, so this is race-detector-clean (confirmed: `go test -race` is clean). It is, however, N concurrent identical REST round-trips to atlas-buffs for the *same* character's buff list (one per receiving session) on every reveal/enter broadcast — this redundancy is inherited from the pre-existing `spawnCharacterForSession` (present in the file before this diff) and this PR's new `spawnCharacterForSessionRevealed` copies the identical pattern rather than threading a single fetched `bs` through the fan-out. Not a new bug, but a missed opportunity to fix it while touching this exact code path. Non-blocking. |
| — | **Untested new suppression/reveal logic** | **FAIL** | The entire GM-hide suppression gate (`consumer.go:456-467`, the `if buff.IsGmHidden(bs) { return nil }` check inside `spawnCharacterForSession`), the new `emitCharacterSpawn` extraction (`:434-444`), `spawnCharacterForSessionRevealed` (`:497-512`), `DespawnCharacterInMap` (`:517-521`), and `SpawnCharacterInMap` (`:533-543`) have **zero** direct unit tests. `kafka/consumer/map/consumer_test.go` (the package's existing test file) only gained no new `Test*` functions in this range — `grep -n "^func Test"` against it lists only pre-existing tests (`TestFetchOtherCharactersInMap_SkipsNotFound`, `TestFetchOtherCharactersInMap_InfraErrorIsHardFailure`, `TestSpawnDoorsForSession_*`), none targeting the new gate. `hidden_test.go` covers only the pure `IsGmHidden` predicate in isolation — it does not verify that `spawnCharacterForSession` actually calls `IsGmHidden` and suppresses the spawn, nor that `spawnCharacterForSessionRevealed` correctly omits the check. This is the actual choke point the feature depends on for correctness and it ships with no direct regression coverage; a future refactor of `consumer.go` could silently drop or invert the gate with no test failing. Severity: Important. |

### skill/handler (recipients.go)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | New selector matches documented contract | PASS | `recipients.go:203-223` `SelectAllCharactersInMap`: doc comment claims "no bitmap, no LT/RB rectangle, no HP>0 filter"; body has none of those filters (unlike `selectPartyMembers`, `:230-303`, which does). `recipients_map_test.go:36-40` explicitly asserts an HP-0 recipient (id 2) is *kept*, matching the contract. |
| DOM-20 | Table-driven test | Minor / FAIL (see healdispel note above) | `recipients_map_test.go` is a single `TestSelectAllCharactersInMap` function, not `t.Run`-table-driven. |

## Design / Correctness Finding — fail-open default on hidden-state resolution error

**Severity: Important.**

`healdispel.go:105-109`:
```go
hidden, hErr := d.isGmHidden(characterId)
if hErr != nil {
    l.WithError(hErr).Debugf("Heal+Dispel: unable to resolve hidden state for caster [%d]; treating as visible.", characterId)
    hidden = false
}
```
and `:138-142`:
```go
if !hidden {
    if err := d.announceForeign(c.Level()); err != nil { ... }
}
```

If the buff-list fetch to atlas-buffs (`bp.GetByCharacterId`, wired at `healdispel.go:168-174`) errors transiently, `hidden` defaults to `false` and `announceForeign` fires — broadcasting the Heal+Dispel skill-use animation, and therefore the caster's map position, to every other player in the field. This directly contradicts the feature's own stated anti-leak guarantee: `design.md:70` (OQ-3) states "A foreign animation for an invisible caster leaks position," and `hide.go:37-39` independently documents the same principle hard enough to omit a foreign-announce seam entirely ("it would leak GM presence in both toggle directions"). A transient atlas-buffs error is exactly the scenario in which a genuinely-hidden GM would be revealed by this code path. The identical fail-open default appears in `hide.go:65-69`, but there the consequence is a harmless idempotent re-hide/re-despawn, not a leak — so only the `healdispel.go` instance is a real defect.

This was implemented exactly as written in `plan.md:822-824`, so it is plan-adherent, but the plan itself bakes in a default that works against the design's own OQ-3 resolution. Recommend defaulting `hidden = true` (fail toward suppressing the foreign announce) on `isGmHidden` error in `healdispel.go`, matching the safer failure direction hide.go's design comment argues for.

## Documentation Gaps (patterns-ingress-documentation.md)

**Severity: Important** — the guideline requires "Adding new Kafka commands or events" to be documented, and this service's own convention (see the "Mystic Door skill handler" section precedent) documents every new skill handler in `docs/domain.md`.

1. `services/atlas-channel/docs/kafka.md:394-398` — the `COMMAND_TOPIC_CHARACTER_BUFF` entry lists `Message Type: Command[ApplyBody], Command[CancelBody]` only. The new `Command[CancelByTypesCommandBody]` / `CANCEL_BY_TYPES` type (`kafka/message/buff/kafka.go:16,46`) is not listed. `grep -n "CANCEL_BY_TYPES" services/atlas-channel/docs/kafka.md` returns nothing.
2. `services/atlas-channel/docs/domain.md` has no section for the new SuperGM Hide or Heal+Dispel skill handlers (`grep -in "hide\|healdispel" docs/domain.md` returns nothing), despite the file documenting the analogous pre-existing "Mystic Door skill handler" (`domain.md:308-313`) and Resurrection handlers in the same style. Neither `docs/kafka.md` nor `docs/domain.md` was touched anywhere in this range (`git diff --stat` for both paths is empty).

## Summary

### Blocking (must fix)

- **Untested GM-hide suppression/reveal logic** in `kafka/consumer/map/consumer.go` — the actual choke point (`IsGmHidden` gate inside `spawnCharacterForSession`, plus the new `spawnCharacterForSessionRevealed`/`DespawnCharacterInMap`/`SpawnCharacterInMap`) has no direct test coverage; only the pure `IsGmHidden` predicate is tested in isolation.
- **Fail-open default leaks hidden GM position** — `healdispel.go:105-109,138-142`: on an `isGmHidden` resolution error, `hidden` defaults to `false`, causing `announceForeign` to broadcast the caster's position even when the caster may actually be hidden. Contradicts `design.md:70` (OQ-3) and the anti-leak rationale documented in `hide.go:37-39`.
- **Documentation not updated** for a new Kafka command type and two new skill handlers — `docs/kafka.md` (missing `CancelByTypesCommandBody` under `COMMAND_TOPIC_CHARACTER_BUFF`) and `docs/domain.md` (no Hide / Heal+Dispel sections), per `patterns-ingress-documentation.md`'s explicit requirement to document all consumed/produced Kafka commands and this service's own precedent of a domain.md section per skill handler.

### Non-Blocking (should fix)

- DOM-20: none of the 5 new/modified test files (`healdispel_test.go`, `hide_test.go`, `hidden_test.go`, `producer_test.go`, `recipients_map_test.go`) use the `tests := []struct{...}{}` + `t.Run` table-driven shape the checklist specifies; coverage itself is adequate.
- `spawnCharacterForSessionRevealed` (new) repeats the pre-existing `spawnCharacterForSession` pattern of one `buff.GetByCharacterId` REST call per receiving session under `ParallelExecute` fan-out, rather than fetching once and threading `bs` through. Not a new bug (the pattern predates this diff) but a missed opportunity to fix it while touching the exact function.
- `skill/handler/hide` importing `kafka/consumer/map` (a Kafka-consumer package) directly for its broadcast helpers is an unusual dependency direction; no circular import exists and no specific guideline forbids it, but it is worth reconsidering — those helpers arguably belong in a neutral `map` package export.
