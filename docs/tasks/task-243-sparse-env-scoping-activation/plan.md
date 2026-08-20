# Sparse Environment Service Binding and Activation — Implementation Plan

Spec: `design.md` (this folder). PRD: `prd.md`. Diagnosis: `diagnosis.md`.
Context for executors: `context.md` (this folder) — read it before Task 1; it
records three corrections to `design.md` that change *mechanism*, not
decisions.

Order matters only where stated. Tasks 1–7 are independent of each other.
Task 8 depends on Task 1. Tasks 9–10 depend on Task 8. Task 12 depends on
Tasks 1 and 11.

---

## Task 1: The deterministic service-id derivation

Implements D1, FR-1.3, FR-2.2. One script, one namespace constant, one
algorithm, used by CI and by nothing else at runtime.

### Files

- `tools/derive-service-id.sh` — **new file**; the single derivation site
- `tools/derive-service-id_test.sh` — **new file**; pinned-value suite

Module root: none (POSIX shell). `tools/verify.sh` gates both automatically —
`touched '^tools/.*\.sh$'` runs `tools/shell-guard.sh --require-shellcheck`
and `changed_tool_suites` selects `tools/derive-service-id_test.sh`
(`tools/verify.sh:488-500`).

Patterns to copy: `tools/gen-tenant-tables_test.sh` (test-script shape, PASS/FAIL
lines, `set -eu`); `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh`
(generator-script shape).

**Note on placement:** `design.md` §3.1 named
`deploy/k8s/overlays/pr/scripts/derive-service-id.sh`. It lives in `tools/`
instead so `verify.sh` gates it and its test — see `context.md` §"Deviation 1".
Its single-site property is unchanged.

### Step 1: Write the failing test

`tools/derive-service-id_test.sh` — a plain `sh` script, `set -eu`, echoing
`PASS:`/`FAIL:` per assertion and `exit 1` on any failure. It invokes
`tools/derive-service-id.sh` as a subprocess (not sourced).

Assertions, with the **exact** expected values (these are the contract; a
different namespace constant produces different values and the test must fail):

| # | invocation | expected stdout (exact) |
|---|---|---|
| 1 | `derive-service-id.sh login-service pr-1411` | `6439ca9c-d28d-5db9-821b-8dd93d318a25` |
| 2 | `derive-service-id.sh channel-service pr-1411` | `5a86d8e6-3167-5e74-9fc5-021d94001da2` |
| 3 | `derive-service-id.sh drops-service pr-1411` | `cbce66aa-facb-5766-8583-84c3478a6ba2` |
| 4 | `derive-service-id.sh world-service pr-1411` | `f80c02bc-2ac4-598e-a8e6-298e7e1d72b5` |
| 5 | `derive-service-id.sh character-factory pr-1411` | `a0bb4ad4-0c2b-5941-b297-fa4b6cf9403e` |
| 6 | `derive-service-id.sh drops-information-service pr-1411` | `87d2d5a6-f37d-5a1e-8e81-bfed3a239e69` |
| 7 | `derive-service-id.sh login-service pr-999` | `e7ae96a2-c484-5617-8e28-2178b60a8378` |
| 8 | `derive-service-id.sh login-service main` | `78d4984e-22dd-5284-8729-61627a5e603f` |
| 9 | `derive-service-id.sh channel-service pr-999` | `2e3b50b4-fb89-5af0-bb51-19749ecb734f` |
| 10 | `derive-service-id.sh channel-service main` | `dff6f040-d4aa-51fa-914b-ff1dff6f6a76` |

Plus four structural assertions:

- **§1.2 regression — the four nil-UUID services derive four DISTINCT ids.**
  Derive `drops-service`, `world-service`, `character-factory` and
  `drops-information-service` for `pr-1411`, sort them, and assert
  `sort -u | wc -l` is `4`. Name the failure
  `FAIL: nil-UUID services collided (design §1.2)`.
- **Determinism.** Two consecutive invocations with identical arguments emit
  byte-identical output.
- **Environment sensitivity.** `login-service pr-1411` != `login-service pr-1412`.
- **Argument validation.** Zero args, one arg, and an empty-string second arg
  each exit non-zero and print a usage line on stderr. `exit 0` with empty
  stdout is the failure mode this guards (the `uuidgen`-swallowed-error defect
  from `service-config.sh`'s header, in a new place).

### Step 2: Implement

```sh
#!/usr/bin/env sh
# Usage: derive-service-id.sh <service-type> <environment>
```

- `set -eu`.
- Reject fewer than two arguments, or an empty value for either, with a usage
  line on stderr and `exit 2`.
- The namespace constant, verbatim and commented as never-regenerable:

  ```sh
  # ATLAS_SERVICE_NS — the UUIDv5 namespace every derived SERVICE_ID depends
  # on. It appears here and NOWHERE else. Never regenerate it: changing it
  # re-keys every sparse environment's service-config row.
  # Reproducible rather than arbitrary, so the value can be re-derived if this
  # line is ever lost:
  #   uuid5(NAMESPACE_DNS, "service-config.atlas.chronicle20")
  ATLAS_SERVICE_NS=c8f90111-a0cf-513e-95e6-c54609e5dec0
  ```

- Emit `uuid5(ATLAS_SERVICE_NS, "<type>/<environment>")` via `python3`, with
  **no trailing newline** (`printf '%s'`), so command substitution and `sed`
  rendering both see exactly the 36 characters:

  ```sh
  python3 - "$ATLAS_SERVICE_NS" "$1/$2" <<'PY'
  import sys, uuid
  sys.stdout.write(str(uuid.uuid5(uuid.UUID(sys.argv[1]), sys.argv[2])))
  PY
  ```

- If `python3` is not on PATH, fail with a named error on stderr and a non-zero
  exit. Never fall through to empty stdout.

### Verification

```
sh tools/derive-service-id_test.sh
tools/shell-guard.sh --require-shellcheck
```

Both exit 0.

---

## Task 2: Legacy tenant reconciliation

Implements FR-3.1, FR-3.2, D4, and the FR-4.6 regression assertion.

### Files

- `libs/atlas-env/tenants.go` — one new arm in `Reconcile`
- `libs/atlas-env/tenants_test.go` — new cases

Module root: `libs/atlas-env`. `ApplyTenant` in
`libs/atlas-env/registry.go:67` is **read-only for this task** — D4 keeps it
storing unconditionally.

### Step 1: Write the failing tests

Append to `libs/atlas-env/tenants_test.go`. Setup is one line in every case —
`r := NewMapRegistry(Id("main"), time.Now)` — copied from the existing cases at
`libs/atlas-env/tenants_test.go:10-62`.

`TestReconcileTrustsTheHeaderForALegacyTenant`
: `r.ApplyTenant("t-1", Id(""))`; `Reconcile(r, Id("pr-1411"), "t-1")` returns
  `(Id("pr-1411"), nil)`. Failure message must name FR-3.1.

`TestReconcileStillRejectsTwoNonEmptyDisagreements`
: `r.ApplyTenant("t-1", Id("pr-123"))`; `Reconcile(r, Id("pr-1411"), "t-1")`
  returns an error satisfying `errors.Is(err, ErrEnvironmentMismatch)` and an
  empty `Id`. (FR-3.2 — the existing
  `TestReconcileRejectsADisagreement` covers header=`main`; this one covers two
  distinct non-baseline environments.)

`TestReconcileWithALegacyTenantAndNoHeaderIsTheLegacyValue`
: `r.ApplyTenant("t-1", Id(""))`; `Reconcile(r, Id(""), "t-1")` returns
  `(Id(""), nil)` — both empty is the legacy deployment, unchanged.

The four existing `Reconcile` tests must keep passing unmodified.

### Step 2: Implement

In `libs/atlas-env/tenants.go`, insert one arm immediately after the
`if !known { return headerEnv, nil }` block and immediately before
`if headerEnv == "" {`:

```go
	// A tenant projected with an EMPTY environment is LEGACY, not
	// "definitely belongs to no environment": a pre-#1427 tenant-status
	// event carried no environment attribute and MapRegistry.ApplyTenant
	// stores unconditionally (registry.go:67). Everywhere else in this
	// codebase "" means legacy-don't-filter (FR-1.8); treating it as a hard
	// mismatch here was the asymmetry that dropped every message a sparse
	// environment produced against a legacy tenant (FR-3.1).
	if tenantEnv == "" {
		return headerEnv, nil
	}
```

FR-3.2 then holds by construction: the `headerEnv != tenantEnv` arm is reached
only when both sides are non-empty.

### Verification

```
cd libs/atlas-env && go build ./... && go test ./...
```

---

## Task 3: Gate drop-reason distinguishability

Implements FR-3.3, D7, and the NFR "Observability" clause on
`gateSkipNotOwner`. `decide()` stays pure; **no verdict changes**.

### Files

- `libs/atlas-kafka/consumer/gate.go` — `gateReason`, second return value, `reason` label
- `libs/atlas-kafka/consumer/manager.go` — the one call site, `:631-641`
- `libs/atlas-kafka/consumer/gate_test.go` — reason assertions + FR-4.6 regression

Module root: `libs/atlas-kafka`.

### Step 1: Write the failing tests

Every existing test in `gate_test.go` calls `decide(...)` and compares one
value; all of them must be updated to `got, _ := decide(...)` — that is
mechanical and part of Step 2, not a new case. The new cases:

`TestGateDropReasons` — table-driven, one subtest per arm. Registry setup per
row is copied from the existing same-named scenarios in
`libs/atlas-kafka/consumer/gate_test.go:17-98`.

| subtest | registry / args | expect verdict | expect reason |
|---|---|---|---|
| `mismatched` | fresh registry, `msgEnv="pr-123"`, `mismatched=true` | `gateDropUnresolvable` | `reasonMismatched` |
| `stale` | registry with a fake clock advanced past staleness, `self="main"`, `msgEnv="pr-123"` | `gateDropUnresolvable` | `reasonStale` |
| `not active` | `main` ACTIVE only, `msgEnv="pr-999"` (no record) | `gateDropUnresolvable` | `reasonNotActive` |
| `not owner` | `main` + `pr-123` ACTIVE, `pr-123` overrides `atlas-character`, service `atlas-character`, `msgEnv="pr-123"` | `gateSkipNotOwner` | `reasonNotOwner` |
| `owner` | `main` + `pr-123` ACTIVE, service `atlas-monsters`, `msgEnv="pr-123"` | `gateProcess` | `reasonOwner` |
| `legacy` | fresh registry, `msgEnv=""` | `gateProcess` | `reasonLegacy` |

Precedence assertion — `mismatched` wins over every other arm:
`TestGateMismatchReasonWinsOverStaleness` builds the stale registry from the
`stale` row above and passes `mismatched=true`; expect
`(gateDropUnresolvable, reasonMismatched)`.

**FR-4.6 regression, by test not inspection** —
`TestGateWithNoRecordsProjectedIsUnchanged`: with
`r := env.NewMapRegistry(env.Id("main"), time.Now)` and **no** `r.Apply` call
at all, assert:

- `decide(r, env.Id("main"), "atlas-monsters", env.Id(""), false)` →
  `(gateProcess, reasonLegacy)` — a legacy deployment's own traffic.
- `decide(r, env.Id("main"), "atlas-monsters", env.Id("pr-999"), false)` →
  `(gateDropUnresolvable, reasonNotActive)` — identical to today's verdict.

Metric assertion — extend the existing
`TestGateDropUnresolvableIncrementsCounterAndSkipsHandler`
(`gate_test.go:130`) so its `testutil.ToFloat64` /
`testutil.CollectAndCount` read includes the `reason` label value the drop
carried, proving the label is populated rather than empty.

### Step 2: Implement

In `libs/atlas-kafka/consumer/gate.go`:

```go
// gateReason names WHICH arm of decide produced the verdict. Three drop
// arms previously emitted identical log text and shared one unlabelled
// counter; telling them apart cost several hours of task-243's diagnosis
// (FR-3.3).
type gateReason string

const (
	reasonMismatched gateReason = "mismatched"  // FR-7.7
	reasonStale      gateReason = "stale"       // registry staleness, design §4.3
	reasonNotActive  gateReason = "not_active"  // FR-4.7 / D4
	reasonNotOwner   gateReason = "not_owner"   // FR-4.4
	reasonOwner      gateReason = "owner"
	reasonLegacy     gateReason = "legacy"      // FR-1.8
)
```

- `decide` becomes
  `func decide(r env.Registry, self env.Id, service string, msgEnv env.Id, mismatched bool) (gateVerdict, gateReason)`.
  Every `return` gains its reason from the table above. **No condition,
  ordering, or verdict changes.**
- `gateDroppedUnresolvable`'s label slice becomes
  `[]string{"service", "environment", "reason"}`. Leave `gateProcessed` and
  `gateSkippedNotOwner` at two labels — only the drop counter needs the
  breakdown, and widening the others would break existing dashboards for no
  gain.

In `libs/atlas-kafka/consumer/manager.go:631`:

- `verdict, reason := decide(env.CurrentRegistry(), env.Self(), c.service, msgEnv, env.Mismatched(wctx))`,
  then `switch verdict {`.
- `gateDropUnresolvable` arm: pass `string(reason)` as the third label value,
  and add `.WithField("reason", string(reason))` to the existing `Error` log
  line. Change the message to name the arm rather than repeating one string
  for all three — e.g.
  `"Dropping message: environment is unresolvable. No deployment will process it."`
  stays, with `reason` now carried as a field.
- `gateSkipNotOwner` arm: it is currently silent. Add
  `l.WithField("environment", string(msgEnv)).WithField("topic", msg.Topic).WithField("reason", string(reason)).Debug("Skipping message: this deployment is not the environment owner.")`
  before the `return true`. Debug level, not Info — this fires on every
  non-owned message.

### Verification

```
cd libs/atlas-kafka && go build ./... && go test ./...
```

---

## Task 4: `servicesuniq` — dedupe and the unique index

Implements FR-2.1, D3, and §4.3's three-layer mitigation. Raw SQL after
`environmentcol`, never `AutoMigrate`.

### Files

- `services/atlas-configurations/atlas.com/configurations/servicesuniq/migration.go` — **new file**
- `services/atlas-configurations/atlas.com/configurations/servicesuniq/migration_test.go` — **new file**
- `services/atlas-configurations/atlas.com/configurations/main.go` — register the migration (see the list at `:48-51`)

Module root: `services/atlas-configurations/atlas.com/configurations`.

Read-only references:
- `services/atlas-configurations/atlas.com/configurations/environmentcol/migration.go` — the raw-`db.Exec` migration precedent to model on
- `services/atlas-configurations/atlas.com/configurations/environmentcol/migration_test.go` — the SQLite shadow-entity test precedent to copy wholesale
- `services/atlas-configurations/atlas.com/configurations/services/processor.go` (lines 29-54) — `serviceOutboxKey`, `enqueueServiceStatus`, `EnvServiceStatusTopic`
- `libs/atlas-outbox/outbox.go:16` — `Enqueue(tx *gorm.DB, msg Message) error`

### Step 1: Write the failing tests

`migration_test.go` — copy `testDatabase`, `testServiceEntity` and
`testServiceHistoryEntity` verbatim from
`environmentcol/migration_test.go:56-100` (drop the tenant/template shadows;
add an outbox shadow entity mirroring `outbox.Entity`'s columns so `Enqueue`
can insert). Each test seeds rows with `db.Exec("INSERT INTO services ...")`
exactly as that file does.

`TestDedupeKeepsTheDerivedIdRow`
: three `services` rows, all `type='login-service'`, `environment='pr-1411'`,
  ids `6439ca9c-d28d-5db9-821b-8dd93d318a25` (the derived id from Task 1),
  `11111111-1111-1111-1111-111111111111`, `22222222-2222-2222-2222-222222222222`.
  After `Migration(db)`: exactly one row for that group survives, and its id is
  `6439ca9c-d28d-5db9-821b-8dd93d318a25`.

`TestDedupeKeepsTheNewestWhenNoDerivedIdMatches`
: two rows, `type='drops-service'`, `environment='pr-1411'`, ids
  `11111111-…-1111` and `22222222-…-2222`, with `service_history` rows whose
  `created_at` is `2026-01-01T00:00:00Z` for the first and
  `2026-02-01T00:00:00Z` for the second. After `Migration(db)` the surviving id
  is `22222222-2222-2222-2222-222222222222`.

`TestDedupeFallsBackToTheLowestIdWithNoHistory`
: two rows, `type='world-service'`, `environment='pr-1411'`, ids
  `22222222-…-2222` and `11111111-…-1111`, no `service_history` rows. Survivor
  is `11111111-1111-1111-1111-111111111111`.

`TestDedupeEnqueuesATombstoneForEveryDeletedRow`
: `t.Setenv(services.EnvServiceStatusTopic, "test.svc.topic")` — **use the
  literal `"EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"` env-var name directly
  rather than importing the `services` package, to avoid an import cycle if one
  arises; confirm which by building.** Seed the three-row group from case 1.
  After `Migration(db)`, the outbox table holds exactly two rows, their
  `topic` is `test.svc.topic`, their `message_key` values are
  `service:11111111-1111-1111-1111-111111111111` and
  `service:22222222-2222-2222-2222-222222222222`, and both `message_value`
  values are NULL/empty (the compaction tombstone).

`TestDedupeLeavesOtherEnvironmentsAlone`
: one `login-service`/`main` row and one `login-service`/`pr-1411` row. Both
  survive — different groups, no duplicates.

`TestMigrationIsIdempotent`
: run `Migration(db)` twice on the three-row group; the second run returns nil,
  deletes nothing further, and enqueues no additional outbox rows.

`TestMigrationCreatesTheUniqueIndex`
: after `Migration(db)` on a clean two-group table, a second insert of
  `('login-service','pr-1411')` fails with a non-nil error.

`TestPreflightNamesDuplicateGroups`
: the read-only pre-flight (Layer 3) returns the three-row group as
  `{Type: "login-service", Environment: "pr-1411", Count: 3}` and does **not**
  delete anything — the row count is unchanged after the call.

### Step 2: Implement

`package servicesuniq`, with a package doc comment naming task-243 D3 and
stating that it must run **after** `services.Migration` and **after**
`environmentcol.Migration` (the latter backfills `environment`, so every row
carries a non-empty value by the time this runs).

Three exported symbols:

- `type DuplicateGroup struct { Type, Environment string; Count int }`
- `func Preflight(db *gorm.DB) ([]DuplicateGroup, error)` — Layer 3. Read-only.
  `SELECT type, environment, COUNT(*) FROM services GROUP BY type, environment HAVING COUNT(*) > 1`.
- `func Migration(db *gorm.DB) error` — Layer 2. In order:
  1. `Preflight`. If empty, skip straight to the index.
  2. For each group, select its rows' ids ordered so the keeper is
     deterministic. Keeper rule, applied in order: the row whose id equals
     `uuid5(ATLAS_SERVICE_NS, type+"/"+environment)`; else the row with the
     newest `service_history.created_at` for that `service_id`; else the
     lowest id lexicographically. The namespace constant is
     `c8f90111-a0cf-513e-95e6-c54609e5dec0` — declare it once as an unexported
     `uuid.UUID` package var with the same never-regenerate comment as
     `tools/derive-service-id.sh`, and cross-reference that file.
  3. Inside **one** `database.ExecuteTransaction`, for each loser: `DELETE FROM
     services WHERE id = ?` and `outbox.Enqueue(tx, outbox.Message{Topic: <the
     EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS env value>, Key: []byte("service:"+id.String()), Value: nil})`.
     When the topic env var is empty, skip the enqueue (mirroring
     `enqueueServiceStatus`'s own guard at `services/processor.go:37-40`) but
     still delete. Never a bare `DELETE` outside this transaction.
  4. If a group cannot be resolved unambiguously by the rule above, return a
     `fmt.Errorf` naming the `(type, environment)` pair and every candidate id,
     and do **not** create the index. A failed migration on a named row set is
     recoverable; a silent wrong-row deletion is not.
  5. `db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_services_type_env ON services (type, environment)")`.

Then in `main.go`, append `servicesuniq.Migration` to the `database.SetMigrations`
list **after** `environmentcol.Migration`, and update that line's trailing
comment (`// must run last: it backfills the columns the three above create`)
so it no longer claims `environmentcol` is last — say instead that
`servicesuniq` runs after it because it depends on the backfilled column.

### Verification

```
cd services/atlas-configurations/atlas.com/configurations && go build ./... && go test ./...
```

---

## Task 5: atlas-login readiness component

Implements FR-1.5 / D6, Go half. The manifest half is Task 7.

### Files

- `services/atlas-login/atlas.com/login/configuration/projection/state.go` — add `HasService`
- `services/atlas-login/atlas.com/login/configuration/projection/projection_test.go` — new cases
- `services/atlas-login/atlas.com/login/main.go` — a second `service.WithReadinessGate` (see `:69`)

Module root: `services/atlas-login/atlas.com/login`.

Read-only references:
- `libs/atlas-service/bootstrap.go:32` — `WithReadinessGate(fn func() bool) Option`
- `libs/atlas-service/bootstrap.go:102` — `Runtime.Ready` ANDs every gate
- `services/atlas-login/atlas.com/login/configuration/projection/subscriber.go:99` — the `"service:"+s.ServiceId.String()` key the projection already recognises

### Step 1: Write the failing tests

Append to `projection_test.go`; reuse whatever `State`/envelope construction
that file already does for `ApplyService` (it is the only setup needed).

`TestHasServiceIsFalseBeforeAnyServiceIsApplied`
: `NewState()`, `HasService()` is `false`.

`TestHasServiceIsTrueAfterTheMatchingServiceIsApplied`
: apply a `ServiceEnvelope` whose `Id` is
  `6439ca9c-d28d-5db9-821b-8dd93d318a25`; `HasService()` is `true`.

`TestHasServiceIsFalseAgainAfterATombstone`
: apply as above, then `ApplyServiceTombstone()`; `HasService()` is `false`.
  (A pod whose row is deleted stops serving and must go non-Ready — this is the
  same transition the apply loop already drains listeners on,
  `state.go:44-52`.)

`TestHasServiceIsFalseAfterOnlyATenantIsApplied`
: apply a tenant envelope only; `HasService()` is `false`.

The "different service's row" case from `design.md` §11 is already enforced one
layer up — `subscriber.handleService` returns early on
`env.Id != s.ServiceId.String()` (`subscriber.go:111-113`), so a foreign row
never reaches `State`. Add a comment in the test file pointing at that line
rather than re-testing it here.

### Step 2: Implement

In `state.go`, beside `Snapshot`:

```go
// HasService reports whether a service-config row for THIS deployment's
// SERVICE_ID has been projected. It is the FR-1.5 readiness signal: a pod
// whose row never arrives binds no socket, and before task-243 it reported
// Ready anyway — which is how a wrong SERVICE_ID survived five deploys.
// The subscriber filters foreign rows out before they reach State
// (subscriber.go:111), so a non-nil service is by construction our own.
func (s *State) HasService() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.service != nil
}
```

In `main.go`, immediately after the existing
`service.WithReadinessGate(caughtUp.CaughtUpNow),` (`:69`), add
`service.WithReadinessGate(state.HasService),` with a one-line comment naming
FR-1.5. `state` is already in scope at that point (it is captured by the
projection closure above it) — confirm before editing; if it is declared later,
move its declaration above the `service.Bootstrap` call rather than
restructuring the closure.

### Verification

```
cd services/atlas-login/atlas.com/login && go build ./... && go test ./...
```

---

## Task 6: atlas-channel readiness component

The same change as Task 5, in atlas-channel. Kept as a separate task because it
is a separate service and a separate review surface.

### Files

- `services/atlas-channel/atlas.com/channel/configuration/projection/state.go` — add `HasService`
- `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go` — new cases
- `services/atlas-channel/atlas.com/channel/main.go` — a second `service.WithReadinessGate` (see `:195`)

Module root: `services/atlas-channel/atlas.com/channel`.

Patterns to copy: whatever Task 5 landed — the two `State` types are
structurally identical (`state.go:16-20` in both trees). Read Task 5's diff
first and mirror it.

### Step 1: Write the failing tests

The same four cases and the same expected values as Task 5, with test names
unchanged, in atlas-channel's own `projection_test.go`. Use the derived
channel-service id `5a86d8e6-3167-5e74-9fc5-021d94001da2` as the envelope's
`Id` in the "true after apply" case.

### Step 2: Implement

`HasService` on `State` (same body, same doc comment with the pointer adjusted
to atlas-channel's `subscriber.go` line), and the extra
`service.WithReadinessGate(state.HasService),` in `main.go` after `:195`.

### Verification

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

---

## Task 7: readinessProbe on the login and channel Deployments

The manifest half of FR-1.5 / D6. Blast radius is **every overlay including the
baseline** — that is the intended semantic (a login server with no socket bound
is not serving), and it is why this is its own task with its own review.

### Files

- `deploy/k8s/base/atlas-login.yaml` — add `readinessProbe` to the `atlas-login` container
- `deploy/k8s/base/atlas-channel.yaml` — add `readinessProbe` to the `atlas-channel` container

Patterns to copy: `deploy/k8s/base/atlas-rankings.yaml:21-26` — the exact probe
shape, `/api/readyz` (the REST server sets base path `/api/` and mounts
`/readyz` under it, `atlas-login/main.go:176-179`), port 8080 (`REST_PORT` in
`deploy/k8s/base/env-configmap.yaml:194`).

Read-only: `tools/service-name-guard.sh`, `tools/overlay-env-guard.sh`,
`tools/sparse-baseline-scoping-guard.sh` — all three are gated on
`deploy/k8s/` changes and must still pass.

### Step 1: Edit

In each file, insert immediately after the container's `ports:` block and
before `envFrom:` (2-space list indentation under `containers: - name:`, i.e.
8 spaces for `readinessProbe:` — match the surrounding `envFrom:`/`env:` keys
exactly):

```yaml
        # FR-1.5 / task-243 D6: not Ready until this pod's OWN service-config
        # row has been projected (projection.State.HasService). atlas-login is
        # projection-driven — with no row it binds no socket, and before this
        # probe existed it reported Ready anyway, which is how a wrong
        # SERVICE_ID survived five sparse deploys undetected.
        readinessProbe:
          httpGet:
            path: /api/readyz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
          failureThreshold: 30
```

`failureThreshold: 30` with `periodSeconds: 10` gives a 5-minute catch-up
budget before the pod is marked unready — deliberately generous, because the
baseline's projection catch-up time has not been measured. `design.md` §8
requires these numbers be validated against the baseline before merge; record
the observed catch-up in `docs/runbooks/sparse-environments.md` (Task 13) and
tighten them there if the measurement supports it.

### Verification

```
kustomize build deploy/k8s/overlays/main >/dev/null
kustomize build deploy/k8s/overlays/pr >/dev/null
kustomize build deploy/k8s/overlays/pr-sparse >/dev/null
tools/service-name-guard.sh
tools/overlay-env-guard.sh
tools/sparse-baseline-scoping-guard.sh
tools/pr-sparse-mirror-guard.sh
```

All exit 0. Then confirm the only diff in the rendered `overlays/main` and
`overlays/pr` streams versus `origin/main` is the two probe blocks.

---

## Task 8: `service-config.sh` takes the id as a parameter

Implements FR-2.2's shell half and removes `new_uuid`. Depends on Task 1 only
for the derived values used in assertions.

### Files

- `services/atlas-pr-bootstrap/scripts/service-config.sh` — `build_service_config` gains an id parameter; `new_uuid` deleted
- `services/atlas-pr-bootstrap/test/service_config_test.bats` — retire the `new_uuid` cases, add the new contract

Module root: none (bash + bats). `tools/verify.sh:520-528` runs
`bats services/atlas-pr-bootstrap/test` on any change under
`services/atlas-pr-bootstrap/`.

### Step 1: Write the failing tests

In `service_config_test.bats`:

**Delete** the four `new_uuid:` cases (`:143`, `:149`, `:155`, `:170`) and the
two `build_service_config:` UUID-source cases (`:185`, `:194`). They test a
function that no longer exists. Delete the `_SC_UUID_PROC` seam references with
them.

**Rewrite** `"sparse mode never reads or writes the pinned main service row"`
(`:79`) to pass the id explicitly. Setup stays as it is (`ATLAS_MODE=sparse`,
`TENANT_ID`, `MAJOR_VERSION`, `LB_IP`, `ATLAS_ENVIRONMENT=pr-999`):

```
run build_service_config login "$CANONICAL/login-service.json" e7ae96a2-c484-5617-8e28-2178b60a8378
```

Assert: status 0; `.data.id` is exactly `e7ae96a2-c484-5617-8e28-2178b60a8378`
(not merely "not the pinned id"); `.data.attributes.tenants | length` is 1 and
`[0].id` is `$TENANT_ID`; `.data.attributes.environment` is `pr-999`.

**Add** `@test "build_service_config: sparse fails loudly when no id is supplied"`
: `ATLAS_MODE=sparse`, `ATLAS_ENVIRONMENT=pr-999`, call with only two
  arguments. Assert non-zero status and that `$output` contains
  `requires a service id`. This is the guard that replaces `new_uuid`'s: the
  CI rendering not having run must fail the hook, never mint an id here (D2).

**Add** `@test "build_service_config: sparse rejects a malformed id"`
: same setup, third argument `not-a-uuid`. Non-zero status, `$output` contains
  `requires a service id`.

**Keep unchanged** — these are the FR-1.4 no-change-outside-sparse assertions:
`"isolated mode still merges into the pinned row"` (`:103`) and
`"isolated mode POST body replaces channel-service's seeded placeholder tenant,
never appends beside it"` (`:111`). Both call `build_service_config` with two
arguments and must keep passing with the id parameter absent — isolated mode
never reads it.

### Step 2: Implement

In `services/atlas-pr-bootstrap/scripts/service-config.sh`:

- Delete `new_uuid` (`:34-51`) and its whole header comment block (`:17-33`).
  Replace the block with a short note that the id is now supplied by the
  caller from the CI-rendered `SERVICE_ID_<TYPE>`, and that deriving one here
  would resurrect the two-derivation-sites defect (D2) — pointing at
  `tools/derive-service-id.sh` as the single site.
- `build_service_config` becomes
  `build_service_config <shape> <template> [<service_id>]`. In the
  `ATLAS_MODE=sparse` arm, replace `id=$(new_uuid) || return 1` with a
  validation of `$3` against the same bash-native UUID regex `new_uuid` used
  (`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`);
  on failure `log error "build_service_config: sparse mode requires a service id (type=$shape, got '${3-}')"` and `return 1`.
- The isolated arm is byte-identical to today and ignores `$3`.

### Verification

```
bats services/atlas-pr-bootstrap/test
tools/shell-guard.sh --require-shellcheck
```

---

## Task 9: `bootstrap.sh` — upsert on the derived id, over the override set

Implements FR-1.1, FR-2.1, FR-2.3 and NFR "Failure visibility". Depends on
Task 8.

### Files

- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` — `create_service_config` rewritten; the three hard-coded calls become a loop; `kubectl set env` deleted
- `services/atlas-pr-bootstrap/test/bootstrap_test.bats` — new cases for the loop and the id-resolution helper

Read-only references:
- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` (lines 435-495) — `upsert_service_config`, whose GET/compare/PATCH shape the new sparse upsert reuses
- `services/atlas-pr-bootstrap/scripts/env-record.sh:16` — `env_record_get`, the source of `.data.attributes.overrides`
- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` (lines 42-60) — `env_header_init` / the `ENV_HEADER` array

### Step 1: Write the failing tests

`bootstrap_test.bats` currently sources `bootstrap.sh`'s helpers in isolation
(read its `setup()` first and follow whatever seam it uses — do not invent a new
one). The network-touching parts are out of scope for bats; test the two pure
pieces:

`@test "sparse service table maps every SERVICE_ID-carrying deployment"`
: For each of `atlas-login atlas-channel atlas-drops atlas-world
  atlas-character-factory atlas-drop-information`, the lookup helper returns
  the triple below. Assert all six, exactly:

| deployment | service type | shape | canonical template |
|---|---|---|---|
| `atlas-login` | `login-service` | `login` | `/atlas/canonical/services/login-service.json` |
| `atlas-channel` | `channel-service` | `channel` | `/atlas/canonical/services/channel-service.json` |
| `atlas-drops` | `drops-service` | `none` | `/atlas/canonical/services/drops-service.json` |
| `atlas-world` | `world-service` | `none` | `/atlas/canonical/services/world-service.json` |
| `atlas-character-factory` | `character-factory` | `none` | `/atlas/canonical/services/character-factory.json` |
| `atlas-drop-information` | `drops-information-service` | `none` | `/atlas/canonical/services/drops-information-service.json` |

**Before writing this table into code, list
`services/atlas-pr-bootstrap/canonical/services/` and confirm which templates
actually exist.** Only the three that exist today (`login-service.json`,
`channel-service.json`, `drops-service.json`) are proven; if the other three
have no canonical template, the table carries only the three that do and the
loop skips an override service with no template, logging it at info. Do not
fabricate a template file.

`@test "service id env var name is derived from the service type"`
: `svc_id_var_name login-service` → `SERVICE_ID_LOGIN_SERVICE`;
  `drops-information-service` → `SERVICE_ID_DROPS_INFORMATION_SERVICE`;
  `character-factory` → `SERVICE_ID_CHARACTER_FACTORY`. (Uppercase, `-` → `_`.)

`@test "sparse service-config step fails when the CI-rendered id is absent"`
: with the env var for a mapped type unset, the per-service function returns
  non-zero and logs `no SERVICE_ID_` — never mints an id, never proceeds.

### Step 2: Implement

Replace `create_service_config` (`:505-553`) with `upsert_sparse_service_config`:

```
upsert_sparse_service_config <deployment>
  resolve (type, shape, template) from the table; unmapped -> log info, return 0
  svc_id = ${!SERVICE_ID_<TYPE>}  ; empty -> log error "no SERVICE_ID_<TYPE> in
      environment; the CI rendering did not run" ; return 1   (D2, NFR failure visibility)
  GET  $ATLAS_UI_BASE/api/configurations/services/$svc_id
  present -> merge this environment's tenant entry into the LIVE attributes
             (merge_tenant_entry, or pass-through for shape=none), and PATCH
             only when the merged attributes differ from the live ones
             — the same jq -cS comparison as upsert_service_config:466-472,
             which also dodges the PATCH-handler panic on tenant-agnostic
             configs
  absent  -> body=$(build_service_config "$shape" "$tmpl" "$svc_id")
             POST with "${ENV_HEADER[@]}"  (ENVIRONMENT is server-owned and
             stamps the row's environment column; omitting it is what put
             every sparse row in the legacy '' environment)
  failure of either call -> log error, return 1
```

Replace the three hard-coded calls (`:566-568`) with a loop over the override
set, read from the environment record this bootstrap run already fetches:

```
overrides=$(env_record_get | jq -r '.data.attributes.overrides // {} | keys[]')
for d in $overrides; do upsert_sparse_service_config "$d" || exit 1; done
```

Creating a row only for a service actually deployed here is what removes the
orphan rows Task 4's unique index would otherwise have to tolerate, and it
removes the `|| log warn`-swallowed patch of Deployments that do not exist in a
sparse namespace.

Delete the `kubectl set env deployment/... SERVICE_ID=...` call and its
"SERVICE_ID routing (task-47)" comment block entirely. Replace with a short
note that the id now arrives in the manifest CI renders, so Argo's `selfHeal`
has nothing to revert (FR-1.2).

In the restart loop (`:585-600`): sparse mode no longer needs to restart
`atlas-login`/`atlas-channel` — their `SERVICE_ID` no longer changes after the
pod starts. Remove them from `restart_targets`' sparse branch and delete the
comment that explains why they were there. Leave the isolated-mode
`restart_targets` untouched.

**RBAC:** the Role's `patch`/`update` on `deployments`
(`deploy/k8s/overlays/pr-sparse/sync-bootstrap.yaml:15-17`) stays. The NFR
"Security" clause asks for an explicit re-examination, and this is its answer:
the restart loop still issues `kubectl rollout restart` for
`atlas-drops`/`atlas-world`/`atlas-character-factory` in both modes, which
needs `patch`. Record that conclusion in a comment on the Role's `verbs` line
so the next reader does not re-litigate it. **This file is mirror-locked
(`tools/pr-sparse-mirror-guard.sh` MIRRORS) — make the identical comment edit in
`deploy/k8s/overlays/pr/sync-bootstrap.yaml` in the same commit, or the guard
fails.**

### Verification

```
bats services/atlas-pr-bootstrap/test
tools/shell-guard.sh --require-shellcheck
tools/pr-sparse-mirror-guard.sh
```

---

## Task 10: `bootstrap.sh` — the activation step

Implements FR-4.1, FR-4.2, D5. Depends on Task 9.

### Files

- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` — a new `ATLAS_STEP=activate` block at the tail
- `services/atlas-pr-bootstrap/test/bootstrap_test.bats` — phase-decision cases

Read-only references:
- `services/atlas-pr-bootstrap/scripts/env-record.sh` — `env_record_get`, `env_record_patch`
- `services/atlas-pr-bootstrap/scripts/bootstrap.sh` (lines 114-126) — `record_environment_tenant`, the existing PATCH-with-current-phase pattern this one lifts the constraint from
- `services/atlas-pr-bootstrap/scripts/bootstrap.sh:676` — `ATLAS_STEP=done`, the insertion point (activation goes immediately before it)

### Step 1: Write the failing tests

A pure decision helper, `activation_decision <phase>`, echoing one of
`activate` / `skip` / `fail` so the branch is testable without a cluster:

| case | input phase | expect stdout | expect status |
|---|---|---|---|
| provisioning | `PROVISIONING` | `activate` | 0 |
| already active | `ACTIVE` | `skip` | 0 |
| deactivating | `DEACTIVATING` | `fail` | 0 |
| deleted | `DELETED` | `fail` | 0 |
| no record | `` (empty) | `fail` | 0 |

`@test "activation is sparse-only"`
: with `ATLAS_MODE=isolated`, the activation block is not entered. Assert via
  the same seam `bootstrap_test.bats` already uses for mode-gated behaviour;
  if none exists, assert that the guard expression in the script is
  `[ "${ATLAS_MODE:-isolated}" = "sparse" ]` by grepping the script — a weak
  test, and say so in a comment, rather than inventing a seam.

### Step 2: Implement

Immediately before `ATLAS_STEP=done` (`:676`), gated on
`[ "${ATLAS_MODE:-isolated}" = "sparse" ]`:

```
ATLAS_STEP=activate
for each deployment in the override set (same jq read as Task 9):
    kubectl rollout status deployment/"$d" --timeout="${ACTIVATION_ROLLOUT_TIMEOUT:-300s}"
        || { log error "activation: $d not ready"; exit 1; }     # FATAL, not `|| log warn`
assert atlas-login and atlas-channel are both in the override set and were
    rolled out above; otherwise log error and exit 1              # FR-4.2 mandatory sockets
body=$(env_record_get) || body=""
phase=$(printf '%s' "$body" | jq -r '.data.attributes.phase // empty')
case "$(activation_decision "$phase")" in
    skip)     log info "environment $ATLAS_ENVIRONMENT already ACTIVE"; ;;
    fail)     log error "environment $ATLAS_ENVIRONMENT is in phase '${phase:-<none>}'; refusing to activate"; exit 1 ;;
    activate) env_record_patch ACTIVE "$baseline" "$namespace" "$tenant" "$overrides" \
                  || { log error "activation PATCH failed"; exit 1; }
              log info "environment $ATLAS_ENVIRONMENT is ACTIVE" ;;
esac
```

`baseline` / `namespace` / `tenant` / `overrides` are read out of `$body` with
the same four `jq` reads `record_environment_tenant` uses (`:123-125`) —
`env_record_patch` zeroes any attribute omitted from the body.

Three properties to state in the block's header comment:

- **Observed, not assumed (FR-4.2).** Readiness comes from `rollout status`.
  Consumer-group initialization comes from the wave-0 precreate Job's own exit
  code, which Argo enforces as a precondition of wave 10 existing at all
  (FR-4.3/FR-4.4 are structural, not checked here).
- **Composes with FR-1.5.** With Task 7's readiness probe, a pod whose
  service-config row never arrived is never Ready, so `rollout status` times
  out and activation fails — the environment stays `PROVISIONING` rather than
  advertising a capability it does not have.
- **Idempotent.** `UpdateByName` accepts a same-phase transition and rejects
  skips and reverts, so a re-sync of an ACTIVE environment is a no-op and a
  re-sync during teardown cannot resurrect it.

### Verification

```
bats services/atlas-pr-bootstrap/test
tools/shell-guard.sh --require-shellcheck
```

---

## Task 11: sparse overlay — the two CI substitution tokens

Adds the render sites Task 12 fills. Nothing is generated here; this task only
places the tokens and the surrounding structure, and keeps
`kustomize build deploy/k8s/overlays/pr-sparse` valid with them unfilled.

### Files

- `deploy/k8s/overlays/pr-sparse/kustomization.yaml` — two `#PLACEHOLDER_` tokens in the `patches:` list

Read-only references:
- `deploy/k8s/overlays/pr-sparse/kustomization.yaml` (lines 85-93) — the `patches:` list head
- `deploy/k8s/overlays/pr-sparse/kustomization.yaml` (lines 168-188) — the existing inline JSON-6902 Job patch that adds `ATLAS_MODE`/`ATLAS_ENVIRONMENT` to `atlas-pr-bootstrap`; the precedent for "sync-bootstrap.yaml is mirror-locked, so overlay-only env lands here"
- `deploy/k8s/overlays/pr-sparse/kustomization.yaml` — the `#PLACEHOLDER_DELETE_BLOCK` line at the tail of `patches:`; the precedent for a comment-token that expands into list entries
- `deploy/k8s/base/atlas-kafka-precreate.yaml` (lines 41-53) — the Job whose container gets `KAFKA_CONSUMER_GROUP`

### Step 1: Edit

Immediately **before** the `#PLACEHOLDER_DELETE_BLOCK` line, add two token
lines, each with a comment block above it explaining what CI fills it with:

```yaml
  # PLACEHOLDER_SERVICE_ID_BLOCK — one inline strategic-merge patch per
  # override-set Deployment that carries a SERVICE_ID in base, setting that
  # Deployment's SERVICE_ID to uuid5(ATLAS_SERVICE_NS, "<type>/<env>") —
  # tools/derive-service-id.sh, the single derivation site (task-243 D1/D2).
  # THIS is what makes the binding survive Argo's selfHeal (FR-1.2): the value
  # is in the manifest Argo renders, so there is nothing to revert. The same
  # CI loop also emits a JSON-6902 patch adding SERVICE_ID_<TYPE> to the
  # atlas-pr-bootstrap Job, so bootstrap POSTs the id it rendered rather than
  # deriving one itself. Filled by .github/workflows/pr-validation.yml's
  # update-pr-overlay job; sparse mode only.
  #PLACEHOLDER_SERVICE_ID_BLOCK
  # PLACEHOLDER_PRECREATE_GROUPS_BLOCK — a JSON-6902 patch adding a
  # newline-delimited KAFKA_CONSUMER_GROUP to the wave-0
  # atlas-kafka-precreate Job, so seed_override_offsets /
  # verify_group_offsets stop taking their unset-guard early return
  # (kafka-precreate.sh:176, task-243 design §1.1 — the mechanism exists and
  # was one wire short). Patched onto the JOB, deliberately NOT added to the
  # atlas-env ConfigMap: every container in the namespace reads atlas-env via
  # envFrom, and a multi-line group list there would become the consumer
  # group of any container without a container-level override.
  #PLACEHOLDER_PRECREATE_GROUPS_BLOCK
```

Both are YAML comments when unfilled, so an unsubstituted
`kustomize build deploy/k8s/overlays/pr-sparse` is unaffected — which
`tools/overlay-env-guard.sh` and `tools/sparse-baseline-scoping-guard.sh` both
depend on.

Neither token may be added to `deploy/k8s/overlays/pr/kustomization.yaml`:
isolated mode has no override set, and CI substitutes only within
`$OVERLAY_DIR`, so a token there would survive to the workflow's
unfilled-`PLACEHOLDER_` sweep (`pr-validation.yml:1035-1043`) and fail every
isolated PR.

### Verification

```
kustomize build deploy/k8s/overlays/pr-sparse >/dev/null
kustomize build deploy/k8s/overlays/pr >/dev/null
tools/pr-sparse-mirror-guard.sh
tools/overlay-env-guard.sh
tools/sparse-baseline-scoping-guard.sh
```

Then confirm the two tokens exist **only** in the sparse overlay: list every
file under `deploy/k8s/overlays/` containing `PLACEHOLDER_SERVICE_ID_BLOCK` or
`PLACEHOLDER_PRECREATE_GROUPS_BLOCK` and check the result is exactly
`deploy/k8s/overlays/pr-sparse/kustomization.yaml`. A hit under
`deploy/k8s/overlays/pr/` would fail every isolated PR at the workflow's
unfilled-`PLACEHOLDER_` sweep.

---

## Task 12: CI renders the ids and the resolved group names

Implements FR-1.3, FR-4.3, design §3.2 and §6. Depends on Tasks 1 and 11.

### Files

- `.github/workflows/pr-validation.yml` — extend the `update-pr-overlay` job's sparse branch

Read-only references:
- `.github/workflows/pr-validation.yml` (lines 986-1028) — the existing
  `OVERRIDES_JSON` / `DELETE_BLOCK` / `NS_OVERRIDES` computation and the
  `\x01`-delimited `sed` pass this joins
- `.github/workflows/pr-validation.yml` (lines 1035-1043) — the unfilled-`PLACEHOLDER_` sweep
- `deploy/k8s/overlays/pr-sparse/patches/consumer-group-env.yaml` — the per-Deployment `KAFKA_CONSUMER_GROUP` values, already `PLACEHOLDER_ATLAS_ENV`-substituted by the time this block runs
- `libs/atlas-kafka/consumergroup/resolver.go` (lines 38-50) — `Resolve` `Sprintf`s the env value with the caller's args
- `services/atlas-login/atlas.com/login/main.go:49,55-56` and `services/atlas-channel/atlas.com/channel/main.go:175,181-182` — the two templated callers; **the `%s` is filled with `serviceId.String()`, not a channel number**

### Step 1: Edit

Inside the existing `if [ "$MODE" = "sparse" ]; then` block, after
`NS_OVERRIDES` is computed and **before** the `D=$(printf '\x01')` substitution
pass, add one loop over `$OVERRIDE_SERVICES` that produces both payloads from a
single pass:

```
SERVICE_ID_BLOCK=""
JOB_ENV_OPS=""
GROUPS=""
for svc in $OVERRIDE_SERVICES; do
  base="deploy/k8s/base/$svc.yaml"
  [ -f "$base" ] || continue
  # Only the six base Deployments that carry SERVICE_ID participate.
  stype=$(yq eval-all 'select(.kind=="Deployment") | .spec.template.spec.containers[0].env[]
            | select(.name=="SERVICE_TYPE") | .value' "$base" | head -1)
  [ -n "$stype" ] && [ "$stype" != "null" ] || continue
  cname=$(yq eval-all 'select(.kind=="Deployment") | .spec.template.spec.containers[0].name' "$base" | head -1)
  sid=$(tools/derive-service-id.sh "$stype" "pr-${PR_NUMBER}")
  # 1. the Deployment's SERVICE_ID (this is what survives selfHeal)
  SERVICE_ID_BLOCK="${SERVICE_ID_BLOCK}<one `- patch: |-` strategic-merge entry
      setting containers[name=$cname].env[SERVICE_ID].value to $sid on Deployment $svc>"
  # 2. the bootstrap Job's SERVICE_ID_<TYPE>
  var="SERVICE_ID_$(printf '%s' "$stype" | tr '[:lower:]-' '[:upper:]_')"
  JOB_ENV_OPS="${JOB_ENV_OPS}<one `- op: add` env/- entry: name=$var value=$sid>"
  # 3. the RESOLVED consumer group name for the precreate Job
  grp=$(yq eval-all 'select(.kind=="Deployment") | select(.metadata.name=="'"$svc"'")
          | .spec.template.spec.containers[0].env[] | select(.name=="KAFKA_CONSUMER_GROUP") | .value' \
        deploy/k8s/overlays/pr-sparse/patches/consumer-group-env.yaml | head -1)
  [ -n "$grp" ] && [ "$grp" != "null" ] || continue
  # The templated callers (atlas-login, atlas-channel) carry a literal "%s"
  # that Resolve() fills with serviceId.String() at runtime. Seeding the
  # template verbatim would create a group no consumer ever joins, and the
  # verification pass would report success having done nothing.
  grp=$(printf "$grp" "$sid")     # no-op when the value carries no %s
  GROUPS="${GROUPS}${grp}\n"
done
```

Then:

- If `SERVICE_ID_BLOCK` is empty, `::error::` and `exit 1` — a sparse
  environment with no `SERVICE_ID`-carrying override would bind nothing.
- If `GROUPS` is empty, `::error::` and `exit 1` — FR-4.4: activating without
  seeded offsets must not be possible, and an empty group list is exactly the
  "seeding reported success and did nothing" failure.
- Assert `GROUPS` contains no `%` character; `::error::` and `exit 1` if it
  does. An unresolved template is the single subtle failure mode of §6.
- Build `PRECREATE_GROUPS_BLOCK` as one JSON-6902 patch targeting
  `kind: Job, name: atlas-kafka-precreate`, adding
  `containers/0/env/-` with `name: KAFKA_CONSUMER_GROUP` and a block-scalar
  `value:` carrying `$GROUPS`.
- Build the bootstrap-Job patch from `JOB_ENV_OPS` as a second JSON-6902 entry
  targeting `kind: Job, name: atlas-pr-bootstrap`; append it to
  `SERVICE_ID_BLOCK` so one token carries both, matching the token's comment in
  Task 11.
- Add `-e "s${D}PLACEHOLDER_SERVICE_ID_BLOCK${D}${SERVICE_ID_BLOCK}${D}g"` and
  `-e "s${D}PLACEHOLDER_PRECREATE_GROUPS_BLOCK${D}${PRECREATE_GROUPS_BLOCK}${D}g"`
  to the existing `sed -i` invocation. Use the same `\x01` delimiter and the
  same `printf '\\\n  - ...'` newline-escaping idiom `DELETE_BLOCK` uses
  (`:993`) — the payloads contain `|` from YAML block scalars, so `|` is not a
  usable delimiter.
- Echo the derived ids and the resolved group names to the job log, one per
  line, so a wrong binding is diagnosable from the CI run alone.

### Step 2: Verify the render offline

Add a step, or extend the existing "Placeholders remaining" step, that runs
`kustomize build "$OVERLAY_DIR"` after substitution in sparse mode and asserts:

- every Deployment in the override set that has a `SERVICE_ID` has the derived
  value, not the base literal;
- the `atlas-pr-bootstrap` Job carries one `SERVICE_ID_<TYPE>` per such
  Deployment, with values matching;
- the `atlas-kafka-precreate` Job carries `KAFKA_CONSUMER_GROUP`, it is
  newline-delimited, and it contains no `%`.

This is the offline half of `design.md` §11's "Rendered-manifest" tests; the
`overlays/main` / `overlays/pr` byte-identity half is Task 7's verification.

### Verification

`.github/workflows/pr-validation.yml` has no local test harness. Verify by
extracting the new loop into a scratch shell script under the scratchpad,
running it against a fixed `OVERRIDE_SERVICES="atlas-login atlas-channel"` and
`PR_NUMBER=1411`, and confirming it emits exactly
`6439ca9c-d28d-5db9-821b-8dd93d318a25` for `login-service` and
`5a86d8e6-3167-5e74-9fc5-021d94001da2` for `channel-service`, and two group
lines carrying those uuids with no `%`. Paste that output into the task report.
Do not commit the scratch script.

---

## Task 13: runbook

### Files

- `docs/runbooks/sparse-environments.md` — new sections

Read-only references: `design.md` §5.4, §12; `prd.md` §2 non-goals.

### Step 1: Edit

Add, in the runbook's own voice and structure:

1. **Upgrade procedure (D9).** Every existing sparse environment must be torn
   down before this change is deployed; `cleanup.sh`'s environment-scoped
   reclaim goes through `ProcessorImpl.DeleteById`, which emits the tombstone a
   compacted topic requires. Recreation, not in-place reconciliation.
2. **The pre-flight (§4.3 Layer 3).** How to run `servicesuniq.Preflight`
   against the baseline database and read the duplicate set before rolling
   `atlas-configurations`. State plainly that if it names a group the dedupe
   rule cannot resolve — including a legitimate co-resident `login-service` row
   on `main` — the rollout stops and D3 is re-decided.
3. **The `ATLAS_ENVIRONMENT` roll precondition (§5.4).** A ConfigMap change does
   not roll the Deployments consuming it via `envFrom`, so baseline deployments
   may be running with `Self() == ""`. This is a **PRD non-goal and a separate
   defect**, but it is a precondition of the end-to-end test: roll the affected
   baseline deployments before running it. Do not describe the fix here; name
   the condition and how to check for it.
4. **How to verify a binding.** Read `SERVICE_ID` from the running override
   Deployment, run `tools/derive-service-id.sh <type> <env>`, and confirm they
   match; then confirm the `services` row with that id has `environment` equal
   to the environment's own.
5. **How to verify a seeded group.**
   `kafka-consumer-groups.sh --describe --group '<resolved name>'` shows a
   committed offset on every topic; a `-` is the FR-4.4 failure.
6. **The observed readiness numbers.** Record the baseline's measured
   projection catch-up time and whether Task 7's
   `initialDelaySeconds`/`failureThreshold` were tightened from the
   conservative defaults.
7. **The activation window (D8).** Accepted and bounded: during the flip a
   message is either dropped as not-yet-ACTIVE or processed by the baseline,
   never processed twice, and in practice the window is empty because the
   ingress does not route during `PROVISIONING`.

### Verification

```
tools/verify.sh
```

Flagless, exit 0. Confirm no literal home or absolute path appears in the new
prose (`docs/` is enforced).

---

## Post-plan gates

- Flagless `tools/verify.sh` exits 0.
- Code review before the PR is opened.
- The live acceptance criteria in `prd.md` §10 (three-run idempotency, Argo hard
  refresh, socket binding, automatic `ACTIVE`, and the end-to-end login) are
  executed against a freshly created sparse PR environment after the rollout
  ordering in `design.md` §12, and their results recorded in the task folder.
  They cannot be satisfied by `verify.sh` and must not be claimed from it.
