# Backend Audit — atlas-parcel (Round 2)

- **Service Path:** services/atlas-parcel (task-241)
- **Also examined (fix-commit-scoped, not a full re-audit):** `libs/atlas-saga`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-channel/atlas.com/channel/parcel`, `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_{send,receive}.go`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-20
- **Baseline:** `docs/tasks/task-241-duey-parcel-delivery/audit-parcel.md` / `.json` (round 1, 11 blocking findings) — NOT overwritten
- **Fix commits examined:** `b550d80d9` (DOM-02/03/05/08/11/30), `8f23610d8` (DOM-21), `a95b7c067` (channel parcel package split), `a6e5e529f`/`3468f1370`/`ce05ea9db` (DOM-20 table-driven tests), `93347e924` (compose/bruno/docs scaffolding, SCAFFOLD-06/08)
- **Build:** PASS
- **Tests:** atlas-parcel: all packages `ok` (0 failures); corroborating: `libs/atlas-saga`, `atlas-saga-orchestrator`, `atlas-channel` all build clean; `atlas-saga-orchestrator` full test suite `ok`
- **Overall:** PASS

## Build & Test Results

```
$ cd services/atlas-parcel/atlas.com/parcel && go build ./...
(clean, exit 0)

$ go test ./... -count=1
?   	atlas-parcel	[no test files]
?   	atlas-parcel/kafka/consumer	[no test files]
ok  	atlas-parcel/kafka/consumer/custody	0.027s
?   	atlas-parcel/kafka/message	[no test files]
?   	atlas-parcel/kafka/message/custody	[no test files]
?   	atlas-parcel/kafka/message/parcel	[no test files]
?   	atlas-parcel/kafka/producer/custody	[no test files]
?   	atlas-parcel/kafka/producer/parcel	[no test files]
ok  	atlas-parcel/parcel	0.106s
?   	atlas-parcel/rest	[no test files]

$ cd libs/atlas-saga && go build ./...            # clean
$ cd services/atlas-saga-orchestrator/.../saga-orchestrator && go build ./... && go test ./... -count=1
(clean; every non-`?` package `ok`, including `atlas-saga-orchestrator/saga` and `atlas-saga-orchestrator/parcel`)

$ cd services/atlas-channel/atlas.com/channel && go build ./...
(clean, exit 0)
```

## Applicability

Unchanged from round 1 for every family whose trigger didn't move; the fix
commits added no new package and touched no new rule family. Two
machine-checked gates were re-run to settle the tenant-sweep judgment call:

| Gate | Result |
|---|---|
| `./tools/scope-guard.sh` (scopeguard analyzer, fleet-wide) | `scopeguard: 89 module(s), 8 parallel` — exit 0, clean |
| `./tools/goroutine-guard.sh` | exit 0, clean |
| `./tools/service-registration-guard.sh` | `service-registration-guard: clean` |
| `./tools/gen-routes.sh --check` | `gen-routes: up to date` |

## Round-1 Findings — Disposition

### 1. DOM-02 — `Model.ToEntity()` in `entity.go`

**CLEARED.** `parcel/entity.go:130` now defines `func (m Model) ToEntity() Entity`, in `entity.go` as required. `entityFromModel` no longer exists; `builder.go` no longer defines it (verified `grep -n "^func" builder.go` carries no `Make`/`ToEntity`/`entityFromModel`).

### 2. DOM-03 — `Make(Entity) (Model, error)` in `entity.go`

**CLEARED.** `parcel/entity.go:107` defines `func Make(e Entity) (Model, error)`, in `entity.go`.

### 3. DOM-05 — `TransformSlice` in `rest.go`, used by list handlers

**CLEARED.** `parcel/rest.go:91` defines `func TransformSlice(ms []Model) ([]RestModel, error)`. `parcel/resource.go:142` (`writeParcels`) calls it: `res, err := TransformSlice(ms)`. No inline `model.SliceMap(Transform)` remains in `resource.go`.

### 4. DOM-08 — POST/PATCH routes via `RegisterInputHandler[T]`

**CLEARED.** `parcel/resource.go:47` — `registerNotifyPatch := rest.RegisterInputHandler[NotifyRestModel](l)(db)(si)`; `resource.go:53` registers `/parcels/{parcelId}/notify` PATCH through it. `NotifyRestModel` (`rest.go:136-156`) is a flat, JSON:API-conformant request model (DOM-18/19 both hold). `resource_test.go:87` adds a `notifyBody` helper and the two notify subtests (`resource_test.go:330,348`) now send a marshaled jsonapi body, matching atlas-channel's actual caller contract per the commit message.

### 5. DOM-11 — Providers lazy via `database.Query`/`SliceQuery`

**CLEARED.** `parcel/provider.go` — every function (`ById:18`, `ByRecipient:29`, `BySender:41`, `ReceivableByRecipient:53`, `ReceivableByRecipientAnyWorld:70`) now composes `database.Query[Entity]`/`database.SliceQuery[Entity]` directly, no eager `Find`/`First` + `model.FixedProvider`. Confirmed `database.Query`/`SliceQuery` (`libs/atlas-database/provider.go:11,20`) return `model.Provider[E]` — lazy composition, matching the rule's required shape. The map-keyed WHERE clauses (GORM zero-value elision guard) are preserved as chained `db.Where(...)` before the query composition, not as an eager pre-execution.

### 6. DOM-30 — `parcel/notification_task.go:165-166` direct producer call

**CLEARED**, and the judgment call is settled: `kafka/message/message.go` (added by `b550d80d9`) is a line-for-line structural mirror of `services/atlas-npc-shops/atlas.com/npc/kafka/message/message.go`'s `Buffer`/`Emit` (diffed both files; parcel's version omits `EmitWithResult`, which it does not need — no other difference). `notification_task.go:167-169` now wraps the emit: `perr := buffer.Emit(kp)(func(mb *buffer.Buffer) error { return mb.Put(parcelmsg.EnvStatusEventTopic, parcelproducer.ParcelArrivedStatusEventProvider(...)) })`.

Per `patterns-kafka.md`'s own DOM-30 verification section (`resources/patterns-kafka.md:58-90`), the documented **pass criterion is precisely** "emits through `AndEmit` + `message.Buffer`" — the `Emit(p.p)(func(mb *message.Buffer) error {...})` shape shown at line 25-34 of the same doc. Nothing in the rule or its verification procedure requires the heavier outbox-backed form (`atlas-fame`); that form is not cited anywhere in the checklist or the linked pattern doc as a requirement, only `AndEmit`+`Buffer` is. The implementer's round-A note ("matches the common `AndEmit` idiom... rather than the heavier outbox-backed form") is therefore correct: the finding required the documented `AndEmit`/`Buffer` remediation, which has been supplied, and no stricter remediation is enforceable under the current checklist.

### 7. DOM-30 — `kafka/consumer/custody/consumer.go:130,158` direct producer calls

**CLEARED.** `handleAcceptToParcel` (`consumer.go:80-134`) now wraps `AcceptCustody` + the ACCEPTED emit inside `buffer.Emit(p)(func(mb *buffer.Buffer) error {...})` (`consumer.go:86-131`); `handleReleaseFromParcel` (`consumer.go:150-167`) does the same for `ReleaseCustody` + RELEASED (`consumer.go:156-163`). The two remaining direct `producer.ProviderImpl`-derived calls in this file — `consumer.go:132` and `consumer.go:165` (`ErrorStatusEventProvider`) — are both on the post-failure branch (reached only after `buffer.Emit` has already returned a non-nil error), which is DOM-30's documented exception 1. `handleRestoreParcel`/`handleRemoveParcel` (`consumer.go:169-207`) emit directly only on their own failure branches too — same exception, unchanged from round 1 where they were not flagged.

### 8. DOM-20 — table-driven tests

**CLEARED for every test file with subtests to convert.** All 8 of the 9 `_test.go` files in scope now contain a `tests := []struct {` table feeding `t.Run`: `administrator_test.go:25`, `builder_test.go`, `provider_tenant_test.go`, `task_test.go`, `processor_test.go` (×4 tables), `resource_test.go`, `notification_task_test.go`, `kafka/consumer/custody/consumer_test.go`. Spot-checked `administrator_test.go:19-50` (`TestUpdateStatusIfPending`) — a genuine data table (name/initialStatus/initialResolve/newStatus/wantRows/wantStatus/wantResolve fields), not a cosmetic wrapper.

`entity_test.go` has zero `t.Run` calls (`grep -c "t.Run(" parcel/entity_test.go` → 0) and was not touched by any of the three DOM-20 fix commits. Its five scenarios are each an independent top-level `func Test*(t *testing.T)` (`entity_test.go:20,26,38,52,78`), not a set of bare `t.Run` subtests sharing one parent function — the shape DOM-20 exists to replace. There is nothing in this file for the `tests := []struct{...}` + `t.Run` pattern to apply to; disposed as N/A, not carried forward as a residual finding, per the same reasoning the round-3 commit (`ce05ea9db`) documented.

### 9. SCAFFOLD-06 — docker-compose entry

**CLEARED.** `deploy/compose/docker-compose.core.yml:481-497` — an `atlas-parcel:` service block using `<<: *atlas-defaults`, matching the peer convention (verified `atlas-storage` also appears only in `docker-compose.core.yml`, not the other two compose files). Carries `DB_NAME`, all three topic env vars, and `PARCEL_EXPIRY_INTERVAL_SECONDS`. `PARCEL_NOTIFICATION_INTERVAL_SECONDS` is omitted but has a coded default (`DefaultNotificationInterval`, `notification_task.go:28`) — same optional-with-fallback shape as the included expiry interval; not a SCAFFOLD-06 gap.

### 10. SCAFFOLD-08 — Bruno collection

**CLEARED.** `services/atlas-parcel/.bruno/` exists with `bruno.json`, `collection.bru`, environment files, and six `.bru` request files covering every route in `parcel/resource.go` (list-by-recipient, list-by-sender, get, discard, notify, character parcel-status).

### 11. DOM-21 — `SourceInventoryType`/`InventoryType` raw `byte`

**CLEARED.** `libs/atlas-saga/payloads.go:992` (`TransferToParcelPayload.SourceInventoryType`) and `:1082` (`WithdrawFromParcelPayload.InventoryType`) now both `inventory.Type`. The compiler-driven follow-through is complete and correct: `saga-orchestrator/saga/processor.go:2200,2238,2344` convert to the still-`byte`-typed sibling payload fields (`byte(payload.SourceInventoryType)`, `byte(payload.InventoryType)`) rather than propagating the type further than the checklist required; `atlas-channel/socket/handler/duey_action_send.go` and `duey_action_receive.go` were followed through to their own typed `it inventory.Type` locals (`duey_action_send.go:43,61`) with no residual `byte` cast at the call boundary. `go build ./...` and `go test ./... -count=1` both clean in `atlas-saga-orchestrator`; `go build ./...` clean in `atlas-channel`.

## Judgment Calls — Explicit Rulings

### DOM-30 / Kafka emit form

**Ruling: the lighter `AndEmit` form is sufficient; no stricter remediation was required.** See finding 6 above — the checklist's own verification procedure (`patterns-kafka.md:58-90`) names `AndEmit` + `message.Buffer` as the pass criterion, cites no outbox requirement, and the parcel service's new `kafka/message/message.go` is a verified line-for-line structural match to the repo's own canonical `AndEmit` example (`atlas-npc-shops`). Holding this finding to the heavier `atlas-fame` shape would be enforcing a rule the checklist does not state.

### `WithoutTenantFilter` sweeps (`notification_task.go:144`, `task.go:144`)

**Ruling: verified correct, not accepted on rationale alone.** Read both `Run()` bodies directly rather than trusting the allowlist text:

- `task.go:144` (`ExpiryTask.Run`): `noTenantCtx` is used only for the claim query (`ClaimExpired`, `task.go:148`). Each claimed row's tenant is reconstructed at `task.go:164` (`tenant.Create(m.TenantId(), ...)`) and re-entered at `task.go:174` (`tenant.WithContext(t.ctx, tm)` — built from `t.ctx`, explicitly **not** `noTenantCtx`, per the comment at `task.go:169-173` explaining why: `WithoutTenantFilter` also disables the `tenant:create` callback's `tenant_id` injection). The return-leg `Create` call at `task.go:205` runs against `t.db.WithContext(tenantCtx)` — the properly-scoped context — not the bypassed one.
- `notification_task.go:144` (`NotificationTask.Run`): same shape. `noTenantCtx` scopes only `ClaimNotifiable` (`:148`). Per-row tenant reconstruction at `:159`, re-entry (plus this pod's `envContext`) at `:164`, and the Kafka emit at `:166-169` runs under `tenantCtx`, not `noTenantCtx`.

Both `ClaimExpired` and `ClaimNotifiable` (`parcel/administrator.go:77-97,118-144`) are themselves single-statement, claim-by-update SQL (`UPDATE ... RETURNING` gated by a `status`/`last_notified` compare-and-swap predicate) run under the bypassed context — this is the cross-tenant sweep itself, not a leak past it, and is exactly what the allowlist rationale describes.

`tools/scopeguard/callsite-allowlist.txt:36-37` — both entries' cited line numbers (`task.go:144,164,174,205`; `notification_task.go:144,159,164,166`) match the code read above exactly, not approximately. `./tools/scope-guard.sh` (the machine-enforced gate) exits 0 across all 89 modules, confirming no unallowlisted `WithoutTenantFilter` call site exists anywhere in the fleet, this pair included.

**Ruling stands: legitimate, verified cross-tenant background sweeps. Not a finding.**

## New Surface Introduced By the Fix Commits — Checked, Not Assumed

| Surface | Check | Result |
|---|---|---|
| `parcel/entity.go` (Make/ToEntity moved here) | FILE-04 (`Entity`/`Migration`/`TableName` in `entity.go`) still holds after the move added ~68 lines | PASS — `entity.go:39,90,102` (struct/`Migration`/`TableName`), `Make`/`ToEntity` are FILE-05's Model/Entity-mapping content, same file, no violation |
| `parcel/builder.go` (Make/ToEntity removed) | FILE-06 (no package-catch-all) still holds — builder.go now only Builder/setters/validate/Build | PASS — `grep -n "^func\|^type" builder.go` shows only `Builder` type + `NewBuilder`/`Set*`/`validate`/`Build` |
| `kafka/message/message.go` (new file) | FILE placement — single-purpose Buffer/Emit helper, not a catch-all | PASS — one type (`Buffer`), `NewBuffer`/`Put`/`GetAll`/`Emit`, no RestModel/Processor/Entity content |
| `resource_test.go` new `notifyBody` helper | Exercises the new `RegisterInputHandler[NotifyRestModel]` route with a real jsonapi body, not a stub | PASS — `resource_test.go:87-...`, used at `:330,348` |
| `services/atlas-channel/atlas.com/channel/parcel` split (`a95b7c067`) | FILE-01/02/05/06, DOM-01 (not part of atlas-parcel's own scope, but the split's correctness was asked to be verified) | PASS — `processor.go` now holds only `Processor`/`NewProcessor`/methods; `model.go` holds `Model`+accessors+`WireId`+`ToPacket`; `rest.go` holds `RestModel`+`Extract`+JSON:API methods; `builder.go` holds `Builder`/`NewBuilder`/setters/`validate`/`Build` — verified via `grep -n "^type\|^func"` on all four files, no cross-contamination |
| `libs/atlas-saga/payloads.go` DOM-21 fix blast radius | Every `byte`→`inventory.Type` call site across 3 services converts correctly, no silently-mismatched wire encoding | PASS — `inventory.Type` is `int8` with no custom `(Un)Marshal*`; commit message's claim ("byte-identical JSON encoding for values 1-5") checked against `libs/atlas-constants/inventory/constants.go:9` (`type Type int8`, no custom marshaler in that file) |

No regression found in any file touched by the six fix commits.

## Checklist Results — Full Re-run (packages touched by fix commits)

### parcel (domain package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` builder pattern | PASS | `parcel/builder.go:14,41,45-...,` `validate`+`Build` unchanged in shape from round 1 |
| DOM-02 | `Model.ToEntity()` in `entity.go` | PASS | `parcel/entity.go:130` |
| DOM-03 | `Make(Entity)` in `entity.go` | PASS | `parcel/entity.go:107` |
| DOM-04 | `Transform` in `rest.go` | PASS | `parcel/rest.go:61-87` (unchanged from round 1) |
| DOM-05 | `TransformSlice` in `rest.go`, used by list handler | PASS | `rest.go:91`; `resource.go:142` |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `parcel/processor.go:42` (unchanged) |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `resource.go` (unchanged call sites) |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]` | PASS | `resource.go:46-47,52-53` |
| DOM-09 | Every `Transform(`/`TransformSlice(` call checked | PASS | `resource.go:142-146` checks `err` before use |
| DOM-11 | Providers lazy via `database.Query`/`SliceQuery` | PASS | `provider.go:18,29,41,53,70` |
| DOM-16 | `administrator.go` holds writes, incl. claim-by-update sweeps | PASS | `administrator.go:77,118` (`ClaimExpired`, `ClaimNotifiable`) plus round-1 writes |
| DOM-17 | Domain errors → HTTP status | PASS | `resource.go:251-253` (notify 404) unchanged pattern |
| DOM-18 | JSON:API interface | PASS | `NotifyRestModel` (`rest.go:140-156`) |
| DOM-19 | Flat request models | PASS | `NotifyRestModel` (`rest.go:136-138`), single field |
| DOM-27 | `server.WriteErrorResponse` for 500s | PASS | `resource.go:258,264` (notify handler's new error branches) |
| DOM-30 | `AndEmit`+`Buffer`, not direct producer call, on DB-write success path | PASS | `notification_task.go:167-169`; `kafka/consumer/custody/consumer.go:86-131,156-163` |
| DOM-31 | Tenant/trace in context only | PASS | `NotifyRestModel` carries no tenant field; sweep tasks pass tenant via `tenant.WithContext`, never a struct field |
| DOM-32 | Routes via `server.RegisterHandler`/`RegisterInputHandler[T]` | PASS | `resource.go:45-47` |
| FILE-01..06 | File placement | PASS | See "New Surface" table above |
| DOM-20 | Table-driven tests | PASS (8/9 files); N/A (`entity_test.go`, no subtests to convert) | See finding 8 above |
| DOM-24 | Emit-reaching tests stub the producer | PASS | `notification_task_test.go` `producertest.InstallCapturing()` in `TestMain` (unchanged); `consumer_test.go` per-test injection (unchanged, now feeding the `buffer.Emit`-wrapped handlers) |

### kafka/message (new support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | PASS | `message.go` is single-purpose (`Buffer`/`Emit`), mirrors the repo-wide idiom's file shape |

### kafka/consumer/custody

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-30 | `AndEmit`+`Buffer` on the success path | PASS | `consumer.go:86-131` (accept), `:156-163` (release) |
| DOM-24 | Producer stubbed in tests | PASS | `consumer_test.go` per-test injection (unchanged mechanism, re-verified against the now-wrapped handlers via `go test` passing) |

### Scaffolding

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SCAFFOLD-06 | docker-compose entry | PASS | `deploy/compose/docker-compose.core.yml:481-497` |
| SCAFFOLD-08 | Bruno collection | PASS | `services/atlas-parcel/.bruno/*` |

### Constants reuse (DOM-21)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | `inventory.Type` reused for parcel payload inventory fields | PASS | `libs/atlas-saga/payloads.go:992,1082` |

## Security Review

Trigger did not fire — unchanged from round 1. SEC-01..04: N/A.

## Not evaluable from the diff

- Full DOM/FILE/REST sweep of `services/atlas-channel/atlas.com/channel/parcel` and `duey_action_{send,receive}.go` beyond the specific fix-commit-scoped checks recorded above (FILE split correctness, DOM-21 type propagation) — this round's mandate was to verify the six named fix commits and the two flagged judgment calls, not a full re-audit of atlas-channel; a sibling reviewer owns that service's own audit.
- SCAFFOLD-07 (channel writer/handler opcode-template seeding) and DOM-25 (packet dispatcher codecs) — same as round 1, out of this audit's assigned scope.

## Summary

### Blocking (must fix)
- None. All 11 round-1 findings are cleared with file:line evidence; no new blocking finding introduced by the fix commits.

### Non-Blocking (should fix)
- None identified.

### Not evaluable
- 2 items — see section above (both carried forward from round 1's scope boundary, neither reopened by this round's fix commits).
