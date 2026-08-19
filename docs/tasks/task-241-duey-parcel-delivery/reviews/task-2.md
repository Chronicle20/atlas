# Review — Task 2: atlas-parcel model, builder, provider, administrator

Range: `c16e1e256..b179059b5` (1 commit, `b179059b5`).

## Scope

`git diff --stat` confirms exactly the 6 new files the brief's `### Files`
lists (`model.go`, `builder.go`, `provider.go`, `administrator.go`,
`builder_test.go`, `provider_tenant_test.go`) plus `go.mod`/`go.sum` deltas
for the new `atlas-constants`/`atlas-model`/`testify` direct requires. No
files outside the task-2 inventory are touched. `go.work.sum`'s pre-existing
unrelated modification (noted in the outer `git status`) is untouched by
this range — confirmed via `git diff c16e1e256..b179059b5 -- go.work.sum`
(empty). Scope matches the brief.

## Requirement-by-requirement

### Model — immutable, field-for-field with Task 1's Entity

`parcel/model.go` — every getter listed in the brief is present with the
exact signature (`Id() uuid.UUID`, `WorldId() world.Id`, ... through
`LastNotified() *time.Time`). All fields are unexported struct fields; the
only construction paths are `Builder.Build()` and `Make(Entity)` — no
exported zero-arg constructor bypasses validation. PASS.

`Make(e Entity) (Model, error)` (`builder.go:187-212`) round-trips every one
of Task 1's 21 domain fields (`gorm.Model`/`TenantId` deliberately excluded
— tenant scoping is off the model by design, matches frederick). `WorldId`
correctly narrows `byte → world.Id` on read and widens back `world.Id →
byte` on write (`entityFromModel`, `builder.go:222`). No renamed or invented
fields — checked field-by-field against task-1's `Entity` struct
(`task-1-report.md`) and both are 1:1. PASS.

### Builder

`NewBuilder()`, one `SetX` per field, `Build() (Model, error)` — matches the
brief exactly. `validate()` (`builder.go:154-166`) enforces `id != Nil`,
`senderId/recipientId != 0`, `status != ""` — reasonable minimal invariants
for a domain constructor; the meso-ceiling/mailbox-capacity/message-length
constraints from the plan's global constraints are correctly deferred to
task-3 (this task's `### Files`/`### Interfaces produced` inventory names
no validation logic beyond the four basics, and task-3's brief owns
`processor.go`/`errors.go`). PASS.

### Provider

`ById`, `ByRecipient`, `BySender`, `ReceivableByRecipient` all present with
the brief's exact signatures, each `func(db *gorm.DB) model.Provider[...]`
via `database.EntityProvider[...]`, composed with `model.Map`/
`model.SliceMap` per the `atlas-mts/listing` idiom. WHERE clauses use
name-keyed maps, not struct conditions — correctly avoids GORM's
zero-value-elision hazard for `world_id = 0` / a zero status string.
`ReceivableByRecipient` hardcodes `status = StatusPending` (not a caller
param) and adds `receivable_at <= now` — matches the brief. No provider adds
a manual `tenant_id` predicate — verified by reading `provider.go` in full;
scoping is left entirely to `atlas-database`'s registered `tenant:query`
callback (`libs/atlas-database/tenant_scope.go:47-77`), which is registered
for `Query`/`Row` (and `Update`/`Delete`) before the underlying gorm
callback and injects `tenant_id = ?` from `tenant.FromContext(ctx)` whenever
the target schema has a `tenant_id` column. PASS.

### Administrator

`Create`, `UpdateStatus`, `StampNotified` present with the brief's exact
signatures. `Create` generates an id when the model's is `uuid.Nil`, leaves
`TenantId` zero for the `tenant:create` callback to inject
(`tenant_scope.go:81-113`, `injectTenantIdIfZero`), and returns `Make` of
the persisted row. `UpdateStatus` sets `status`+`resolved_at` atomically in
one `Updates` call, `StampNotified` no-ops on an empty id slice and updates
`last_notified` for a batch otherwise. Both rely on the `tenant:update`
callback (also registered, `tenant_scope.go:49`) for isolation on the
`Where("id = ?", ...)`/`Where("id IN ?", ...)` mutations — no manual
`tenant_id` predicate, consistent with the providers. PASS.

## The world_id judgment call

**Verdict: the shipped behavior is correct, and is the only defensible
reading of the brief.**

The Entity's actual index set (task-1's report, `entity.go`) is:
`idx_parcels_recipient = (tenant_id, recipient_id, status)`,
`idx_parcels_sender = (tenant_id, sender_id, status)`,
`idx_parcels_sweep = (tenant_id, status, expires_at)`. `world_id` is not a
column in any composite index, and the brief's own signature for
`ByRecipient`/`ReceivableByRecipient` takes a `worldId world.Id` parameter
that a caller — task-3's processor, task-4's REST handler — will pass a
real, non-zero world for in a multi-world tenant (a single tenant server
config commonly runs several `world_id` values; recipient character ids are
not guaranteed globally unique across worlds within one tenant). Dropping
`world_id` from the WHERE clause to strictly honor "indexed columns only"
would return parcels addressed to a character in a *different* world than
the one the caller is asking about — a real correctness bug, not a
performance nit, and strictly worse than the residual-filter cost.

On the indexing question: keeping `world_id` as a residual predicate after
`(tenant_id, recipient_id, status)` narrows the row set does not defeat
NFR-4 (no full table scan) — SQLite/Postgres will still use the composite
index to locate the `(tenant_id, recipient_id, status)`-matching rows, then
filter that already-small row set by `world_id` in-memory/via index
condition pushdown. This is exactly the shape `atlas-mts/listing`'s
`browseFilterQuery` uses (index-narrow, then layer additional non-indexed
predicates) — the implementer's own citation checks out; I independently
confirmed `browseFilterQuery`'s shape matches (`atlas-mts/listing/provider.go`,
not modified by this diff, reference-only, consistent with the brief's own
pointer to it as the world-scoped provider reference).

The one thing this decision does *not* get free: if `ByRecipient` is ever
called on a tenant/recipient/status combination with a very large row count
(e.g. a recipient with hundreds of mailbox rows across many worlds — outside
the 10-parcel mailbox cap, so bounded in the pending case, but `ByRecipient`
takes an arbitrary `status` and is not itself capped), the residual
`world_id` filter would degrade toward a partial scan of that recipient's
rows. Given the mailbox cap (10 pending) and that resolved/expired parcels
are presumably pruned by task-23's sweep, this is not a currently
demonstrated problem, and is not something task-2 can fix without inventing
a fourth index column the brief never asked for. Not a blocking finding —
recorded for the controller's awareness in case task-23/24 query volumes
change this calculus.

## Tests

`TestBuilderRoundTrip` (4 subtests) and `TestProviderTenantIsolation` (3
subtests) both present, both exercise every case in the brief's tables.
`TestProviderTenantIsolation` genuinely tests tenant isolation, not just
that the query runs: two parcels are seeded under different tenant ids with
the *same* `recipientId`/`senderId`, and the assertion is `require.Len(...,
1)` plus an id match — this would fail (return 2 rows, or the wrong row) if
the manual-tenant-predicate omission were wrong or if the callback weren't
wired. The implementer's report includes RED evidence (files temporarily
removed, compile failure matching the brief's expectation) and GREEN
evidence (full suite pass) — I did not re-run these per the task
instructions but the evidence format is convincing and the test bodies
substantiate it on inspection. PASS.

No `*_testhelpers.go` file was created; `newParcelTenantDB` in
`provider_tenant_test.go` is a local unexported helper function (matching
the "helper function, not a builder-avoidance shim" allowance already used
in task-1's `entity_test.go`), and the Builder is used for the
`TestBuilderRoundTrip` cases, satisfying "use the project's Builder pattern
for test setup" for the cases where a `Model` needs constructing.

## libs/atlas-constants check

`world.Id` (`libs/atlas-constants/world`) is reused, not reinvented — grep
of the new files shows no new domain type/alias/constant defined outside
what task-1 already landed (`StatusPending` etc., `ReceivableDelay`,
`ExpiryWindow`). PASS.

## Cross-task additivity (forward-looking, not blocking)

- **Task 3** consumes `Model`'s getters, `NewBuilder`, `Make`, the four
  providers and the three administrator functions directly by name/shape —
  all present exactly as documented in the report's interface record.
- **Task 23** is expected to add an `Expired(now, limit)`-shaped function to
  `provider.go` and reuse `UpdateStatus` for the sweep's status transition —
  both are additive; no signature in this diff would need to change.
- **Task 24**'s notification sweep is expected to query off
  `ReceivableByRecipient` or a similar new provider function and call
  `StampNotified` — additive, no conflict.

I did not attempt to verify task 23/24's actual brief text (out of scope
for this review); the above is a structural compatibility check against
what task-2 shipped, per the reviewer brief's request to judge whether both
are additive from here.

## Findings

None blocking. No non-blocking findings beyond the one forward-looking note
under "The world_id judgment call" above (recorded for awareness, not a
defect in this task).

## Not evaluable

- Whether task-3/23/24's actual briefs match the "additive" expectation
  above — those tasks have not landed yet; I can only assess structural
  compatibility of what task-2 shipped, not their as-yet-unwritten content.
