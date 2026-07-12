# Backend Audit — atlas-character-factory

- **Service Path:** services/atlas-character-factory/atlas.com/character-factory
- **Scope:** task-102 modified packages only → `configuration/projection` (1 file, `subscriber.go`)
- **Guidelines Source:** backend-dev-guidelines skill + backend-guidelines-reviewer definition (FILE-01..06)
- **Date:** 2026-07-10
- **Build:** PASS
- **Vet:** PASS
- **Tests:** 9 passed, 0 failed (`configuration/projection`)
- **Overall:** PASS

## Build & Test Results

```
go build ./...                              → exit 0
go vet ./configuration/projection/...       → exit 0
go test ./configuration/projection/... -count=1 → ok  atlas-character-factory/configuration/projection  0.006s
```

## task-102 Change Under Audit

`configuration/projection/subscriber.go:93` — single-line swap
`consumer.ReadEndOffsets(...)` → `consumer.ReadReplayableEndOffsets(...)`.

**Verdict: correct and well-motivated.** `ReadReplayableEndOffsets`
(`libs/atlas-kafka/consumer/offsets.go:35`) collapses a retention-purged
partition's `(log-start, high-water-mark)` pair to `0` via `ReplayableEnd`
(offsets.go:42). The catch-up gate `evaluateLocked`
(`configuration/projection/caughtup.go:101`) treats `end == 0` as trivially
caught up. This is exactly the fix for the config-projection retention-purge
readiness-wedge failure mode; the previous `ReadEndOffsets` returned an
unreachable high-water mark for a purged partition, which no FirstOffset
consumer could ever observe → gate never flips → pod stays 0/1. No stale
`ReadEndOffsets` callers remain in the service (grep clean).

## Package Classification (Phase 2)

`configuration/projection` has **no `model.go` and no `resource.go`** →
**Support package** (a consumer-side in-memory projection of
atlas-configurations' tenant config-status outbox). The full DOM-* domain
checklist and SUB-* sub-domain checklist do **not** apply. The File
Responsibilities Checklist (FILE-01..06) applies to every package including
support, and was run in full below.

Files (non-test): `caughtup.go`, `envelope.go`, `state.go`, `subscriber.go`.

## File Responsibilities Checklist (FILE-01..06)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | `Processor` in `processor.go` | PASS (N/A) | No `type Processor`, `ProcessorImpl`, or `NewProcessor` anywhere in the package — a projection consumer, not a domain processor. No misplacement possible. |
| FILE-02 | `RestModel`+`Transform`/`Extract`+JSON:API in `rest.go` | PASS (N/A) | No `RestModel`/`Transform`/`Extract`/`GetName`/`GetID`/`SetID` defined here. `envelope.go` defines a wire `TenantEnvelope` + `DecodeTenantEnvelope` (a Kafka envelope decoder, not a JSON:API DTO); the JSON:API model consumed is the external `tenant.RestModel` (`state.go:16`). No misplacement. |
| FILE-03 | Cross-service request funcs in `requests.go` | PASS (N/A) | No `requests.RootUrl`/`GetRequest`/`PostRequest`/`getBaseRequest` in the package. Data arrives via Kafka, not REST client calls. |
| FILE-04 | Entity+`Migration`+`TableName` in `entity.go` | PASS (N/A) | No GORM `entity` struct, `Migration`, or `TableName` — the projection is in-memory only (`state.go:14` `map[uuid.UUID]tenant.RestModel`). |
| FILE-05 | Builder/Model/administrator/provider/state placement | PASS | No `Builder`, domain `Model`, `Create*/Update*/Delete*` DB writes, or `database.Query`/`SliceQuery` readers exist. `state.go` holds the in-memory `State` store (`state.go:14`); naming reads as a projection-state store rather than the table's "domain enums" sense, but nothing is *misplaced* — there is no canonical domain symbol living in the wrong file. |
| FILE-06 | No package-named catch-all bundling ≥2 responsibilities | PASS | No `projection.go` catch-all exists. Each file is single-purpose: `caughtup.go` = readiness gate (`CaughtUp`), `envelope.go` = wire decode + package doc, `state.go` = in-memory snapshot store, `subscriber.go` = Kafka consumer wiring. None carries ≥2 File-Responsibilities roles. This is the clean split the task-102 `wallet.go` finding warns about — this package does NOT reproduce it. |

## Other Applicable Checks

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Logger type is `logrus.FieldLogger` | PASS | `subscriber.go:36,60,92` all take `logrus.FieldLogger`, never `*logrus.Logger`. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS (N/A) | No emit call sites (`AndEmit`/`message.Emit`/`producer.Produce`/`ProviderImpl`) in the package or its tests; `subscriber.handleTenant` is consume-only (grep clean). No producer stub required. |

## Non-checklist Observations (informational, not violations)

- `projection_test.go` uses individual `func Test*` cases rather than the
  DOM-20 `tests := []struct{}` + `t.Run` table form. DOM-20 is a domain-package
  criterion and does not bind a support package; the coverage is adequate
  (envelope decode/reject, tombstone, apply/snapshot copy-semantics, bad-id
  rejection, and four catch-up gate transition cases). Noted, not a finding.

## Summary

### Blocking (must fix)
- None.

### Important
- None.

### Minor
- None.

Zero FAIL checks. Build, vet, and tests pass. The one-line task-102 change is
correct and the modified package is cleanly file-organized.
