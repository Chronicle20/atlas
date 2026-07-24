# Backend Audit — atlas-data (baseline restore resilience)

- **Service Path:** services/atlas-data
- **Commit:** 86eb0b27f "fix(atlas-data): make baseline restore resilient to cancellation"
- **Guidelines Source:** backend-dev-guidelines skill (ai-guidance.md, file-responsibilities.md, patterns-resilience.md)
- **Date:** 2026-07-17
- **Build:** PASS (`go build ./...` in services/atlas-data/atlas.com/data)
- **Tests:** `go test ./baseline/... -count=1 -race` → PASS (ok, 1.043s)
- **Scope:** limited to the 6 files touched by 86eb0b27f, per audit request. Findings that cite unchanged files (handler.go, deploy/k8s/base/atlas-data.yaml) are cited only as evidence for a hazard introduced by the changed files' new call pattern (`Reconcile`), not as independent violations of unchanged code.
- **Overall:** NEEDS-WORK — no build/test/guideline-checklist failure, but one Important correctness/safety finding (unguarded concurrent restore) that the task explicitly asked to check for.

## Correctness / Safety Findings

### 1. [Important] `Reconcile` has no mutual-exclusion against concurrent `Restore()` for the same tenant — and the deployment topology guarantees the race fires on exactly the recovery path this fix targets

Evidence:
- `services/atlas-data/atlas.com/data/baseline/reconcile.go:57-73` — `Reconcile` iterates every `StatusRestoring` row and spawns one `routine.Go`-wrapped `r.Restore(ctx, ...)` per tenant, unconditionally, with no lock, lease, advisory lock, or staleness/heartbeat check to confirm the restore is actually dead rather than merely in progress.
- `services/atlas-data/atlas.com/data/baseline/restore.go:106-114,227-234` — neither `cleanupAfterFailure` nor `markRestoring` take or check any per-tenant lock; `markRestoring`'s `ON CONFLICT (tenant_id) DO UPDATE` is idempotent for a single writer but provides no isolation between two concurrent writers.
- `services/atlas-data/atlas.com/data/main.go:146` — `baseline.Reconcile(rt.Context(), l, db, mc)` runs unconditionally at every process start, before the REST routes are even mounted.
- `deploy/k8s/base/atlas-data.yaml:39-41` — `replicas: 4`, `strategy: Recreate`. `Recreate` tears down all 4 old pods and starts all 4 new pods together, so **every** pod boot runs `Reconcile` concurrently against the same Postgres.
- `services/atlas-data/atlas.com/data/baseline/handler.go:107-129` (unchanged, pre-existing) — `restoreInner` still lets an operator POST `/data/baseline/restore` for an arbitrary tenant at any time, synchronously calling the same unguarded `Restorer.Restore`.

Consequence: the exact scenario this commit is designed to heal — a pod killed mid-`COPY`, leaving a tenant `StatusRestoring` — is the scenario in which the next rolling restart (`strategy: Recreate`, 4 replicas) starts all 4 pods at once. All 4 independently see the same pending row and each spawns its own `Restore()` for that tenant: 4 concurrent `DELETE FROM <table> WHERE tenant_id=?` + `COPY ... FROM STDIN BINARY` transactions racing on the same 6 tables (`restore.go:82-91`), and 4 concurrent finalize UPSERTs into `tenant_baselines` (`restore.go:212-219`) whose last writer wins regardless of which restore's table data actually landed intact. If an operator also manually re-triggers the same restore (a very plausible action right after observing an interrupted one), a 5th concurrent run joins the race. The result can be a `tenant_baselines` row asserting `StatusComplete`/a given sha while the underlying tables contain an interleaved mix of two dump runs — a *new* silent-corruption failure mode, distinct from (and arguably worse than) the atlas-pr-933 bug this commit fixes, because it now has a durable, self-triggering cause (every crash-and-restart cycle) instead of a one-off proxy timeout.

Not a build/test failure — `go test ./baseline/... -race` passes because no test exercises two concurrent `Restore()` calls for the same tenant.

**Recommendation scope note (not prescribing the fix, just what's missing):** nothing in the diff establishes single-flight/leader semantics for `Reconcile`, nor a "restore already in progress" guard in `restoreInner`. A `pg_advisory_lock(hashtext(tenant_id))`-style guard, a `SELECT ... FOR UPDATE SKIP LOCKED` claim on the `tenant_baselines` row, or gating `Reconcile` to a single replica would close this gap.

### 2. [Minor] `routine.Go`-spawned reconcile restores are not tracked by any WaitGroup — a shutdown mid-heal can re-produce the exact half-restore this commit fixes, on the next boot

Evidence:
- `libs/atlas-routine/routine.go:15-26` — `Go` only recovers panics; it does not register with any `sync.WaitGroup` or block `rt.Wait()`.
- `services/atlas-data/atlas.com/data/baseline/reconcile.go:65-72` — the reconcile-triggered `Restore()` runs fully detached from the process lifecycle once spawned.
- `services/atlas-data/atlas.com/data/main.go:146,198` — `rt.Wait()` (line 198) has no dependency on the goroutines `Reconcile` (line 146) spawned.

Consequence: if the pod receives SIGTERM/SIGKILL while a reconcile-triggered restore is mid-`COPY`, the restore is interrupted again, leaving the tenant `StatusRestoring` — which is fine in isolation (the design is meant to be eventually convergent: the *next* `Reconcile` picks it up again), but combined with finding #1, repeated restarts under sustained pressure (e.g., a crash-looping pod) can produce a steady stream of overlapping restore attempts rather than converging. Flagged as Minor because the design is self-healing by intent and this is a lower-probability compounding factor on top of #1, not a standalone correctness bug.

## Domain / File-Responsibilities Checklist

`baseline` is a support package (no `model.go`; uses `Restorer`/`Reconcile` + `handler.go` instead of the `Processor`/`resource.go` DOM pattern — that structure predates this diff and is unchanged by it). Checks scoped to the 6 changed files:

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Goroutines spawned via `routine.Go`, no bare `go` | PASS | `reconcile.go:65` uses `routine.Go(l, ctx, ...)`; `grep -rnE '^\s*go (func\|[A-Za-z_])' baseline/` returns zero matches |
| DOM-27 | Transient DB errors → 503, not bare 500 | N/A for these files | `restore.go`/`reconcile.go` return raw `error`; HTTP mapping happens in unchanged `handler.go`, which already routes through `server.WriteErrorResponse` (handler.go:47,95,123) and `main.go:134-140` already registers `RegisterTransientErrorClassifier` — pre-existing, untouched by this diff |
| DOM-28 | No silent degradation in decorators/enrichment | N/A | No `model.Decorator` in these files |
| FILE-06 | No package-named catch-all file mixing responsibilities | PASS | `restore.go` (restore logic), `reconcile.go` (reconcile logic), `migration.go` (entity+migration, pre-existing placement, unchanged by this diff's structure) each hold one responsibility; no new symbol was misplaced by this diff |
| — | SQL built from validated table names only (no injection) | PASS | `restore.go:64` gates every table name against the fixed `DumpTables` allowlist (`dump.go:20-27`) before any string-concatenated `DELETE`/`COPY`/`ANALYZE`; `cleanupAfterFailure` (restore.go:109) and the `ANALYZE` loop (restore.go:204) iterate `DumpTables` directly, never externally-supplied strings |
| — | `ON CONFLICT` UPSERT correctness | PASS | Both `markRestoring` (restore.go:227-234) and the finalize UPSERT (restore.go:212-219) use `ON CONFLICT (tenant_id) DO UPDATE SET ... = EXCLUDED....`, valid Postgres syntax exercised by the migration-added `status` column; `markRestoring` deliberately omits `restored_at` from its `DO UPDATE SET` list (preserves last-*completion* time rather than last-attempt time) — correct given `restore.go:212-219` is what stamps `restored_at=now()` on actual completion |
| — | Migration backfill safety on existing rows | PASS (with a Minor redundancy note) | `migration.go:24` (`Status string \`gorm:"not null;...;default:'complete'"\``) — verified against `gorm.io/gorm@v1.31.2/migrator/migrator.go:91-109` (`FullDataTypeOf`) that `AutoMigrate` emits a single `ALTER TABLE ... ADD status ... NOT NULL DEFAULT 'complete'` statement, which Postgres backfills atomically for pre-existing rows before enforcing `NOT NULL` — no risk of the migration failing on a populated table. The explicit follow-up `UPDATE tenant_baselines SET status = ? WHERE status IS NULL OR status = ''` (migration.go:36) is therefore dead code / a no-op given the `DEFAULT` clause already backfilled every row — harmless, but the comment above it ("Pre-existing rows are backfilled to StatusComplete by the migration") implies this `UPDATE` does the backfill when in fact the `AutoMigrate` DDL already did. Minor, non-blocking. |
| — | 30m/2m timeout sanity | PASS | `restoreOpTimeout = 30 * time.Minute` / `cleanupTimeout = 2 * time.Minute` (restore.go:38-41) — documented rationale (documents ~50k rows, shared-Postgres contention) is consistent with the restore's actual work (6-table COPY + ANALYZE); no evidence of undersizing |
| — | `context.WithoutCancel` + timeout correctly detaches from caller | PASS | `restore.go:128-130` — `opCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), restoreOpTimeout)`; preserves tenant/trace values per `context.WithoutCancel` semantics while dropping the parent's cancellation, matching the stated intent |
| — | `cleanupAfterFailure` uses `context.Background()`, not the restore ctx | PASS | `restore.go:107` — `ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)`, confirmed independent of the (possibly-cancelled) `Restore` ctx passed in from the caller |
| DOM-20 | Table-driven tests | PARTIAL | `reconcile_test.go` covers only `pendingRestores` (the SQL query) with a table of rows (lines 25-32) — genuine table-driven test, PASS for that unit. `Reconcile` itself (goroutine spawn, bad-UUID handling at reconcile.go:58-62, `mc == nil` skip at reconcile.go:44-47) has zero test coverage — no assertion exists that `Reconcile` actually calls `Restore` for a pending row, skips on bad UUID, or no-ops on nil `mc`. Minor/non-blocking (the source-structure tests in `restore_failure_test.go` are the established pattern for the COPY-path per task instruction and are not flagged), but distinct from that carve-out since this is a gap in `Reconcile`'s own behavior, not the binary-COPY path. |

## Summary

### Blocking (must fix)
- None — build and tests pass, no DOM checklist item fails outright.

### Important (should fix before/soon after merge)
- **Finding #1**: `Reconcile` (reconcile.go:57-73) + 4-replica `Recreate`-strategy deployment (deploy/k8s/base/atlas-data.yaml:39-41) + no per-tenant lock anywhere in `Restore`/`markRestoring`/`cleanupAfterFailure` (restore.go) → guaranteed concurrent-restore race on every rolling restart that follows an interrupted restore, capable of producing a `tenant_baselines` row that lies about which dump's data actually landed.

### Non-Blocking (should fix)
- **Finding #2**: reconcile-spawned restores aren't tracked by any WaitGroup (reconcile.go:65-72, libs/atlas-routine/routine.go:15-26) — a shutdown mid-heal re-creates the interruption this commit fixes, on the next boot. Self-correcting by design but compounds with Finding #1 under repeated restarts.
- `migration.go:36`'s explicit `UPDATE ... WHERE status IS NULL OR status = ''` is dead code given GORM's `ADD COLUMN ... NOT NULL DEFAULT 'complete'` already backfills existing rows; comment implies otherwise.
- `Reconcile`'s own logic (goroutine spawn, bad-UUID skip, nil-`mc` skip) has no test coverage; only the `pendingRestores` query is tested.
