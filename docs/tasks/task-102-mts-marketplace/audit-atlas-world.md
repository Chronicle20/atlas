# Backend Audit — atlas-world

- **Service Path:** services/atlas-world/atlas.com/world
- **Guidelines Source:** backend-dev-guidelines skill (reviewer def with File Responsibilities Checklist FILE-01..06)
- **Date:** 2026-07-10
- **Scope:** task-102 modified `configuration/projection` (1 file: `subscriber.go`, 1 line changed)
- **Build:** PASS
- **Vet:** PASS
- **Tests:** PASS (all packages; `configuration/projection` ok 0.007s)
- **Overall:** PASS

## Build & Test Results

```
go build ./...   -> exit 0 (clean)
go vet ./...     -> exit 0 (clean)
go test ./... -count=1 -> all ok; atlas-world/configuration/projection  ok  0.007s
```

## Scope Derivation

`git diff --stat main..HEAD` under `configuration/projection/` shows exactly one changed file:

```
subscriber.go | 2 +-  (1 insertion, 1 deletion)
```

The change (subscriber.go:91): `consumer.ReadEndOffsets(...)` → `consumer.ReadReplayableEndOffsets(...)`
inside `offsetsOrEmpty(...)`. Commit `27cc238067` — "survive retention-purged config topics + compact them."
Target function verified to exist at `libs/atlas-kafka/consumer/offsets.go:35`. Build passes; no leftover
references to the old `ReadEndOffsets` remain in the package.

## Package Classification

`configuration/projection` is a **support package**: no `model.go`, no `resource.go`. It is the
consumer-side mirror of atlas-configurations' outbox — a Kafka consumer + in-memory tenant-config
snapshot + readiness gate. Per the updated reviewer def, support status is NOT a checklist exemption:
the File Responsibilities Checklist (FILE-01..06) still applies.

Non-test files: `caughtup.go`, `envelope.go`, `state.go`, `subscriber.go` (+ `projection_test.go`).

## File Responsibilities Checklist (FILE-01..06)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | `Processor` in `processor.go` | PASS | Package defines no `Processor`/`ProcessorImpl`/`NewProcessor` (grep: NONE). Nothing to misplace. |
| FILE-02 | `RestModel` + `Transform`/`Extract` + JSON:API methods in `rest.go` | PASS | Package defines no `RestModel`/`Transform`/`Extract` (grep: NONE). It reuses `tenant.RestModel` from the sibling `configuration/tenant` package (state.go:16). Nothing to misplace. |
| FILE-03 | Cross-service request funcs in `requests.go` | PASS | No `requests.RootUrl(`/`GetRequest[`/`PostRequest[`/`getBaseRequest(` in package (grep: NONE). Pure Kafka consumer, no outbound HTTP. |
| FILE-04 | Entity + `Migration` + `TableName` in `entity.go` | PASS | No GORM `type entity`/`Migration(`/`TableName()` (grep: NONE). `TenantEnvelope` (envelope.go:17) is a Kafka wire DTO, not a persisted entity — correctly lives with its decode logic in `envelope.go`. |
| FILE-05 | Builder/model/administrator/provider/state placement | PASS | No builder, no domain `Model`, no `Create*/Update*/Delete*` DB writes, no `database.Query/SliceQuery` providers exist. `State` (state.go:14) is an in-memory snapshot store, not a persisted domain Model — correctly isolated in `state.go`. |
| FILE-06 | No package-named catch-all file | PASS | No `projection.go`. Each of the 4 non-test files is single-responsibility: `subscriber.go` (Kafka consumer registration + envelope handler), `caughtup.go` (readiness end-offset gate), `state.go` (RW-locked in-memory tenant snapshot), `envelope.go` (wire envelope type + decode + tombstone check + package doc). No file carries ≥2 of the table responsibilities. |

## Domain / Sub-Domain / EXT / Scaffolding Checklists

- **DOM-01..25:** Not applicable — no `model.go` (support package). Spot-checks on changed code:
  - **DOM-06** (processor takes `FieldLogger`): the package's handler funcs take `logrus.FieldLogger`
    (subscriber.go:59, 90), never `*logrus.Logger` — clean, though no Processor exists.
  - **DOM-23** (Kafka topic naming): the consumed topic is env-resolved via `s.TenantTopic`
    (`EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`, subscriber.go:23-25), not a Go literal — clean.
    Not modified by task-102.
  - **DOM-24** (producer stubbed in emitting tests): N/A. This is a consumer; `projection_test.go`
    exercises only `DecodeTenantEnvelope`, `State`, and `CaughtUp` — no emit call sites
    (`AndEmit`/`message.Emit`/`producer.Produce`) and no transitive producer path. `handleTenant`/
    `Start`/`offsetsOrEmpty` are not test-driven, so no unstubbed-emit hang risk.
- **SUB-01..04:** Not applicable — no `resource.go`, no REST handlers in this package.
- **EXT-01..04:** Not applicable — no cross-service HTTP client (`requests.*`) in this package.
- **SCAFFOLD-01..08:** Not applicable — no new service directory and no atlas-channel writer/handler
  registration in this change.
- **SEC-01..04:** Not applicable — atlas-world is not an auth/token service; the changed code is a
  config-status consumer with no token handling, redirects, or secrets.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- **Coverage (Minor):** the task-102 change lives in `offsetsOrEmpty` (subscriber.go:90-97), which has
  no unit test; `Start`, `handleTenant`, and `offsetsOrEmpty` are entirely untested. The swap to
  `ReadReplayableEndOffsets` is verified only by compilation + the lib function's existence
  (`libs/atlas-kafka/consumer/offsets.go:35`), not by a projection-level test. Not a FILE/DOM
  requirement violation — informational.
- **Convention (Minor):** the package doc comment sits atop `envelope.go` (envelope.go:1-7) rather than
  a dedicated `doc.go`. The guideline explicitly allows a thin doc placement, so this is not a finding —
  noted for completeness only.

### Verdict
Build/vet/test all clean. FILE-01..06 all PASS on the modified support package — the projection package
already observes one-responsibility-per-file (unlike the atlas-mts collapses that motivated the reviewer
update). Zero FAIL checks. **Overall: PASS.**
