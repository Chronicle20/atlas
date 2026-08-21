# Review — Task 4: `servicesuniq` dedupe migration and unique index

Range reviewed: `2072f881e..4f5742892` (single commit `4f5742892`).

Scope: `services/atlas-configurations/atlas.com/configurations/servicesuniq/migration.go` (new),
`servicesuniq/migration_test.go` (new), `main.go` (migration registration),
`tools/derive-service-id.sh` (comment cross-reference only). Read-only reference files consulted:
`environmentcol/migration.go`, `environmentcol/migration_test.go`,
`services/processor.go:1-53`, `libs/atlas-outbox/outbox.go`, `libs/atlas-outbox/entity.go`,
`libs/atlas-database/transaction.go`.

## Requirement-by-requirement

1. **Idempotency (not reversibility — accepted per context.md).** `TestMigrationIsIdempotent`
   (migration_test.go:224-247) runs `Migration(db)` twice on a three-row group and asserts no
   further row/outbox change on the second run. Traced through the code: the second run's
   `Preflight` returns no groups (the first run already collapsed the group to one row), so
   `Migration` falls straight to `CREATE UNIQUE INDEX IF NOT EXISTS …` (migration.go:120), which
   is itself idempotent. PASS.

2. **Namespace constant appears in exactly two places, each cross-referencing the other.**
   - `servicesuniq/migration.go:29-36` — `atlasServiceNS`, comment names
     `tools/derive-service-id.sh` as the reciprocal site.
   - `tools/derive-service-id.sh:26-29` (diff) — comment updated from "appears here and NOWHERE
     else" to name `atlasServiceNS` in `servicesuniq/migration.go` as the reciprocal site.
   Both directions verified present in the tree (`git diff 2072f881e..4f5742892 --
   tools/derive-service-id.sh` shows only the comment changed, value untouched). PASS.

3. **Follows the `environmentcol` precedent shape.** `servicesuniq/migration.go` uses raw
   `db.Exec`/`db.Raw` throughout, no `AutoMigrate` of production structs — matches
   `environmentcol/migration.go`. `migration_test.go`'s `testServiceEntity`/
   `testServiceHistoryEntity` (lines 22-38) are verbatim copies of
   `environmentcol/migration_test.go:53-70`; `testDatabase` (lines 61-82) is the same
   in-memory-SQLite/`SetMaxOpenConns(1)`/`t.Cleanup` shape. The added `testOutboxEntity`
   (migration_test.go:42-53) mirrors every column `outbox.Entity` (libs/atlas-outbox/entity.go)
   writes, including `enqueued_at`, `sent_at`, `attempts`, `last_error` — necessary because
   `outboxlib.Enqueue` does a `tx.Create(&ent)` against the full entity, not just the columns the
   tests read back. PASS.

4. **Outbox consistency on deletion.** Every loser is deleted and (topic permitting) tombstoned
   inside the same `database.ExecuteTransaction(db, …)` call (migration.go:93-113): `DELETE FROM
   services WHERE id = ?` followed immediately by `outboxlib.Enqueue(tx, outboxlib.Message{Topic:
   topic, Key: []byte("service:"+loser.String()), Value: nil})`, both using `tx`. When
   `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` is empty the enqueue is skipped but the delete still
   runs (migration.go:99-101) — mirrors `enqueueServiceStatus`'s guard at
   `services/processor.go:37-40`. Key prefix `"service:"` matches `serviceOutboxKey` at
   `services/processor.go:29-31`. `outboxlib.Enqueue` requires non-empty `Key` (outbox.go:23-24) —
   always satisfied since `loser` is a real UUID. `TestDedupeEnqueuesATombstoneForEveryDeletedRow`
   (migration_test.go:172-211) asserts exactly 2 tombstone rows with the two loser keys, `Value`
   empty. PASS.

   One structural note (non-blocking): the brief's step 3 reads "Inside **one**
   `database.ExecuteTransaction`, for each loser: …" directly under a step-2 sentence that begins
   "For each group, select its rows' ids…". A literal reading could mean *one transaction covering
   every loser across every duplicate group*, rather than one transaction per group (which is what
   the code does — `database.ExecuteTransaction` is called inside the `for _, g := range groups`
   loop, migration.go:81-114). The delete+enqueue pair for a given loser is always atomic either
   way, which is the invariant the brief's closing sentence ("Never a bare DELETE outside this
   transaction") actually protects, and per-group transactions do not weaken idempotency or
   correctness — if migration fails on a later group, earlier groups' dedup already committed
   validly. Not blocking, but worth flagging since the wording is genuinely ambiguous.

5. **Migration registered in `main.go`.** `servicesuniq.Migration` appended to
   `database.SetMigrations` at `main.go:52` after `environmentcol.Migration`; import added at
   `main.go:8`; the trailing comment on `environmentcol.Migration` no longer claims "must run
   last," and `servicesuniq.Migration` now carries that claim with the correct reason (depends on
   environmentcol's backfilled `environment` column). Matches the brief's exact required text.
   PASS.

6. **Keeper rule, in order: derived id → newest history → lowest id → error.**
   `resolveGroup` (migration.go:187-232) implements exactly this order.
   - Derived-id branch (192-197): `uuid.NewSHA1(atlasServiceNS, []byte(g.Type+"/"+g.Environment))`
     — matches `uuid5(ATLAS_SERVICE_NS, "<type>/<environment>")` used by
     `tools/derive-service-id.sh`. `TestDedupeKeepsTheDerivedIdRow` confirms the pinned value
     `6439ca9c-d28d-5db9-821b-8dd93d318a25` for `login-service/pr-1411` is picked over two other
     candidate ids.
   - Newest-history branch (199-214): only entered when no row's id matches the derived id; ties
     (`.Equal(newest)`) are collected into `newestRows` and only resolve the branch if exactly one
     row is the unique newest, otherwise falls through — correct tie handling.
   - Lowest-id branch (216-227): string-lexicographic compare; ties again correctly fall through
     instead of picking an arbitrary one.
   - Unresolvable case (229-237): returns `fmt.Errorf` naming type, environment, and every
     candidate id; migration returns early and skips index creation (the caller propagates the
     error out of `Migration` before reaching `db.Exec("CREATE UNIQUE INDEX …")` at
     migration.go:118-121). PASS — matches the brief's Step 2.4 exactly, including the "do not
     create the index" requirement.

7. **Unique index.** `CREATE UNIQUE INDEX IF NOT EXISTS idx_services_type_env ON services (type,
   environment)` (migration.go:120), only reached once every duplicate group resolved without
   error. `TestMigrationCreatesTheUniqueIndex` confirms a subsequent duplicate insert fails.
   `IF NOT EXISTS` makes re-running post-dedup free of side effects, keeping the whole migration
   idempotent even when there were zero duplicate groups to begin with (`Preflight` empty → skip
   straight to the index, per Step 2.1). PASS.

8. **`Preflight` (Layer 3), read-only.** `SELECT type, environment, COUNT(*) AS count FROM
   services GROUP BY type, environment HAVING COUNT(*) > 1` (migration.go:52-54), no mutation.
   `TestPreflightNamesDuplicateGroups` asserts the exact `{Type, Environment, Count}` shape from
   the brief and that row count is unchanged after the call. PASS.

## The implementer's `newestHistoryFor` concern — adjudicated

Claim: SQLite's `MAX(created_at)` aggregate returns the value as a driver string rather than a
native `time.Time` in this query shape, requiring a manual `interface{}` scan-and-normalize
(migration.go:154-176).

- **(a) Correctness for every driver this migration will actually meet.** This module's
  Postgres driver stack is `gorm.io/driver/postgres` → `jackc/pgx/v5` (go.mod:63,81); `lib/pq` is
  present only as an indirect dependency of another package, not the one `libs/atlas-database`
  opens (verified `postgres.Open`/`Dialector` usage in `libs/atlas-database`). pgx's stdlib driver
  maps `timestamptz`/`timestamp` columns — including through a `MAX()` aggregate, which does not
  change the column's declared type — to `time.Time` as the `driver.Value`. The `case time.Time:`
  branch (migration.go:161) handles this directly with no string round-trip, so production
  behavior does not go through the string-parsing path the implementer added for SQLite. The
  `string`/`[]byte` branches (162-166) exist purely for the SQLite test harness. This is correct
  as written: it does not silently coerce a `time.Time` through string formatting/reparsing, which
  is the failure mode that would risk precision or timezone loss.
- **(b) Wrong-keeper risk.** Because the `time.Time` case returns the driver value unmodified and
  compares it with `.After`/`.Equal` (native `time.Time` methods, migration.go:203-209), there is
  no string-comparison path in the code that could produce a different ordering than a native time
  comparison for the production (Postgres) case. The string-parsing branches are only reachable
  under SQLite, where the test suite exercises and passes both the newest-history and tie-breaking
  branches. No wrong-keeper risk identified for either driver actually in play.
- **(c) Is the SQLite workaround masking a query shape that should have been written
  differently?** Partially yes, as a style note, non-blocking: `SELECT id FROM service_history
  WHERE service_id = ? ORDER BY created_at DESC LIMIT 1` would let the caller `Scan` into a
  concrete `time.Time`/`uuid.UUID` pair directly and avoid the `interface{}` type-switch
  entirely, on both drivers, since ordering-and-limiting a single column is portable SQL that
  doesn't hit the aggregate-return-type quirk at all. The chosen shape (`MAX(created_at)`) is not
  incorrect — it is verified correct on both drivers actually met — but the type-switch is
  more code than a `SELECT … ORDER BY … LIMIT 1` would have required. Not a defect; a
  simplification opportunity outside this review's mandate to demand.

**Verdict on this concern: not a blocking defect.** The normalization is correct for the only two
drivers this code will ever run against (verified above, not merely asserted), does not touch the
production comparison path with the string-parsing logic, and the passing SQLite test suite
exercises the identical logical branches (tie handling, no-history fallback) that Postgres will
take through the native-`time.Time` path.

## Backend guidelines

- Immutable models: `DuplicateGroup` and `candidateRow` are plain value structs, no mutation
  outside construction. Consistent with guideline.
- Functional composition: `Preflight` → `candidateRowsFor` → `resolveGroup` is a clean pipeline;
  `Migration` is the only place with side effects (delete/enqueue/index), matching the intended
  separation between Layer 2 (mutating) and Layer 3 (read-only).
- `libs/atlas-constants/` checked (grep for the namespace UUID and the topic env-var literal): no
  existing equivalent found, consistent with the implementer's note that this is a migration-local
  pinned value, not a general domain constant.
- Builder pattern / no `*_testhelpers.go`: test setup lives entirely in `migration_test.go` itself
  (`testDatabase`, shadow entities), no separate testhelpers file introduced. Consistent.
- Line endings / repo-relative paths: N/A, no cross-cutting sweep in this diff.

## Build/test verification (targeted, not a full re-run)

```
cd services/atlas-configurations/atlas.com/configurations
go build ./...      # clean
go test ./servicesuniq/... -v   # 8/8 PASS
gofmt -l servicesuniq/ main.go  # no output
go vet ./...         # no output
```
Confirms the implementer's report. Not a substitute for `tools/verify.sh`, which was not re-run
here per the task instructions.

## Not evaluable

- Real Postgres behavior of `newestHistoryFor`'s `time.Time` branch was assessed by tracing the
  driver stack (pgx v5 via `gorm.io/driver/postgres`) and pgx's documented type mapping, not by
  running this migration against a live Postgres instance with actual duplicate `service_history`
  rows. No live Postgres environment was available in this review's scope. This is a reasoned
  inference from the driver contract, not an empirical confirmation.
- Whether any production `services` table currently holds a duplicate group whose id equals
  neither the derived id nor resolves unambiguously by history/lowest-id (i.e., whether the
  Step-2.4 error path will actually fire against real data) is outside this review's surface —
  that is an operational/rollout concern for whoever runs this migration, not a code defect.

## Findings summary

No blocking findings. One non-blocking note: the brief's "one `database.ExecuteTransaction`"
wording in Step 3 is ambiguous between per-group and whole-migration transaction scope; the
implementation chose per-group, which satisfies every invariant the brief actually protects
(atomic delete+enqueue, no bare DELETE) but a stricter reading could disagree. Flagging for
awareness, not requesting a change.
