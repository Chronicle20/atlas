# DB Transaction Coverage for Multi-Entity Mutations — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

`database.ExecuteTransaction` (libs/atlas-database/transaction.go:9) is the project-standard way to make a multi-statement database mutation atomic. It has 53 call sites across 18 services — but 14 of the 32 database-backed services have **zero** call sites, and several of them mutate multiple tables (or perform multi-statement read-modify-write sequences) inside a single handler. A failure partway through such a flow leaves the database partially applied: for example, atlas-monster-book upserts a `card` row and then updates the `collection` book level as two independent writes — a crash between them permanently desynchronizes card count from book level. Combined with the dual-write problem (CD-2), an un-transacted partial write can also diverge from what was emitted to Kafka.

This task is backlog item **DL-4** (docs/architectural-improvements.md:152, severity Medium). It has two parts, delivered together: (1) a systematic **audit** of all write paths in the 14 services, producing a committed per-service verdict document, and (2) **remediation** — wrapping every flow that genuinely needs atomicity in `ExecuteTransaction`, with a rollback test per wrapped flow. Services whose write paths are all single-statement (or whose semantics intentionally tolerate partial application, e.g. saga compensation) get an explicit "no change needed" verdict rather than a mechanical wrap.

Notably, some of these services already carry the transaction plumbing unused: monster-book's `card.ProcessorImpl` and `collection.ProcessorImpl` both expose `WithTransaction(tx)` (services/atlas-monster-book/atlas.com/monster-book/card/processor.go:45), so remediation there is wiring an `ExecuteTransaction` around existing composable pieces, not a redesign.

## 2. Goals

Primary goals:
- Every multi-statement mutation flow in the 14 audited services is either wrapped in `ExecuteTransaction` or documented with an explicit justification for why atomicity is not required.
- A committed audit artifact records the verdict for every write path in every audited service — including the "no multi-statement flows" services — so DL-4 can be closed with evidence, not assertion.
- No new CD-2 instances: Kafka emits stay **outside** every transaction boundary introduced by this task.
- Each wrapped flow has a test proving that a mid-flow failure rolls back all prior writes in that flow.

Non-goals:
- Transactional-outbox adoption (CD-2 / task-114-outbox-adoption — lands before this task; touched flows follow whatever emit-after-commit convention it establishes).
- Wrapping single-statement CRUD in transactions (nothing to make atomic; GORM single statements are already atomic).
- Touching the 18 services that already use `ExecuteTransaction`.
- DL-5 manual tenant-filter cleanup (separate backlog item).
- Cross-service atomicity (that is saga territory, out of scope by design).

## 3. User Stories

- As an operator, I want a mid-flow crash or DB error in any service to leave the database in the pre-flow state, so that no manual data repair is needed after incidents.
- As a player, I want game-state pairs that must move together (e.g. monster card count and book level) to never desynchronize, so that my progression is never silently corrupted.
- As a developer, I want a committed audit document stating which write paths were examined and why each was or wasn't wrapped, so that future changes to those services know the atomicity contract.
- As a reviewer, I want a rollback test per wrapped flow, so that the transaction boundary is verified behavior, not decoration.

## 4. Functional Requirements

### FR-1. Audit phase

- FR-1.1 — For each of the 14 services (list in §7), enumerate every code path that issues a database write (Create/Save/Update/Updates/Delete/Exec), grouped by the handler/processor entry point that triggers it.
- FR-1.2 — Classify each entry point as one of:
  - **A. Multi-table** — writes to two or more tables in one logical operation.
  - **B. Multi-statement single-table** — multiple writes, or a read-modify-write sequence, where a mid-flow failure (or a concurrent interleaving exploiting the RMW gap) leaves inconsistent state. In scope per the confirmed definition of "multi-entity".
  - **C. Single-statement** — one atomic GORM statement; no change needed.
  - **D. Intentionally non-atomic** — partial application is by design (e.g. saga-orchestrator state persistence with compensation semantics); requires a written justification.
- FR-1.3 — The audit must be a **full sweep** of write paths per service, not a spot-check (per project verification rules). Cite `file:line` for every classified entry point.
- FR-1.4 — Verdicts land in `docs/tasks/task-119-db-transaction-coverage/audit.md`, one section per service, committed on the task branch.

### FR-2. Remediation phase

- FR-2.1 — Every class-A and class-B flow is wrapped in `database.ExecuteTransaction`, threading the `tx` through the existing `WithTransaction`/administrator-function patterns of that service. Follow the established Processor pattern: pure `Method(mb)` logic composed inside the transaction, side-effecting emit after commit.
- FR-2.2 — Kafka emits (`message.Emit`, outbox enqueue, or whatever convention task-114 lands) must occur **after** the transaction commits, never inside `fn`. Exception: if task-114's outbox convention requires enqueue-in-tx, follow that convention explicitly.
- FR-2.3 — `ExecuteTransaction` is re-entrant (it detects an existing transaction and joins it), so nested processor composition is safe; remediation must not introduce manual `db.Begin()`/`Commit()` calls.
- FR-2.4 — No behavior changes beyond atomicity: same writes, same order, same emitted events, same REST responses.
- FR-2.5 — Class-D flows (saga-orchestrator expected here) receive no code change; the audit justification is the deliverable.

### FR-3. Testing

- FR-3.1 — Each wrapped flow gets a test that injects a failure after the first write and asserts the earlier write(s) rolled back (database state unchanged). Use the project's Builder pattern for setup; no `*_testhelpers.go` constructors.
- FR-3.2 — Existing tests in touched services stay green (`go test -race ./...` per changed module).

## 5. API Surface

No new or modified REST endpoints, request/response shapes, or Kafka message contracts. This task changes only the internal write-path composition of the affected services. Any observable API change is a regression against FR-2.4.

## 6. Data Model

No schema changes, no new entities, no migrations. Transaction boundaries only.

## 7. Service Impact

The 14 database-backed services with zero `ExecuteTransaction` call sites (verified by grep, 2026-07-02):

| Service | Expected outcome |
|---|---|
| atlas-monster-book | Known multi-table flow: card upsert + collection level update. Wrap. |
| atlas-account | Audit; wrap any multi-statement flows found. |
| atlas-ban | Audit; wrap any multi-statement flows found. |
| atlas-families | Audit; wrap any multi-statement flows found. |
| atlas-keys | Audit; likely batch key-binding writes (class B candidates). |
| atlas-map-actions | Audit; wrap any multi-statement flows found. |
| atlas-maps | Audit; wrap any multi-statement flows found. |
| atlas-marriages | Audit; wrap any multi-statement flows found. |
| atlas-npc-conversations | Audit; wrap any multi-statement flows found. |
| atlas-party-quests | Audit; wrap any multi-statement flows found. |
| atlas-portal-actions | Audit; wrap any multi-statement flows found. |
| atlas-reactor-actions | Audit; wrap any multi-statement flows found. |
| atlas-storage | Audit; wrap any multi-statement flows found. |
| atlas-saga-orchestrator | Included in audit; "no change" (class D) is the expected legitimate verdict given saga compensation semantics. |

Per-service expected outcomes other than monster-book and saga-orchestrator are deliberately unprejudged — the audit determines them.

### Sequencing dependencies

- **task-114-outbox-adoption (CD-2) lands first.** This task rebases on it and follows its emit convention for every touched flow.
- **task-116-processor-gen3-unification lands before implementation.** It may rewrite the same processor files; the audit phase can proceed in parallel, but remediation commits happen only after task-116 is merged, against the unified processor generation.

## 8. Non-Functional Requirements

- **Multi-tenancy** — transactions must not bypass the GORM tenant-scope callbacks; `ExecuteTransaction`'s `tx` inherits the session context, and wrapped code must keep passing the tenant-bearing context as before.
- **Performance** — wrapping 2–5 statements in a transaction adds negligible overhead; no flow in scope is hot-path packet handling. If the audit finds a high-frequency flow (e.g. map-actions on player movement), note the call frequency in the audit before wrapping.
- **Observability** — no logging changes required; transaction rollback errors must propagate to the existing error-handling/logging path, not be swallowed.
- **Build verification** — full project gate per CLAUDE.md: `go test -race`, `go vet`, `go build` per changed module, `docker buildx bake atlas-<svc>` per touched service, `tools/redis-key-guard.sh`.

## 9. Open Questions

- Does task-114 land an emit-after-commit helper or outbox-enqueue-in-tx convention? Whichever it is, FR-2.2 defers to it — confirm at design time (after task-114 merges).
- atlas-maps and atlas-map-actions may have write paths on latency-sensitive flows; if the audit finds one, decide at design time whether to wrap or document as class D with a performance justification.

## 10. Acceptance Criteria

- [ ] `audit.md` exists with one section per audited service, every write entry point classified A/B/C/D with `file:line` citations; no service skipped.
- [ ] Every class-A and class-B flow is wrapped in `database.ExecuteTransaction`; zero manual `Begin/Commit` introduced.
- [ ] No Kafka emit occurs inside any transaction introduced by this task (unless task-114's landed convention explicitly requires enqueue-in-tx, in which case that convention is followed and cited).
- [ ] Every class-D verdict has a written justification in `audit.md`.
- [ ] Each wrapped flow has a rollback test (fail-mid-flow → prior writes reverted) and it passes under `go test -race`.
- [ ] All changed modules pass `go test -race ./...`, `go vet ./...`, `go build ./...`; every touched service passes `docker buildx bake atlas-<svc>`; `tools/redis-key-guard.sh` clean.
- [ ] No observable behavior change: same events emitted, same REST responses (FR-2.4).
- [ ] Branch is rebased on main after task-114 and task-116 merge, before remediation commits.
