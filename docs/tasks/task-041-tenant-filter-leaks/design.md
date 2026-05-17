# Tenant Filter Leaks — Audit & Fix — Design

Version: v1
Status: Draft
Created: 2026-05-17

---

## 1. Premise check — existing infrastructure

The PRD characterizes the affected providers (atlas-guilds, atlas-character) as "leaks" because `WHERE tenant_id = ?` is not present in the function bodies, and explicitly notes a non-goal: *"Introducing a GORM global plugin or callback to auto-inject `tenant_id` (decided against — too broad a blast radius)."* During design exploration this premise was inverted:

**A tenant callback already exists and is wired up.** `libs/atlas-database/tenant_scope.go:43-49` registers four GORM `Before` callbacks:

```go
db.Callback().Query().Before("gorm:query").Register("tenant:query", tenantQueryCallback(l))
db.Callback().Row().Before("gorm:row").Register("tenant:row", tenantQueryCallback(l))
db.Callback().Create().Before("gorm:create").Register("tenant:create", tenantCreateCallback(l))
db.Callback().Update().Before("gorm:update").Register("tenant:update", tenantQueryCallback(l))
db.Callback().Delete().Before("gorm:delete").Register("tenant:delete", tenantQueryCallback(l))
```

`database.Connect` (`connection.go:123`) invokes `registerTenantCallbacks` on every service's DB at startup. The query callback (`tenant_scope.go:51-78`) checks:
1. `db.Statement.Schema` has a `tenant_id` column (via `hasTenantColumn`).
2. Context carries a tenant (`tenant.FromContext(ctx)`).
3. The skip flag (`WithoutTenantFilter`) is not set.

If all three hold, it appends `clause.Where{Eq{Column: tenant_id, Value: t.Id()}}` to the statement. That means `db.Where("id = ?", characterId).First(&entity{})` issued from a tenanted context is rewritten by GORM to `WHERE id = ? AND tenant_id = ?` *before the SQL is built*.

The audit on 2026-05-17 that listed the "leaks" was performed by reading provider source only; it didn't account for the callback layer. As a result, the PRD's primary fix prescription ("add `WHERE tenant_id = ?` to each provider") would produce *defense-in-depth duplicate filters* — which `TestDoubleWhereIsHarmless` (`tenant_scope_test.go:182-193`) already proves is safe — rather than closing a leak.

This does **not** mean the task is empty. The callback covers a large surface area but has real gaps and unknowns. The work re-shapes from *"add WHERE clauses everywhere"* to *"audit which call sites the callback does not cover, and close those gaps."*

## 2. Threat model (post-callback)

A tenant filter leak can still occur if any of the following is true:

| # | Failure mode | Why callback does not cover |
|---|---|---|
| F1 | Call does not use `WithContext(ctx)` | Without context, `tenant.FromContext` errors and callback returns silently (`tenant_scope.go:66-70`). |
| F2 | Context carries no tenant | Same as F1. Background goroutines that build their own context are at risk. |
| F3 | Entity lacks `tenant_id` column but should have one | `hasTenantColumn` returns false → callback skips. Either the column was forgotten, or the join target is global on purpose. |
| F4 | Raw SQL (`db.Exec`, `db.Raw`) | GORM Query/Row callbacks operate on parsed schema; raw SQL paths do not invoke schema-based callbacks. |
| F5 | Cross-table joins where the *driving* table has `tenant_id` but the joined table does not | The callback filters the driving table; the join can still surface rows from foreign tenants if the join key collides. |
| F6 | `Create` does not set `TenantId` on the struct | The create callback **only warns**; it does not inject the column. Row is inserted with the zero UUID, becoming invisible to all real tenants. |
| F7 | Update/Delete using `Where(&Entity{ID: x})` struct query | GORM struct-where ignores zero fields, so a struct with an empty `TenantId` does not constrain tenant; the callback adds it, but if the struct also sets a non-zero `TenantId` from a different source, the callback's eq + struct eq combine — usually fine, occasionally redundant. Worth confirming during audit. |
| F8 | Preload of an unrelated tenant-scoped table whose foreign key happens to collide across tenants | Preload issues a separate query; that query *does* invoke the Query callback, so it scopes correctly **only if** the preload target has its own `tenant_id` column. Tables related by foreign key but without their own `tenant_id` (e.g., child tables like `guild_members`, `guild_titles`) fall under F3. |
| F9 | Test-only DB connections that don't register the callback | `RegisterTenantCallbacks` is exported and used by tests, but a forgotten test setup would yield false-positive green tests that leak in prod. |
| F10 | `WithoutTenantFilter` used in too broad a scope | Background tasks (atlas-merchant, atlas-data, atlas-saga-orchestrator already use it) — risk is a tenant-aware call accidentally inheriting the bypass context. |

The PRD's listed call sites in atlas-guilds and atlas-character are mostly *not* leaks under F1-F10 — they use `WithContext(p.ctx)` from a tenanted processor, query tables that have `tenant_id`, and don't do raw SQL. **Exception: F8** — `Preload("Members")` and `Preload("Titles")` on the `guilds` table fan out to `guild_members` and `guild_titles` (`guild/title/entity.go`, `guild/member/entity.go`). Whether those child tables have `tenant_id` columns determines whether they're actually leak-free; the audit must confirm.

## 3. Architectural options

### Option A — Trust the callback; close gaps only

Reframe Phase 1 (audit) and Phase 2 (fix) around F1–F10. Do not add per-query `WHERE tenant_id = ?` clauses where the callback already handles them. Where the audit surfaces a gap, fix it specifically:
- F1/F2: add `WithContext(p.ctx)` or thread tenanted context into goroutines.
- F3/F8: add `tenant_id` columns + backfill migration to child tables that should be tenant-scoped.
- F4: rewrite raw SQL to GORM API, or add explicit `tenant_id` predicate.
- F6: harden `tenantCreateCallback` to *inject* `TenantId` from context (not just warn) — and remove redundant manual assignments at call sites in a follow-up.
- F9: add a lint or a test that asserts callbacks are registered before any test connection is returned.
- F10: scope-check every existing `WithoutTenantFilter` call site and document why it's safe.

**Pros:** Minimal blast radius. Centralised invariant. Future entities inherit protection for free. Defense matches what the codebase already does.
**Cons:** Doesn't satisfy the PRD's literal text ("Fix each identified leak by adding `tenant_id = ?`"). Requires re-baselining acceptance criteria.

### Option B — Belt and suspenders (callback + per-query)

Keep the callback as-is, but also add explicit `WHERE tenant_id = ?` to every read/write provider for tenant-scoped entities. Treat the callback as a backstop and the per-query filter as the primary contract.

**Pros:** Satisfies PRD as-written. Each provider is self-evidently safe when read in isolation. New engineers don't have to know about the callback layer.
**Cons:** Many low-value changes (the callback already enforces them). Doesn't close F4/F6/F8/F9 — those gaps remain. Adds parameter plumbing (`tenantId uuid.UUID`) to dozens of providers and their processors. The hidden gaps stay hidden while the cosmetic work absorbs the budget.

### Option C — Hybrid: minimal per-query reinforcement + gap-closing

Add an explicit `tenant_id = ?` predicate only to providers that:
1. Issue raw SQL (F4).
2. Are invoked from a background goroutine whose context discipline cannot be guaranteed (F1/F2 in housekeeping tasks).
3. Drive a join/preload whose target table is missing `tenant_id` and adding the column is impractical.

Everything else: do the F-class fixes from Option A. Tests cover both per-query and callback paths.

**Pros:** Partial PRD compliance, full real-leak closure, modest churn.
**Cons:** Mixed mental model (some providers filter explicitly, others rely on callback). Harder to write a single "rule" engineers can follow.

### Decision

**Option A** — locked. The callback is the existing, tested invariant; this task treats it as the baseline and closes the F1–F10 gaps it cannot reach.

Operating rule for the audit and all subsequent work: **tenant scoping is a per-provider decision, not a universal mandate.** Most providers default to tenant-scoped (callback covers them automatically). A subset is intentionally cross-tenant and stays that way using `WithoutTenantFilter`, with a justification comment and a verified bypass boundary. The audit captures which side every call site falls on.

PRD acceptance criteria were adapted accordingly: a tenant-scoped call site is acceptable when it is either (a) covered by the callback (PASS-CB, verified by audit), (b) has an explicit predicate where the callback cannot reach (PASS-EXPLICIT, e.g., raw SQL), or (c) is intentionally cross-tenant (PASS-CROSS-TENANT, justified). Option B (defense-in-depth duplicate WHERE clauses) was rejected — it does not close F4/F6/F8/F9 and adds churn without correctness gain.

## 4. Audit methodology

Phase 1 produces `audit.md` next to this design. The audit must enumerate every call site that touches a tenant-scoped entity and classify it as:

- **PASS-CB** — covered by the callback (uses `WithContext` from a tenanted processor, entity has `tenant_id`, no raw SQL, no F-class gap).
- **PASS-EXPLICIT** — has its own `tenant_id` predicate (legitimately, e.g., in `WithoutTenantFilter` scope or background recovery).
- **PASS-CROSS-TENANT** — intentionally cross-tenant; cite the comment and the bypass call.
- **LEAK-F<n>** — fails a specific F-class check. Must be fixed.
- **UNCLEAR** — needs reviewer judgment.

Audit pipeline (executed in plan phase, not now):

1. **Static enumeration** — `rg --type go -e 'p?\.db\.\b' -e 'tx\.\b' -e 'db\.Where' -e 'db\.Find' -e 'db\.First' -e 'db\.Create' -e 'db\.Save' -e 'db\.Updates' -e 'db\.Delete' -e 'db\.Exec' -e 'db\.Raw' services/atlas-* | rg -v _test`. Pipe into a CSV with `service,file:line,call,context`.
2. **Entity inventory** — for each `entity.go`, record table name and whether `TenantId` is declared. Cross-reference against the 13 entities currently lacking it (already enumerated in research). For each missing-column entity, decide: (a) genuinely global, (b) should have `tenant_id` and column add is in scope, (c) child of a tenant-scoped parent (relies on join discipline — see F3/F8).
3. **WithContext audit** — for each call from step 1, verify `WithContext` is present on the chain. Flag bare `p.db.<verb>(...)` calls.
4. **Background goroutine audit** — `rg -e 'go ' -e 'time.Ticker' -e 'task\.' services/atlas-* --type go` to find tasks; verify each goroutine's `ctx` chain originates either from `tenant.WithContext(...)` (the per-tenant pattern in `atlas-guilds/.../task.go:43`) or from `WithoutTenantFilter` with explicit justification.
5. **Raw SQL audit** — Exec/Raw matches from step 1 get individual review.
6. **Preload audit** — every `Preload(...)` chain is matched against the target child table's entity. If the child lacks `tenant_id`, F3/F8.
7. **WithoutTenantFilter audit** — confirm each existing call site has a comment and that the bypass scope ends before any tenant-sensitive operation downstream.

Output: `audit.md` table with columns `service, file:line, function, op (R/W), classification, fix_required, notes`. PRD §4.1 already mandates this format; we extend the classification column.

## 5. Fix strategy (post-audit)

Per gap class:

- **F1 (no WithContext)**: thread context. If the call is inside `database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error { ... })`, the inner `tx` already inherits context — verify; otherwise pass `tx.WithContext(p.ctx)`. (`tx` from `ExecuteTransaction` is a child of the parent DB and inherits its statement context.)
- **F2 (no tenant in context)**: convert ad-hoc background contexts to `tenant.WithContext(parent, t)` where `t` is enumerated from the registry. Existing pattern: `atlas-guilds/guild/task.go:43` — `tctx := tenant.WithContext(sctx, g.Tenant())`.
- **F3/F8 (missing column on child table)**: prefer adding `tenant_id` column + index + backfill via a one-shot Migration function next to the entity. Backfill source: parent table's `tenant_id` joined by the existing FK. If column add is rejected (e.g., huge table), accept F8 as long as the audit documents that the access path always joins through a parent-with-tenant_id.
- **F4 (raw SQL)**: prefer rewriting to GORM API. If unavoidable, parameterize tenant and pass it explicitly.
- **F6 (Create without TenantId)**: harden `tenantCreateCallback` to inject `tenant_id` on the reflected struct/slice when (a) context carries a tenant, (b) skip flag is unset, (c) current value is zero UUID. Existing test `TestCreateDoesNotInjectWhere` covers the no-WHERE path; add `TestCreateInjectsTenantIdWhenMissing` for the new behavior. Remove redundant TenantId assignments at call sites in a follow-up sweep — *out of scope for this task* (mechanical, low-risk, can wait).
- **F9 (test setup forgot to register callback)**: add an assertion helper in `libs/atlas-database` (e.g., `MustHaveTenantCallbacks(db) panic if not registered`) and call it in shared test scaffolds. Adopt opportunistically; not a blocker.
- **F10 (over-broad WithoutTenantFilter)**: each call site gets a comment justifying the bypass and a verified scope boundary. Tests cover that the bypass does not bleed into downstream calls.

## 6. Test strategy

PRD §4.3 was amended during design: the project uses the existing in-memory sqlite pattern from `libs/atlas-database/tenant_scope_test.go` rather than introducing testcontainers. Rationale:
- The leak under test is "does the WHERE clause filter rows by tenant_id" — sqlite reproduces this faithfully.
- CLAUDE.md's verification loop runs plain `go test -race ./...`; keeping it fast (no Docker) matters for the inner loop.
- No testcontainers tests exist anywhere in the repo today; adopting them is a separate decision.
- Postgres-specific behavior (uuid type semantics, `LOWER(name)` collation, RETURNING) is not under test here. Revisit if those become load-bearing.

Approach:

1. **Reuse existing pattern** — `libs/atlas-database/tenant_scope_test.go` already shows the canonical setup: `gorm.Open(sqlite.Open(":memory:"), ...)`, `RegisterTenantCallbacks`, then `AutoMigrate` test entities. Provider tests in services follow the same shape, swapping in the service's real `Migration` and `tenant.WithContext` to switch tenants.
2. **Shared helper (optional)** — if duplication becomes painful across services, lift the boilerplate into a small helper under `libs/atlas-database` (e.g., `NewInMemoryTenantDB(t *testing.T, migrations ...Migrator) *gorm.DB`). Do this only when at least three services would benefit; it is not a prerequisite.
3. **Provider-level tests** — for each fixed call site, a `provider_test.go` adjacent to it:
   - Seed two tenants with overlapping primary keys / unique-by-tenant columns (e.g., same character name in both tenants, same guild id in both).
   - Run the provider with tenant A's context.
   - Assert tenant A's row only.
   - Assert mutation calls don't touch tenant B's rows.
4. **Per-service smoke coverage** — for every service that touches a tenant-scoped entity, add at least one read and one write provider test (PRD §10 strict). Same shape as #3.
5. **F6 regression test** — extend `tenant_scope_test.go` with a `TestCreateInjectsTenantIdWhenMissing` case: Create an entity whose `TenantId` is the zero UUID under a tenant context; assert the persisted row has `tenant_id = t.Id()`. Also add a counterpart `TestCreateDoesNotOverrideExplicitTenantId` covering the not-zero path.
6. **Fixture construction** — use the project's Builder pattern (CLAUDE.md Test Helper Pattern). Do not create `*_testhelpers.go` files with test-only constructors.

## 7. Scope decisions

**In scope (this PR):**
- Audit pipeline + `audit.md`.
- All F1/F2/F4/F6 fixes in atlas-guilds, atlas-character, and any other service surfaced by the audit.
- F3/F8 column additions for child tables found leaky (single-PR idempotent migrations).
- Hardened `tenantCreateCallback` (inject `tenant_id` when missing) bundled in this PR.
- Optional shared sqlite test helper under `libs/atlas-database` if at least three services would benefit; otherwise services use the existing `tenant_scope_test.go` pattern in place.
- Provider regression tests per PRD §10 strict: every fix + one read + one write per audited service.

**Out of scope (separate task or explicitly deferred):**
- Removing redundant `TenantId` assignments at Create call sites once the callback injects them (mechanical follow-up).
- Testcontainers Postgres integration (OQ-4: sqlite is sufficient for the leak being tested).
- A CI lint to enforce `WithContext` on every `p.db.*` call (PRD §2 explicitly defers).
- Refactoring `EntityProvider`/`database.Query` abstractions.
- Changes to atlas-tenants, atlas-data (read-only WZ), atlas-ui (no DB).
- Backfill of `tenant_id` on entities whose tables genuinely never had it (decide in audit; default = leave alone with a justification comment).

## 8. Resolution of PRD Open Questions

- **§9 cross-tenant queries** — already mapped. atlas-merchant (3 sites), atlas-data (5 sites), atlas-saga-orchestrator (2 sites). Audit will verify each site's bypass scope; no new cross-tenant API needed unless a new gap is found.
- **§9 atlas-asset-expiration / atlas-object-id allocator** — Postgres side is covered by the callback assuming entities have `tenant_id` and calls use `WithContext`. Redis side is not in scope (PRD §2). The audit verifies the Postgres side; allocator coordination is not changed.
- **§9 testcontainers helper** — confirmed not to exist. Decision: not introducing one. The project's existing in-memory sqlite pattern (`libs/atlas-database/tenant_scope_test.go`) covers the WHERE-clause filtering being tested. See §6.

## 9. Service inventory implication

The PRD lists ~31 likely-affected services. With the callback already in place, the audit will likely produce:
- **Most services:** PASS-CB across the board after a mechanical context check. No code changes.
- **A handful (atlas-merchant, atlas-data, atlas-saga-orchestrator, atlas-guilds child tables, possibly atlas-parties, atlas-messengers):** F-class gaps requiring targeted fixes.
- **Test coverage burden:** PRD §10 (strict, OQ-5) mandates regression tests per fixed call site *plus* one read and one write provider per service that touches a tenant-scoped entity. ~30 services × 2 thin sqlite-backed tests ≈ 60 small test files. Accepted as the cost of thorough verification; the test scaffolding per service is small (~40 lines) because it reuses the `tenant_scope_test.go` pattern.

## 10. Migration / rollout notes

- Single PR per PRD §8. Run on PR overlay with two tenants in fixtures; tests fail if any audit-marked site regresses.
- F6 hardening (Create callback injecting tenant_id) is the most behavior-changing piece. It needs to land *with* the audit doc so reviewers can verify intent. The change is opt-in by entity (only entities with a `tenant_id` column are affected; entities without one are untouched).
- No DB schema changes from this design — except F3/F8 column additions where audit deems necessary. Those each get their own `Migration` function and idempotent backfill.

## 11. Resolved Decisions

All five open questions were resolved during design before plan.md.

- **OQ-1 — Option A.** Trust the existing tenant callback as the invariant. The audit closes F1–F10 gaps it cannot reach. No defense-in-depth duplicate WHERE clauses. Tenant scoping is a per-provider decision: most providers default to scoped (callback covers); a justified subset is intentionally cross-tenant via `WithoutTenantFilter`.
- **OQ-2 — Bundled.** F6 hardening (`tenantCreateCallback`: warn → inject) lands in the same PR as the audit and its consumers. Single atomic change.
- **OQ-3 — Single-PR migration.** F3/F8 child-table column additions use one idempotent `Migration` function per entity (AutoMigrate adds column, in-place backfill, index, NOT NULL where safe). No two-phase deploy. Table sizes in scope are small (guild members, titles, etc.).
- **OQ-4 — Sqlite, defer testcontainers.** Use the existing in-memory sqlite pattern from `libs/atlas-database/tenant_scope_test.go`. CI continues to run plain `go test -race`. No new lane, no Docker dependency in the verification loop. Testcontainers can be revisited later if Postgres-specific behavior becomes load-bearing.
- **OQ-5 — Strict per-PRD §10.** Regression test per fix, plus one read + one write provider per service that touches a tenant-scoped entity. ~30 services × 2 thin tests is acceptable budget against the sqlite harness.
