# Task 19 — `EXPLAIN ANALYZE` evidence for the poller and gameplay indexes

PRD §14 asks that the composite index shape be confirmed against the real
query plan rather than assumed. This file records the verbatim
`EXPLAIN ANALYZE` output captured against a real Postgres for the three
queries the indexes added in this task exist to serve.

## How this was produced (reproducible)

- Image: `postgres:16-alpine`, started via `testcontainers-go/modules/postgres`
  (`tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.BasicWaitStrategies())`),
  the same pattern Task 18's `poller_integration_test.go` uses.
- Schema: `scheduling.MigrateTable`, `definition.MigrateTable`, and
  `occurrence.MigrateTable` run unmodified against the container — i.e. the
  same code path a real deployment runs, including the seven indexes added by
  this task.
- Seed data (one tenant, region `GMS` major 83 minor 1):
  - `scheduled_event_work`: 5,000 rows in state `COMPLETED` with `execute_at`
    randomly spread across the last ~7 days (the accumulated backlog
    `ix_sew_pending_due` must stay independent of), plus 200 rows in state
    `PENDING` with `execute_at` in the past (the poller's actual due set).
  - `event_occurrence`: 5,000 rows in state `COMPLETED` for the same
    tenant/world(1)/channel(4)/type(`ANNIVERSARY`), plus one row in state
    `ACTIVE` — the row the FR-API7 and FR-B15 queries target.
  - `event_occurrence_map`: 500 noise rows with scattered `map_id`s (half
    `visual = true`, half `visual = false`), plus one row
    (`map_id = 200090010, visual = true`) tying to the `ACTIVE` occurrence
    above.
  - `ANALYZE` run after seeding, before capturing any plan.
- Harness: a throwaway `//go:build explaintool`-tagged test
  (`event/scheduling/zz_explain_temp_test.go`, run via
  `go test -tags explaintool ./event/scheduling/... -run TestCaptureExplainPlans -v`)
  that seeds the data above and prints each `EXPLAIN ANALYZE` result row via
  `db.Raw(query).Rows()`. The file is **not** committed — it existed only to
  produce this evidence and was deleted before the Task 19 commit. To
  reproduce, re-create it from this description (or from the Task 19 diff in
  git history if retained) and re-run the command above; Docker must be
  available (`docker info` succeeds).
- Run: `go test -tags explaintool ./event/scheduling/... -run TestCaptureExplainPlans -v`
  — PASS, 11.69s total including container start/stop.

## FR-S4 poller hot path (`ix_sew_pending_due`)

```sql
EXPLAIN ANALYZE
SELECT * FROM scheduled_event_work
WHERE state = 'PENDING' AND execute_at <= now()
ORDER BY execute_at ASC LIMIT 50
```

```
Limit  (cost=0.15..105.33 rows=50 width=136) (actual time=0.013..0.027 rows=50 loops=1)
  ->  Index Scan using ix_sew_pending_due on scheduled_event_work  (cost=0.15..420.90 rows=200 width=136) (actual time=0.012..0.024 rows=50 loops=1)
        Index Cond: (execute_at <= now())
Planning Time: 0.579 ms
Execution Time: 0.049 ms
```

Index-served (`Index Scan using ix_sew_pending_due`). No sequential scan on
`scheduled_event_work`, and the plan never touches the 5,000-row `COMPLETED`
backlog — the partial `WHERE state = 'PENDING'` predicate excludes it from
the index entirely, which is the FR-N16 property this index exists for.

## FR-B15 map-entry query (`ix_occ_map` + `ix_occ_active_scope`)

```sql
EXPLAIN ANALYZE
SELECT o.* FROM event_occurrence o
JOIN event_occurrence_map m ON m.occurrence_id = o.id
WHERE m.map_id = 200090010 AND m.visual
  AND o.tenant_id = '51e12a4a-db3c-4bd5-81d9-61eb9eedfb95' AND o.world_id = 1 AND o.channel_id = 4
  AND o.state = 'ACTIVE'
```

```
Nested Loop  (cost=0.40..16.45 rows=1 width=121) (actual time=0.051..0.052 rows=1 loops=1)
  Join Filter: (m.occurrence_id = o.id)
  ->  Index Scan using ix_occ_active_scope on event_occurrence o  (cost=0.12..8.15 rows=1 width=121) (actual time=0.023..0.024 rows=1 loops=1)
        Index Cond: ((tenant_id = '51e12a4a-db3c-4bd5-81d9-61eb9eedfb95'::uuid) AND (world_id = 1) AND (channel_id = 4))
  ->  Index Only Scan using ix_occ_map on event_occurrence_map m  (cost=0.27..8.29 rows=1 width=16) (actual time=0.025..0.025 rows=1 loops=1)
        Index Cond: (map_id = 200090010)
        Heap Fetches: 1
Planning Time: 0.895 ms
Execution Time: 0.075 ms
```

Both sides of the join are index-served: `event_occurrence` via
`ix_occ_active_scope` (tenant/world/channel, partial on `state = 'ACTIVE'`)
and `event_occurrence_map` via `ix_occ_map` (leading `map_id`, partial on
`visual = true`) as an Index Only Scan. No sequential scan on either table.

## FR-API7 second query (`ix_occ_type_state` / `ix_occ_active_scope`)

```sql
EXPLAIN ANALYZE
SELECT * FROM event_occurrence
WHERE tenant_id = '51e12a4a-db3c-4bd5-81d9-61eb9eedfb95' AND type = 'ANNIVERSARY' AND state = 'ACTIVE'
```

```
Index Scan using ix_occ_active_scope on event_occurrence  (cost=0.12..8.14 rows=1 width=121) (actual time=0.011..0.012 rows=1 loops=1)
  Index Cond: (tenant_id = '51e12a4a-db3c-4bd5-81d9-61eb9eedfb95'::uuid)
  Filter: (type = 'ANNIVERSARY'::text)
Planning Time: 0.058 ms
Execution Time: 0.026 ms
```

Index-served. No sequential scan on `event_occurrence`. The planner chose the
partial `ix_occ_active_scope` (tenant/world/channel/state) over the
non-partial `ix_occ_type_state` (tenant/type/state) for this row set — both
indexes cover the query with an index-only tenant lookup plus a residual
`type` filter, and Postgres correctly judged `ix_occ_active_scope` cheaper
given `state = 'ACTIVE'` is in both predicates and the seeded data has only
one `ACTIVE` row for this tenant/world/channel. `ix_occ_type_state` remains
useful for tenants/queries that filter by `type` without a world/channel/
ACTIVE scope (e.g. a query across all states for a type), which
`ix_occ_active_scope` cannot serve since it is partial on `state = 'ACTIVE'`.

## Summary

All three plans are index-served — no `Seq Scan` appears on
`scheduled_event_work`, `event_occurrence`, or `event_occurrence_map` in any
of the three plans above.
