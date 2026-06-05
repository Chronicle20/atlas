# Backend Audit — task-082 (monster-book cover mob-id)

- **Scope:** Changed Go packages on branch `task-082-monsterbook-cover-mobid` (diff vs `main`)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-06-05
- **Build:** PASS (atlas-monster-book, atlas-channel)
- **Tests:** PASS (monster-book all packages; channel monsterbook + socket/writer; atlas-packet clientbound)
- **Overall:** PASS

## Build & Test Results

- `services/atlas-monster-book/atlas.com/monster-book` — `go build ./...` exit 0; `go test ./... -count=1` all OK (collection, data/consumable, kafka consumers all pass).
- `services/atlas-channel/atlas.com/channel` — `go build ./...` exit 0; `go test ./monsterbook/... ./socket/writer/...` OK.
- `libs/atlas-packet` — `go test ./character/clientbound/...` OK.
- `go vet` clean on all changed packages.

## Domain Discovery

- `collection/` — **domain package** (has `model.go`). Full DOM checklist applies. Note: this domain's REST handlers live in the shared `rest/` + `character/` packages (no `resource.go` in the package itself), so handler-oriented checks (DOM-07/08/12/13/14/17) are evaluated against the actual call sites.
- `data/consumable/` — **external HTTP client package** (new). EXT checklist applies; it is not a persistence domain (no entity/builder), so DB-oriented DOM checks (DOM-01/02/03/10/11/15/16) are N/A.
- `monsterbook/` (atlas-channel) — outbound client / view-model package. EXT-01 ref-stub check applies; persistence DOM checks N/A.
- `socket/writer/character_info.go` — packet writer (1-line crash fix + test).

## collection/ (domain) Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go exists | PASS | collection/builder.go (NewModelBuilder, fluent setters, Build() validates characterId at builder.go:60-62) |
| DOM-02 | ToEntity() | PASS | collection/model.go:45 `func (m Model) ToEntity() entity` (now maps CoverMobId at :49) |
| DOM-03 | Make(Entity) | PASS | collection/builder.go:88 `func Make(e entity) (Model, error)` (sets CoverMobId at :93) |
| DOM-04 | Transform | PASS | collection/rest.go:33 `func Transform(m Model) (RestModel, error)` (maps CoverMonsterId at :41) |
| DOM-05 | TransformSlice | N/A | Single-resource domain (collection is per-character singleton; no list endpoint). Not introduced by this change. |
| DOM-06 | Processor takes FieldLogger | PASS | collection/processor.go:78 `NewProcessor(l logrus.FieldLogger, ...)`; consumable/processor.go:19 same |
| DOM-07 | Handlers pass d.Logger() | PASS | character/resource.go (cover-set call site) and Kafka consumer use injected logger; no `logrus.StandardLogger()` in changed code |
| DOM-08 | POST/PATCH typed input | PASS | Cover-set is driven via Kafka command (kafka/consumer/monsterbook/consumer.go:85) and a typed handler in character/resource.go:76; no untyped RegisterHandler for a body endpoint introduced |
| DOM-09 | Transform errors handled | PASS | Transform returns (RestModel, error); no `_, _ :=` discards introduced (rest.go:33) |
| DOM-10 | Test DB tenant callbacks | WARN (pre-existing) | collection/administrator_test.go:13 `newDB` does NOT call `database.RegisterTenantCallbacks`. Tests pass `tenantId` explicitly to admin/provider functions, which use literal `WHERE tenant_id = ?` (administrator.go:55, provider.go:13). Pre-existing pattern; the diff did not touch `newDB`. Not a regression. |
| DOM-11 | Providers lazy | PASS | collection/provider.go:13 uses `database.Query[entity](...)`; unchanged by diff and compliant |
| DOM-12 | No os.Getenv in handlers | PASS | No `os.Getenv` in collection/ or changed handlers |
| DOM-13 | No cross-domain logic in handlers | PASS | Cross-domain (card→mob resolution) lives in processor `resolveCoverMobId` (processor.go:226), not in any handler |
| DOM-14 | Handlers don't call providers directly | PASS | Cover flow goes handler/consumer → `colp.SetCoverAndEmit` (processor) → administrator |
| DOM-15 | No direct entity writes in handlers | PASS | All writes via `setCover` administrator (administrator.go:55); no `db.Create/Save/Delete` in handlers |
| DOM-16 | administrator.go for writes | PASS | collection/administrator.go `setCover` now persists cover_mob_id (administrator.go:61) |
| DOM-17 | Domain error → HTTP status | PASS | Typed sentinels (ErrCoverNotOwned, ErrCardIdOutOfRange) defined for mapping (processor.go:26-63); resolution failure intentionally NEVER errors (fail-safe) so cannot produce a 5xx |
| DOM-18 | JSON:API iface on RestModel | PASS | collection/rest.go:22-31 GetName/GetID/SetID present |
| DOM-19 | Flat request models | PASS | PatchInput (rest.go:46) is flat — no nested Data/Type/Attributes |
| DOM-20 | Table-driven tests | PASS | collection/processor_test.go:142 `TestResolveCoverMobId` uses `cases := []struct{...}` + `t.Run`; covers clear/resolve/error/non-book/zero-mob |
| DOM-21 | No duplication of atlas-constants | PASS | cover mob id typed `monster.Id` from `libs/atlas-constants/monster` (model.go:16, builder.go:17, rest.go:18, administrator.go signature). `monster.Id` is `uint32` (libs/atlas-constants/monster/constants.go:3). No redeclared type/constant. |
| DOM-22 | Dockerfile lib coverage | PASS | New direct requires (atlas-constants, atlas-kafka, atlas-rest, atlas-tracing) all present in shared root `Dockerfile` mod-only block (lines 32/34/42/50), source block (61/63/71/79), and `go.work` (4/6/14/22). This repo uses the `ARG SERVICE` shared Dockerfile, not the per-service 4-block template — equivalent coverage verified. |
| DOM-23 | Kafka topic naming | N/A | No new COMMAND_TOPIC_*/EVENT_TOPIC_* constants introduced; cover-set reuses the existing monster-book command topic |
| DOM-24 | Kafka producer stubbed in tests | PASS | collection/testmain_test.go and kafka/consumer/monsterbook/testmain_test.go both call `producertest.InstallNoop()` (shared lib); no `t.Cleanup(producer.ResetInstance)`. processor_test SetCover test asserts only the pre-producer validation path. |

## data/consumable/ (external client) — EXT Checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | JSON:API ref interface on target struct | PASS | rest.go:34-37 SetToOneReferenceID / SetToManyReferenceIDs (+ GetReferences/GetReferencedIDs) implemented as no-ops; test TestRestModel_UnmarshalWithRelationships (rest_test.go:29) proves a `relationships` block unmarshals |
| EXT-02 | httptest-backed integration test | PASS | rest_test.go:60 `TestGetById_RoundTrip` spins `httptest.NewServer`, serves a JSON:API body incl. `relationships`, asserts populated domain Model |
| EXT-03 | 404 distinguished from other errors | PASS | rest_test.go:88 `TestGetById_NotFound` asserts `errors.Is(err, requests.ErrNotFound)`; `requests.GetRequest` maps only genuine 404→ErrNotFound (libs/atlas-rest/requests/get.go:78). Caller folds all failures to 0 by design (fail-safe FR-5), which is correct, not error-masking at the client layer. |
| EXT-04 | URL via RootUrl(domain) | PASS | requests.go:17 `requests.RootUrl("DATA")`; no hardcoded DNS. `SetBaseURLForTest` is an explicit, documented test seam (requests.go:30) restoring the prior provider via the returned closure. |
| Tenant/version propagation | — | PASS | `requests.GetRequest` auto-applies `TenantHeaderDecorator(ctx)` (libs/atlas-rest/requests/decorated.go:12); processor.go:26 passes `p.ctx`. Tenant + region/version headers propagate; round-trip test runs under a tenant context. |

## atlas-channel monsterbook/ + writer — Checklist

| Item | Status | Evidence |
|------|--------|----------|
| EXT-01 ref stubs on client RestModel | PASS | monsterbook/rest.go:60/65 (Collection) and :96/97 (Card) implement the relationship stubs |
| Crash fix correctness | PASS | socket/writer/character_info.go:60 now writes `uint32(mb.CoverMonsterId())` (mob id) instead of card id; `CoverMonsterId` plumbed model.go:23 → processor.go Collection field → rest.go Extract:112 |
| Wire-level test | PASS | socket/writer/character_info_test.go:15 `TestCharacterInfoBody_CoverIsMobId` round-trips the packet and asserts `MonsterBookCover() == 100100` (mob id, not card 2380000) |
| Fail-safe value | PASS | Resolution failure stores 0 (processor.go:226-235); cover 0 → client receives 0, which is the documented un-poison value |

## Security Review

N/A — no authentication, authorization, token, or redirect handling in scope. No JWT/secret code touched. SEC-01..04 not applicable.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- **DOM-10 (pre-existing, not a regression):** `collection/administrator_test.go:13 newDB` does not register `database.RegisterTenantCallbacks`. The package's providers/administrators pass `tenantId` explicitly with literal `WHERE tenant_id = ?`, so tests are correct, but they don't exercise the automatic-tenant-filtering callback path the guidelines prefer. Out of scope for this change; flag only.

### Verdict
PASS. The card→mob resolution is correctly isolated in the processor layer with a strict fail-safe (never errors, never propagates a client-crashing value), the new outbound atlas-data client satisfies the full EXT checklist (ref stubs, httptest round-trip, ErrNotFound distinction, RootUrl, tenant/version propagation), `monster.Id` from atlas-constants is used throughout (DOM-21), the Kafka producer is stubbed via the shared `producertest` lib (DOM-24), and the one-line packet crash fix is covered by a byte-level round-trip test asserting the mob id reaches the wire.
