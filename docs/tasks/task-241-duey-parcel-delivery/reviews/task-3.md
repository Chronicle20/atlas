# Review — Task 3: atlas-parcel processor (state machine, injectable clock, error taxonomy)

Range reviewed: `b179059b5..91dc1cae9` (1 commit).

## Scope

Diff stat:

```
 .../atlas-parcel/atlas.com/parcel/parcel/errors.go |  27 +++
 .../atlas-parcel/atlas.com/parcel/parcel/model.go  |   8 +
 .../atlas.com/parcel/parcel/processor.go           | 169 +++++++++++++++
 .../atlas.com/parcel/parcel/processor_test.go      | 227 +++++++++++++++++++++
 4 files changed, 431 insertions(+)
```

Matches the brief's file list exactly (`processor.go`, `processor_test.go`, `errors.go` new;
`model.go` gets one added method). No scope drift. Reviewed the full diff plus the
Task 1/2 contracts it depends on (`entity.go`, `provider.go`, `administrator.go`,
`builder.go`) and the relevant slices of `prd.md`/`design.md` for the requirements those
files encode (timers, NFR-3 idempotency, gate 12 destination-independence).

## PASS — requirement by requirement

- **Sentinel taxonomy** (`errors.go`): `ErrNotFound`, `ErrNotPending`, `ErrNotRecipient`,
  `ErrNotYetReceivable`, single `var (...)` block, doc comment per sentinel, matches the
  `listing/errors.go` shape the brief named to copy. `processor.go:1-169`.
- **Injectable clock**: `ProcessorImpl.now func() time.Time`, defaulted to `time.Now` only
  in `NewProcessor` (`processor.go:45`), overridden via the unexported copy-returning
  `withClock` (`processor.go:56-62`), mirroring `pending_change`'s
  `withTransferEligibilityGates` seam. Grepped `processor.go`/`model.go` for `time.Now` —
  the only two call sites are the doc comment and the `NewProcessor` default; `resolve` and
  `HasInFlight` both go through `p.now()` (`processor.go:96, 137`). Confirms "time.Now()
  must not be called directly in the state machine."
- **`Model.Receivable(now)`** (`model.go:86-89`): `status == StatusPending &&
  !receivableAt.After(now)`, exactly the brief's Step 3 formula.
- **`Receive`/`Discard` transition shape**: one `database.ExecuteTransaction`, re-read via
  `ById(id)(tx)()`, ownership check, caller gate, `UpdateStatus(tx)`, re-read and return
  (`processor.go:141-169`). All 8 brief-specified `TestProcessorReceive`/
  `TestProcessorDiscard` subtests are present verbatim and pass per the implementer's
  report; I did not re-run them (per instructions) but did read the test bodies and confirm
  each asserts what its subtest name claims (`processor_test.go:65-179`).
- **`HasInFlight` outbound half**: `BySender(characterId, StatusPending)`, non-empty ⇒ true,
  matching gate 12's "outbound pending" half. `processor_test.go:181-190` (outbound pending)
  passes.
- **Builder-pattern test setup**: `seedParcel` composes `NewBuilder()...Build()` then
  `Create` (`processor_test.go:44-68`); no `*_testhelpers.go` file exists in the package.
- **`libs/atlas-constants` reuse**: only `world.Id`, already established by Task 2; no new
  domain type/constant introduced.
- **`Create` scope boundary** (the pre-ruled decision): `Create` is a bare pass-through to
  `administrator.Create` (`processor.go:107-109`) — no meso-ceiling, level-meso-limit,
  mailbox-capacity, or message-length check anywhere in this diff, and none of the four
  sentinels maps to those PARCEL result arms. Confirmed it does not half-validate: there is
  no partial check of any one of the four constraints either. Consistent with the ruled
  boundary (those checks belong to Tasks 16/17 in atlas-channel, applied before `Create` is
  reached).
- **Timers are not computed in `Create`**: verified this is correct, not a gap — `design.md`
  §4.2 (line 232-234) states `AcceptToParcelPayload` carries "the computed `ReceivableAt` /
  `ExpiresAt`" so that "atlas-parcel creates the row from the payload alone." Timer
  computation is a saga/producer-side responsibility, not this processor's.
- **Total state coverage**: `Receive`/`Discard`/`HasInFlight` treat every non-`pending`
  status (`received`, `discarded`, and the not-yet-shipped `expired`) uniformly as
  `ErrNotPending`/excluded — no unreachable or ambiguous branch for a status this task
  doesn't yet produce.
- **NFR-6 logging**: `p.l` (the injected `logrus.FieldLogger`) is never invoked anywhere in
  `processor.go`. Checked the pattern file this task was told to copy
  (`services/atlas-merchant/atlas.com/merchant/frederick/processor.go`) — it also carries an
  unused logger field and logs nothing from the processor layer. This is the repo's
  established convention (logging happens at the REST/consumer boundary, not the
  processor), so this is **not** a defect.

## Blocking findings

1. **`HasInFlight`'s inbound half hardcodes `world.Id(0)`, breaking gate 12 for every
   non-zero world.** `services/atlas-parcel/atlas.com/parcel/parcel/processor.go:97`:
   ```go
   inbound, err := ReceivableByRecipient(characterId, world.Id(0), now)(tdb)()
   ```
   `ReceivableByRecipient` (Task 2, `provider.go:70-83`) filters with an exact-match
   `"world_id": byte(worldId)` predicate. Tenants in this codebase are multi-world
   (confirmed: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
   defines `"worlds": [{"name": "Scania"...}, {"name": "Bera"...}, ...]`), and a parcel's
   `WorldId` is the real world it was sent within (same-world delivery, FR-7) — not
   necessarily `0`. For any recipient whose inbound parcel has `WorldId != 0`, this query
   silently returns zero rows, so `HasInFlight` reports `false` even though a receivable
   inbound parcel exists. Task 26's world-transfer gate 12 is the direct consumer of this
   method (per the interface record) and is explicitly required to be "destination-
   INDEPENDENT" and answer "any parcel in flight, either direction" (`design.md:680-682`);
   as shipped, the inbound half of that answer is wrong for every world but world 0,
   silently letting a character with an unclaimed inbound parcel transfer worlds — exactly
   the scenario PRD's acceptance criteria (`prd.md:516`) says must be refused.

   Not caught by tests: every `seedParcel` call in `processor_test.go` explicitly sets
   `SetWorldId(0)` (`processor_test.go:50`), so `TestProcessorHasInFlight`'s "inbound
   receivable" subtest exercises exactly the one world value that happens to work.

   The task-3 report's own text acknowledges the hardcoded value but rationalizes it as
   "a pass-through of the residual-filter convention `ReceivableByRecipient` already
   applies, not a genuine per-world restriction" — that characterization is incorrect:
   `ReceivableByRecipient`'s WHERE clause is a genuine exact-match filter on `world_id`, so
   passing a wrong constant is a genuine (and incorrect) restriction, not a no-op.

   Fix shape: either drop the `world_id` predicate for this internal, recipient-scoped
   query (a character's `recipientId` is already world-unambiguous — same-world delivery is
   the only kind that exists), or thread the character's real `worldId` through
   `HasInFlight`. Either is a signature/behavior change inside this task's own file, not a
   downstream task's problem to patch around.

2. **`GetById` does not map a missing row to `ErrNotFound`, contradicting the task's own
   documented contract.** `services/atlas-parcel/atlas.com/parcel/parcel/processor.go:65-67`:
   ```go
   func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
       return ById(id)(p.db.WithContext(p.ctx))()
   }
   ```
   `ById` (Task 2, `provider.go:20-28`) on a `gorm.ErrRecordNotFound` returns
   `model.ErrorProvider[Model](err)`, which (confirmed by reading
   `libs/atlas-model/model/processor.go:335`) returns the error **unchanged** — the raw GORM
   sentinel, not this package's `ErrNotFound`. `GetById` forwards that raw error verbatim; it
   is never mapped to `ErrNotFound`. The task-3 report's own interface record states
   plainly: "`ErrNotFound` — `GetById`/`Receive`/`Discard` on an id with no row" — that
   sentence is factually wrong for `GetById` as shipped. `Receive`/`Discard` are fine (their
   shared `resolve` helper explicitly translates a not-found re-read to `ErrNotFound` at
   `processor.go:151-153`); `GetById` is not.

   This matters because it is exactly the kind of interface-record claim Task 4 (REST) and
   Task 26 (gate lookup) are told to build against: a caller doing
   `errors.Is(err, parcel.ErrNotFound)` after `GetById` to decide "return 404 / PARCEL
   result X" will never match, and will instead surface a raw GORM error (or a 500) on a
   perfectly ordinary not-found. Not exercised by any test — `processor_test.go` never
   calls `GetById` on a missing id (only after a successful `Receive`/`Discard`, or via the
   `resolve` path which already handles the mapping correctly).

## Non-blocking findings

3. **`resolve`'s transition is a read-then-write, not a compare-and-swap; concurrent
   duplicate `Receive`/`Discard` calls can both succeed.** `processor.go:141-169`: the
   transaction re-reads the row (`ById`), checks status/ownership in Go, then calls
   `UpdateStatus(tx)(id, status, now)` — whose own WHERE clause is `id = ?` only
   (`administrator.go:27-36`, no `status = 'pending'` predicate). Under default (READ
   COMMITTED) Postgres isolation, two overlapping, uncommitted transactions can both read
   `status = pending`, both pass the gate, and both successfully `UPDATE` — the second
   silently overwriting the first's `resolved_at`, with both callers observing success. That
   is exactly the double-award NFR-3 exists to prevent ("A replayed receive must not award
   twice," `prd.md:420-422`), and the PRD explicitly names a different, race-safe pattern to
   follow (`database.Once`/`ApplyOnce`'s unique-constraint claim row, or the conditional
   `UPDATE ... WHERE status = 'pending' ... RETURNING` this same design uses for the expiry
   sweep, `design.md:620-629`, and for `atlas-mts/listing.transitionToSellerHolding`'s
   affected-row-count race arbiter — the very function this task's own doc comment claims to
   mirror). As shipped, only *sequential* replay (redelivery strictly after the first
   transaction commits) is actually safe; true concurrent duplicate delivery is not. No test
   in `processor_test.go` exercises concurrent replay (SQLite in-memory, single-threaded) —
   the "already received"/"already discarded" subtests only prove sequential-replay safety,
   which any implementation persisting a status column would pass trivially.

   Attribution: this exact shape (re-read + `UpdateStatus`, no conditional WHERE) is
   specified verbatim in `task-3-brief.md`'s Step 3 ("re-read the row inside the
   transaction, validate ... then `UpdateStatus`"), so the implementer followed the brief as
   written; this is a plan-level gap inherited into the code, not an independent deviation.
   Flagging for the controller because the shipped state machine, not just the plan text, is
   what Task 4/15 will build atop, and "the second delivery finds status != pending" is true
   only for the common (sequential) case, not the general one the NFR names.

## Not evaluable

- Whether `Receive`/`Discard` are actually reachable concurrently in production (REST
  double-submission vs. Kafka at-least-once redelivery timing) depends on Task 4's and
  Task 15's transport-layer behavior, neither of which exists yet in this range. Finding 3
  is reported as a property of the code as written, not a confirmed production incident.
- Whether Task 4's REST/PARCEL-result mapping was actually going to rely on
  `errors.Is(err, ErrNotFound)` for `GetById` specifically (finding 2) can't be confirmed
  without Task 4's code, which postdates this range; the report's own interface record is
  what establishes the expectation, and that record is what's contradicted by the code.

## Verdict rationale

Two blocking defects: one silently breaks a cross-service consumer (gate 12) for the
common multi-world case, the other contradicts this task's own documented error contract in
a way a downstream consumer would reasonably rely on. Both are inside this task's own
surface (not something a later task must route around), both are unverified/untested by the
task's own test suite, and both are concrete enough to hand to an implementer without
further investigation. The concurrency gap is real but lower-confidence/lower-severity and
partially a plan-level inheritance, so it's recorded as non-blocking.
