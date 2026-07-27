# Plan Adherence Review — Tasks 1–3 Only (task-119-db-transaction-coverage)

**Scope:** This review covers ONLY Tasks 1, 2, and 3 of `plan.md`. Tasks 4–16 are
intentionally not implemented (blocked by the Task 4 merge gate) and are out of scope —
their absence is not a finding.

**Commits reviewed:**
- `ead3aed9ab` — Task 1 (fix `isTransaction` + regression tests)
- `35bb90a005` — Task 2 (`databasetest.FailWritesOn` helper + tests)
- `5ce27de633` — Task 3 (`audit.md`)

**Verification run:** `cd libs/atlas-database && go test -race ./... && go vet ./...` — both PASS
(`ok github.com/Chronicle20/atlas/libs/atlas-database`, `ok .../databasetest`, vet clean).

---

## Task 1 — Fix `isTransaction` + regression tests: IMPLEMENTED

- `libs/atlas-database/transaction_test.go` — all four named tests present and byte-identical
  to the plan's Step 1 listing: `TestExecuteTransaction_RollsBackOnError`,
  `TestExecuteTransaction_CommitsOnSuccess`, `TestExecuteTransaction_NestedJoinsOuterTransaction`,
  `TestExecuteTransaction_TenantCallbacksActiveInsideTransaction` (lines 24, 41, 54, 72).
- `libs/atlas-database/transaction.go:20-26` — `isTransaction` matches the plan's Step 3
  replacement exactly: nil-guard, then `db.Statement.ConnPool.(gorm.TxCommitter)` type
  assertion, with the same doc comment referencing GORM's own `finisher_api.go` idiom.
  Verified `gorm.TxCommitter` is a real exported type used the same way in
  `gorm.io/gorm@v1.30.0/finisher_api.go:625,702,712` (vendored module) — the fix is not
  invented, it mirrors GORM's own pattern.
- Commit message (`ead3aed9ab`) matches the plan's Step 5 text verbatim, including the "53 call
  sites across 18 services" detail.
- Step 6 ("surface standalone-PR recommendation") is explicitly the controller's job per the
  scoping instructions, not the implementer's — correctly absent from the diff, not a gap.
- `go test -race ./...` and `go vet ./...` both clean in `libs/atlas-database`.

No gaps found.

## Task 2 — `databasetest.FailWritesOn` helper: IMPLEMENTED

- `libs/atlas-database/databasetest/failwrites.go` — `WriteVerb` type, `WriteCreate`/
  `WriteUpdate`/`WriteDelete` constants, and `FailWritesOn(t, db, table, verbs...)` all present
  (lines 11-46), matching the plan's Step 3 listing exactly, including the "no verbs = fail all
  three" default (line 27-29) and per-verb `Callback().Create()/Update()/Delete().Before(...)`
  registration.
- Doc comment (lines 20-24) explicitly states "Raw `.Exec(...)` statements bypass GORM callbacks
  and are not intercepted" — the plan-required documentation caveat is present verbatim.
- `libs/atlas-database/databasetest/failwrites_test.go` — all three named tests present:
  `TestFailWritesOn_FailsNamedVerbOnNamedTableOnly`, `TestFailWritesOn_DefaultsToAllVerbs`,
  `TestFailWritesOn_DrivesRollbackThroughExecuteTransaction` (lines 29, 43, 54), matching the
  plan's Step 1 listing exactly.
- Commit message (`35bb90a005`) matches plan's Step 5.
- Tests pass (verified alongside Task 1's run — `databasetest` package `ok`).

No gaps found.

## Task 3 — Full write-path audit → audit.md: IMPLEMENTED

- `docs/tasks/task-119-db-transaction-coverage/audit.md` (450 lines) contains one section per
  service for all 14 named services from the plan's Step 1 list: atlas-npc-conversations,
  atlas-keys, atlas-families, atlas-marriages, atlas-monster-book, atlas-storage, atlas-account,
  atlas-ban, atlas-maps, atlas-map-actions, atlas-portal-actions, atlas-reactor-actions,
  atlas-party-quests, atlas-saga-orchestrator — confirmed count and names match exactly.
- Each section follows the plan's Step 4 template: Write inventory table (file:line / entry
  point / writes / class / flags), Exclusions table, Verdicts with justification.
- Taxonomy (A/B/C/D + [T]/[E] flags) is applied consistently and matches design §5.2's refined
  rule (RMW-with-one-write → C + race annotation, not B) — e.g. atlas-account's `GetOrCreate`
  (line 219) and atlas-storage's `GetOrCreateStorageId`/`GetOrCreateStorage` (lines 181-182) are
  both correctly classified C + race annotation, not B.
- Closing 14-service matrix present (lines 423-440) summarizing every service.
- Remediation pointers use the required `_(commit: pending — Task N)_` placeholder format
  throughout (24 occurrences, grep-verified) — this placeholder state is expected/correct per
  the scoping instructions, not a gap.
- Commit `5ce27de633` touches only `audit.md` (450 insertions, no code files) — consistent with
  the plan's "analysis-and-document task — no code changes" framing for Task 3.
- No `TODO`/`FIXME`/`XXX` markers in the audit or in the Task 1/2 code.

**Spot-verified factual claims in the audit (sampled, not exhaustive) — all checked out:**
- `gorm.TxCommitter` real type, used the same way GORM itself uses it (see Task 1 above).
- Marriages "dead code" claim: `AcceptProposalWithTransactionAndEmit` (processor.go:1581) has
  zero callers outside its own interface declaration (line 47) and definition (line 1581) —
  confirmed via grep across the whole service.
- Ban↔history "pairing refuted" claim: `atlas-ban`'s ban-consumer subscribes to topic
  `ban_command` (`kafka/consumer/ban/consumer.go:21`) while the history-consumer subscribes to
  `account_session_status_event` (`kafka/consumer/account/consumer.go:21`) — two independent
  Kafka topics, confirming the audit's claim that no single entry point writes both `bans` and
  `login_history`.
- npc-conversations `createWithSkipTracking`/`Create` line citations (processor.go:128-160)
  match the audit's `:130`/`:153` references and the two-write (conversation + recipe rebuild)
  shape described.

No gaps found. This is a thorough, evidence-backed sweep rather than a spot-check dressed up as
one — citations are specific enough (file:line, topic names, call-graph traces) to be
independently re-verified, and where the audit corrects its own design-phase hypotheses (e.g.
marriages, ban/history, maps) it shows the verification work rather than asserting the
correction.

---

## Overall Verdict (Tasks 1–3 only)

| Task | Status | Blocking issues |
|---|---|---|
| 1 | IMPLEMENTED | None |
| 2 | IMPLEMENTED | None |
| 3 | IMPLEMENTED | None |

Nothing blocks a standalone PR of these three commits. `go test -race ./...` and `go vet ./...`
are clean in `libs/atlas-database` (the only module touched by code, Tasks 1–2); Task 3 is
docs-only. Tasks 4–16 remain correctly gated and out of scope for this review.
