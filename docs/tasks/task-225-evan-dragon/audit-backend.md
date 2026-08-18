# Backend Audit — task-225-evan-dragon

- **Scope:** `services/atlas-dragons/atlas.com/dragons/`, `libs/atlas-packet/dragon/`, and the dragon-related additions/modifications in `services/atlas-channel/atlas.com/channel/`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-13
- **Build:** PASS (per controller; not re-run by this audit — see task instructions)
- **Tests:** PASS (per controller; not re-run by this audit)
- **Overall:** NEEDS-WORK

## Domain / Package Classification

atlas-dragons is deliberately database-free (Redis-backed), so no package has `entity.go`/`administrator.go`/`provider.go` in the DOM-* sense. `dragon/registry.go` is the Redis analogue (mirrors `atlas-summons/summon/registry.go`). Packages:

| Package | Classification |
|---|---|
| `atlas-dragons/dragon` | Domain package (Redis-backed; `model.go`+`builder.go`+`registry.go` stand in for entity/provider/administrator) |
| `atlas-dragons/character` | Support / External-HTTP-client package (REST client into atlas-character) |
| `atlas-dragons/world` | Support package (cross-field REST listing) |
| `atlas-dragons/rest` | Support package (HTTP handler-dependency wrappers) |
| `atlas-dragons/kafka/consumer/dragon` | Sub-domain (command consumer) |
| `atlas-dragons/kafka/consumer/character` | Sub-domain (event consumer / lifecycle cascade) |
| `atlas-channel/dragon` | Support / External-HTTP-client + Kafka-producer package |
| `atlas-channel/kafka/message/dragon` | Support (mirrored wire contract) |
| `atlas-channel/kafka/consumer/dragon` | Sub-domain (event consumer → packet broadcast) |
| `atlas-channel/socket/handler/dragon_move.go`, `socket/writer/dragon.go` | Support (packet glue) |

---

## File Responsibilities Checklist

| ID | Package/File | Status | Evidence |
|----|------|--------|----------|
| FILE-05 (Builder placement) | `atlas-dragons/dragon` | PASS | `dragon/builder.go:1-49` — `Builder`/`NewBuilder`/`Clone` all in `builder.go`; `dragon/model.go:18-32` holds only `Model` + accessors. |
| FILE-06 (no catch-all file) | `atlas-dragons/dragon` | PASS | No `dragon.go`; responsibilities split across `model.go`, `builder.go`, `processor.go`, `registry.go`, `resource.go`, `rest.go`, `producer.go`, `kafka.go`. |
| FILE-02 (RestModel/Transform in rest.go) | `atlas-dragons/dragon` | PASS | `dragon/rest.go:15-55` — `RestModel`, `GetID/SetID/GetName`, `Transform` all present in `rest.go`. |
| FILE-05 (Builder placement) | `atlas-dragons/character` | **FAIL** (Important) | `character/model.go:1-43` — `Model` **and** `Builder`/`NewBuilder`/`Build()` are both defined in `model.go`; there is no `builder.go` in the package. Per file-responsibilities.md, Builder belongs in its own `builder.go`. Small blast radius (43-line file) but it is the same collapsed-file pattern the file-responsibilities table exists to prevent. |
| FILE-02 (RestModel/Extract in rest.go) | `atlas-dragons/character` | PASS | `character/rest.go:9-32` — `RestModel` + `Extract` correctly isolated from `model.go`. |
| FILE-03 (request funcs in requests.go) | `atlas-dragons/character` | PASS | `character/requests.go:17-23`. |
| **FILE-02 (RestModel/Extract in rest.go)** | `atlas-channel/dragon` | **FAIL** (Important) | `dragon/model.go:1-57` in `services/atlas-channel/atlas.com/channel/dragon/` contains `Model` (lines 13-25), `RestModel` (27-38), `GetName/GetID/SetID` (40-47), **and** `Extract` (49-57) all in one file. There is **no `rest.go`** anywhere in this package (`find dragon -type f` → only `model.go`, `processor.go`, `producer.go`, `requests.go`). This is exactly the collapsed-file anti-pattern the reviewer brief called out by name (the `wallet.go` precedent) — prevalence of similar collapses elsewhere does not exempt this instance; grading against the table, not the sibling packages. |
| FILE-01 (Processor in processor.go) | `atlas-channel/dragon` | PASS | `dragon/processor.go:15-57` — interface + impl + all methods. |
| FILE-03 (request funcs in requests.go) | `atlas-channel/dragon` | PASS | `dragon/requests.go:12-22`. |

---

## External HTTP Client Checklist

Two client packages trigger this: `atlas-dragons/character` (calls atlas-character) and `atlas-channel/dragon` (calls atlas-dragons).

| ID | Check | Package | Status | Evidence |
|----|-------|---------|--------|----------|
| EXT-01 | Relationship interfaces | `atlas-dragons/character` | **FAIL** | `character/rest.go:9-28` — `RestModel` implements `GetName/GetID/SetID` only. No `SetToOneReferenceID`/`SetToManyReferenceIDs`, even as no-ops. Currently low-risk in practice because atlas-character's own `RestModel` (`services/atlas-character/atlas.com/character/character/rest.go`) also has no `GetReferences`, so no `relationships` block is emitted today — but the guideline is explicit that the methods must be present regardless, precisely because task-037 showed this failure mode surfaces later, silently, when the upstream model gains a relationship. |
| EXT-01 | Relationship interfaces | `atlas-channel/dragon` | **FAIL** | `dragon/model.go:27-47` — same gap; `RestModel` has no `SetToOneReferenceID`/`SetToManyReferenceIDs`. |
| EXT-02 | httptest integration test | `atlas-dragons/character` | **FAIL** | No test file exists in the package at all: `find character -name "*_test.go"` returns nothing. `character.Model`/`Extract`/`requestById` are exercised only indirectly through a hand-written `stubCharacters` fake in `dragon/processor_test.go:21-33`, which bypasses `Extract`/JSON unmarshal entirely — exactly the "FakeClient mocks alone do NOT satisfy this" case the checklist calls out. |
| EXT-02 | httptest integration test | `atlas-channel/dragon` | **FAIL** | No test file exists in `services/atlas-channel/atlas.com/channel/dragon/` (`find dragon -type f` → no `*_test.go`). `InMapModelProvider`/`Extract` are unexercised by any httptest fixture. |
| EXT-03 | 404 vs other errors distinguished | `atlas-dragons/character` | PASS | `character/processor.go:28-32` doc comment + `dragon/processor.go:76` `errors.Is(err, requests.ErrNotFound)` — the one call site that consumes this client correctly distinguishes not-found from fetch failure. |
| EXT-04 | RootUrl, no hardcoded URL | both | PASS | `character/requests.go:18` `requests.RootUrl("CHARACTERS")`; `atlas-channel/dragon/requests.go:12-13` `requests.RootUrl("DRAGONS")`. |

---

## Domain Checklist — `atlas-dragons/dragon`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go / NewBuilder / Build | PASS | `dragon/builder.go:19-49`. |
| DOM-06 | Processor takes `logrus.FieldLogger` | PASS | `dragon/processor.go:45`, `character/processor.go:22`. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `dragon/resource.go:32`, `world/resource.go:47`. |
| DOM-09 | Transform errors checked | PASS | `dragon/resource.go:38-42`, `world/resource.go:60-64` both check `err`. |
| DOM-12 | No `os.Getenv` in handlers | PASS | Zero matches in `dragon/resource.go`, `world/resource.go`. |
| DOM-14/DOM-15 | Handlers call processor only, no direct writes | PASS | `dragon/resource.go:32-33`, `world/resource.go:47-48` — both call `NewProcessor(...)`, no registry/db calls. |
| DOM-17 | Domain error → HTTP status mapping | **FAIL** (Important) | `dragon/resource.go:33-36`: `m, err := p.GetByCharacterId(characterId); if err != nil { w.WriteHeader(http.StatusNotFound); return }` maps **every** error to 404, not just `atlasredis.ErrNotFound`. `libs/atlas-redis/registry.go:39-51` shows `Registry.Get` returns a *different*, wrapped error (`fmt.Errorf("redis get: %w", err)`) for genuine failures (timeouts, connection errors) versus `ErrNotFound` for a missing key — the two are distinguishable and the handler does not distinguish them. A Redis outage during this call is indistinguishable, to every caller, from "this character has no dragon" (the documented *normal* case per the doc comment at `dragon/rest.go:24-28`), which actively hides an infrastructure failure as a benign business state. |
| DOM-25 | Client wire values config-resolved | N/A (documented exemption) | jobId/coords are not dispatcher mode bytes; version gate at `SPAWN_DRAGON` and the discarded uint16 are pre-cleared by the task brief. |
| DOM-26 | Goroutines via `routine.Go` | PASS | No bare `go` statements found in `dragon/`, `character/`, `world/`, `rest/`, `kafka/`. |
| DOM-27 | Transient DB errors → 503 | N/A | Service has no database (`atlas.Connect` is Redis via `atlas-redis`, not `database.Connect`). |

### Swallowed-error findings (explicitly requested judgment)

| Location | Verdict | Reasoning |
|---|---|---|
| `dragon/processor.go:120-124` (`Destroy`) | **FAIL** (Important) | `m, err := GetRegistry().Get(...); if err != nil { return nil }` treats **every** `Get` error — not just `atlasredis.ErrNotFound` — as "no dragon, nothing to do." `Registry.Get` (`libs/atlas-redis/registry.go:39-51`) returns a distinct wrapped error for real Redis failures. Failure scenario: a transient Redis timeout during `Destroy` (called on every logout/map-change/channel-change/job-change) makes the function silently return `nil` (success) **without ever calling `Remove`**. The stale registry entry and its `fieldIdx` membership are never cleaned up; the dragon persists forever (no TTL is set anywhere — `Registry.Put` calls `r.reg.Put` → `Set(ctx, rk, data, 0)`, TTL 0 = no expiry) and keeps appearing in `GetInField` results for that map/instance. The identical pattern exists in the claimed precedent, `atlas-summons/summon/processor.go:416-420` (`Despawn`) — that does not cure the defect, it means atlas-summons has the same latent bug. Recommend: only swallow when `errors.Is(err, atlasredis.ErrNotFound)`; propagate everything else. |
| `dragon/registry.go:128-133` (`Put`) and `:207-212` (`Remove`) — `_ = r.fieldIdx.Remove(...)` | Minor | Self-healing in practice: `GetInField` (`registry.go:160-178`) explicitly skips stale members it can't resolve (comment "stale index entry"), so a dropped `fieldIdx.Remove` failure degrades to "briefly returns a ghost id that resolves to nothing," not silent data corruption. Still zero-logged — a persistently failing `fieldIdx.Remove` (e.g., Redis ACL denies `SREM` on the `dragon-map` key) would never surface anywhere. Recommend at minimum a `Warnf` on the discarded error. Matches `atlas-summons/summon/registry.go:266-267` verbatim — same non-fatal gap there too. |
| `socket/handler/dragon_move.go:29` (`_ = dragoncmd.NewProcessor(l, ctx).Move(...)`) | Minor | Unlike `Create`/`Destroy` command paths in the same feature (which all log `.WithError(err).Errorf(...)` on the producer side, e.g. `kafka/consumer/dragon/consumer.go:49-51`), this call drops the Kafka-emit error with **zero** log line, not even Debug. Failure scenario: a Kafka-producer outage silently drops every MOVE command with no diagnostic signal anywhere in the channel logs — the dragon simply appears frozen to everyone else with no error to grep for. Matches `socket/handler/summon_move.go:29` verbatim, so this is an existing service-wide idiom for movement handlers, not a new defect — but "the sibling does it too" is not a guideline exemption per the audit brief, and the complete absence of a log line (vs. a rate-limited Warn) is a real diagnosability gap for a production feature. |

---

## Testing Gaps

| Area | Status | Evidence |
|---|---|---|
| `atlas-dragons/kafka/consumer/dragon` | **FAIL** (Important) | Zero test files in the package (`find kafka -name "*_test.go"` under atlas-dragons returns nothing at all). `handleCreateCommand`/`handleDestroyCommand`/`handleMoveCommand` (`kafka/consumer/dragon/consumer.go:44-70`) — including their `c.Type != CommandTypeX` guards and `field.NewBuilder(...)` wiring — are entirely untested. The underlying `Processor` methods are unit-tested (`dragon/processor_test.go`), but the consumer-level dispatch/guard logic is not. |
| `atlas-dragons/kafka/consumer/character` | **FAIL** (Important) | Same: zero tests for `handleLogin`/`handleLogout`/`handleMapChanged`/`handleChannelChanged`/`handleJobChanged` (`kafka/consumer/character/consumer.go:64-130`). This package also mirrors a cross-module contract (atlas-maps' `EVENT_TOPIC_CHARACTER_STATUS`, `services/atlas-maps/atlas.com/maps/kafka/message/character/kafka.go`) with **no pinned test** protecting against field/tag drift, unlike the dragon-status/dragon-command mirror which is guarded by `services/atlas-channel/atlas.com/channel/kafka/message/dragon/kafka_test.go`. Verified by hand that the field sets currently agree (`atlas-dragons/kafka/consumer/character/kafka.go:44-66` vs. `atlas-maps/kafka/message/character/kafka.go:51-68`), but nothing machine-checks that agreement going forward. |
| `atlas-channel/kafka/consumer/dragon` — handler/helper disconnect | **FAIL** (Important, per explicit reviewer instruction to judge this) | `consumer_test.go` exercises `excludesOwner` and `handles` (`consumer.go:30-43`) directly, but `handleStatusEventCreated`/`handleStatusEventMoved`/`handleStatusEventDestroyed` (`consumer.go:85-143`) **never call either helper** — each hardcodes its own `ForSessionsInMap` or `ForOtherSessionsInMap` call inline (lines 95, 114, 136). The two helpers are therefore dead production code exercised only by tests that assert nothing about the actual broadcast wiring. Concretely: swap `ForOtherSessionsInMap` for `ForSessionsInMap` in `handleStatusEventMoved` (line 114) — the owner's client would start double-applying its own movement — and `go test ./...` stays green, because `TestRecipientPolicyPerEventType` only calls the orphaned `excludesOwner` function, not the handler. The task brief states the handlers were manually verified correct; that verification has no regression net. Fix: either have the handlers call `excludesOwner`/`handles` (make the helpers load-bearing), or replace the unit test with one that drives `handleStatusEventCreated`/`Moved`/`Destroyed` end-to-end against a fake `_map.Processor`/`writer.Producer` and asserts the actual recipient set. |

---

## Cross-Module Contract Duplication Assessment

Two independent wire contracts are hand-mirrored across the atlas-dragons / atlas-channel module boundary:

1. **Dragon status/command contract** (`atlas-dragons/dragon/kafka.go` + `atlas-dragons/kafka/consumer/dragon/kafka.go` ↔ `atlas-channel/kafka/message/dragon/kafka.go`). Guarded by `atlas-channel/kafka/message/dragon/kafka_test.go`, which pins **literal JSON** for the envelope and all three event bodies plus the `MOVE` command body (`kafka_test.go:15-94`).
   - **Assessment: partially adequate, not equivalent to the trade contract's guard.** The literal-JSON tests would catch a renamed/removed json tag (unmarshal into a field that no longer exists silently zero-values, but the test's own assertions on that field would then fail) and would catch an added *required* field with no default. They would **not** catch: (a) a type-width change on one side only (e.g. `X int32` → `int64` on the producer, still tagged `x`) unless the literal fixture's value happens to overflow the narrower type; (b) a field added on one side that the other silently ignores (extra JSON keys don't fail `json.Unmarshal`). Unlike the trade contract, which has `tools/trade-contract-mirror-guard.sh` diffing the two struct definitions **structurally, byte-for-byte, from the `package` clause down**, there is no equivalent mechanical guard for the dragon contract — only prose comments (`dragon/kafka.go:19-23`, `kafka/message/dragon/kafka.go:11-22`) asking the next author to keep the tags aligned by hand.
2. **Character-status contract** (`atlas-maps/kafka/message/character/kafka.go` → `atlas-dragons/kafka/consumer/character/kafka.go`). **No guard at all** — not even a literal-JSON test. See Testing Gaps above.

Recommendation (non-blocking for this audit, but worth a follow-up): either add a `trade-contract-mirror-guard.sh`-style structural diff for the dragon contract pair, or at minimum add the missing literal-JSON pin for the character-status consumer side.

---

## Not Flagged (context supplied by the task brief, verified consistent with code)

- `SPAWN_DRAGON` version-gated `jobId` field and the discarded `uint16` — `libs/atlas-packet/dragon/clientbound/spawn.go` (not independently re-derived from IDA by this audit; accepted per brief).
- `int32` coordinates end-to-end (`dragon/model.go:21-22`, `storedDragon.X/Y`, `RestModel.X/Y`, `StatusEventCreatedBody.X/Y`) — consistent everywhere, no narrowing conversion found.
- No database in atlas-dragons — confirmed, no `gorm`/`database.Connect` import anywhere in the service.
- `MOVE_DRAGON` serverbound packet has no identity field — confirmed at `socket/handler/dragon_move.go:18-22` and `dragon/processor.go:136-142`, consistent with the design doc's stated reasoning.

## Summary

### Blocking (must fix)
- FILE-02: `atlas-channel/dragon/model.go` collapses `Model`+`RestModel`+`Extract`+JSON:API methods with no `rest.go` in the package.
- EXT-01: neither client `RestModel` (`atlas-dragons/character/rest.go`, `atlas-channel/dragon/model.go`) implements `SetToOneReferenceID`/`SetToManyReferenceIDs`.
- EXT-02: no httptest-backed integration test for either REST client package.
- DOM-17: `dragon/resource.go` `handleGetDragonByCharacterId` maps every processor error to 404, not just not-found.
- Swallowed-error correctness bug: `dragon/processor.go` `Destroy` treats every registry `Get` error (including genuine Redis failures) as "already gone," silently no-opping instead of propagating — leaks orphaned registry/field-index entries with no error surfaced.
- Testing: zero test coverage for both `atlas-dragons` Kafka consumer packages (`kafka/consumer/dragon`, `kafka/consumer/character`).
- Testing: `atlas-channel/kafka/consumer/dragon` tests exercise `excludesOwner`/`handles` helpers the handlers never call — no regression net against a `ForSessionsInMap`/`ForOtherSessionsInMap` swap.
- FILE-05: `atlas-dragons/character/model.go` combines `Model` and `Builder` with no `builder.go`.

### Non-Blocking (should fix)
- `dragon/registry.go` `Put`/`Remove` silently discard `fieldIdx.Remove` errors with no log (self-healing via stale-entry skip in `GetInField`, but zero visibility on persistent failure).
- `socket/handler/dragon_move.go:29` discards the MOVE command's Kafka-emit error with no log line at all (Create/Destroy paths log; Move does not).
- No structural/mechanical guard (only prose + partial literal-JSON tests) protects either cross-module Kafka contract from silent drift; the character-status mirror has no test protection whatsoever.
