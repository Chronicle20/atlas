# task-114 Outbox Migration Inventory

Per-service audit record (FR-3.5). Sections are appended as each service
migrates. "Left direct" sites keep the direct producer path deliberately.

## atlas-character

Module: `services/atlas-character/atlas.com/character`. All line numbers
below reflect `character/processor.go` (and `drop/processor.go`,
`skill/processor.go` where noted) as of the Task 9 commit
(`6bf4a2218`), i.e. after Tasks 7, 8, and 9 all landed.

### Migrated

Each site now enqueues its success-path event(s) via
`message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))` inside the same
`database.ExecuteTransaction` closure that performs the DB write, instead of
firing them through `producer.ProviderImpl` (direct Kafka) in-tx or
fire-and-forget after the transaction closed.

- `character/processor.go:782` `RequestChangeMeso` — Task 7. Fixed unchecked
  `dynamicUpdate` error, fake-nil overflow return (now `ErrMesoOverflow`),
  and moved `MESO_CHANGED`/`STAT_CHANGED` in-tx.
- `character/processor.go:819` `AttemptMesoPickUp` — Task 7. Same
  unchecked-error/overflow fixes; `STAT_CHANGED` moved in-tx.
- `character/processor.go:844` `RequestDropMeso` — Task 7. Headline fix: the
  post-commit fire-and-forget `STAT_CHANGED` emit (no atomicity guarantee vs.
  the meso deduction) is now enqueued inside the transaction.
- `character/processor.go:881` `RequestChangeFame` — Task 8. Fixed unchecked
  `dynamicUpdate` error; `FAME_CHANGED`/`STAT_CHANGED` moved in-tx.
- `character/processor.go:907` `RequestDistributeAp` — Task 8. Restructured
  around the `rejectEmit` closure device (see Left Direct); success-path
  `STAT_CHANGED` moved in-tx.
- `character/processor.go:1006` `RequestDistributeSp` — Task 9. Non-`AndEmit`
  flow; `SetSP` write and `STAT_CHANGED` emit both moved inside the existing
  `ExecuteTransaction` closure.
- `character/processor.go:254` `CreateAndEmit` — Task 9, Pattern A.
- `character/processor.go:309` `DeleteAndEmit` — Task 9, Pattern A.
- `character/processor.go:364` `DeleteForSagaCompensationAndEmit` — Task 9,
  Pattern A (applied uniformly even to the already-absent/no-write
  idempotency branch — harmless empty-transaction commit).
- `character/processor.go:382` `DeleteByAccountIdAndEmit` — Task 9, custom
  variant: **not** one shared outer transaction across the account's whole
  character batch. Loops and calls the already-migrated `DeleteAndEmit` once
  per character instead, so each character gets its own atomic
  mutation+enqueue transaction while preserving the original
  best-effort/log-and-continue-per-character contract (a shared transaction
  would let one character's failure abort every subsequent delete in the
  same batch — a regression).
- `character/processor.go:485` `ChangeJobAndEmit` — Task 9, Pattern A.
- `character/processor.go:517` `ChangeHairAndEmit` — Task 9, Pattern A.
- `character/processor.go:551` `ChangeFaceAndEmit` — Task 9, Pattern A.
- `character/processor.go:585` `ChangeSkinAndEmit` — Task 9, Pattern A.
- `character/processor.go:629` `AwardExperienceAndEmit` — Task 9, Pattern A
  (also buffers an `EnvCommandTopic` `awardLevelCommandProvider` message
  in-tx when a level was earned — this is the method's own follow-on effect
  of the state change it just made, not a cross-service command emit, so it
  migrates with the rest of the buffer).
- `character/processor.go:693` `DeductExperienceAndEmit` — Task 9, Pattern A.
- `character/processor.go:736` `AwardLevelAndEmit` — Task 9, Pattern A.
- `character/processor.go:1214` `ChangeHPAndEmit` — Task 9, Pattern A.
- `character/processor.go:1262` `SetHPAndEmit` — Task 9, Pattern A.
- `character/processor.go:1314` `ChangeMPAndEmit` — Task 9, Pattern A.
- `character/processor.go:1348` `ClampHPAndEmit` — Task 9, Pattern A.
- `character/processor.go:1384` `ClampMPAndEmit` — Task 9, Pattern A.
- `character/processor.go:1420` `ProcessLevelChangeAndEmit` — Task 9, Pattern
  A. Required a bundled bug fix: `WithTransaction` (line 154) was missing an
  `sdp: p.sdp` field copy, which would have left `p.sdp` nil on the
  tx-scoped processor and panicked the first time a leveling character had a
  `resolveHPMPGainParams`-eligible skill. Fixed as part of this migration
  (see Task 9 report §3).
- `character/processor.go:1654` `ProcessJobChangeAndEmit` — Task 9, Pattern
  A.
- `character/processor.go:1749` `UpdateAndEmit` — Task 9, Pattern A (wraps
  `Update`, whose `dynamicUpdate`+`mb.Put` calls were already interleaved
  inside their own `ExecuteTransaction`; confirmed the outer wrap is a
  correct pass-through under re-entrancy).
- `character/processor.go:1926` `ResetStatsAndEmit` — Task 9, Pattern A.
- `character/processor.go:1990` `RebalanceAPAndEmit` — Task 9, Pattern A.

### Left direct

- `character/processor.go:792` `RequestChangeMeso` not-enough-meso
  `rejectEmit` closure — rejection emit; no state was written on this path,
  captured as a closure and fired via the direct producer only after
  `ExecuteTransaction` returns `ErrNotEnoughMeso`, outside the transaction
  boundary (Task 7).
- `character/processor.go:854` `RequestDropMeso` not-enough-meso `rejectEmit`
  closure — same reason as above (Task 7).
- `character/processor.go:915` `RequestDistributeAp` not-enough-AP
  `rejectEmit` closure — rejection emit, no state written (Task 8).
- `character/processor.go:979` `RequestDistributeAp` invalid-ability
  `rejectEmit` closure — rejection emit, no state written (Task 8).
- `character/processor.go:990` `RequestDistributeAp` update-failure
  `rejectEmit` closure — emitted only when `dynamicUpdate` itself failed, so
  no committed state change to be atomic with (Task 8).
- `character/processor.go:417` `LoginAndEmit` and `character/processor.go:451`
  `LogoutAndEmit` — the wrapped `Login`/`Logout` methods perform **no DB
  write**: they call `location.GetField` (a REST call to atlas-maps) and
  `mb.Put` a single event, with no `dynamicUpdate`/`create`/`delete`
  anywhere in either method. **Known cross-processor atomicity gap,
  recorded for follow-up, not fixed here**: the actual session-history DB
  write (`StartSession`/`EndSession`, `session/history/processor.go`) lives
  in a separate package/processor and is invoked from a *different* caller
  (`kafka/consumer/session/consumer.go:64,79-80` for login;
  `session/task.go:51` for logout/timeout) with no shared transaction across
  the history write and the `LoginAndEmit`/`LogoutAndEmit` call. True
  atomicity between the session-history row and the LOGIN/LOGOUT event would
  require restructuring both call sites to share one transaction across the
  `history` and `character` processors — out of this task's file scope
  (`character/processor.go` only) and outside the Pattern A/B recipe (which
  assumes the `*AndEmit` method's own wrapped call is the DB write). This is
  a discrepancy from the original plan enumeration (which expected these two
  methods to migrate); reclassified after reading the actual code (Task 9).
- `drop/processor.go:31,35,39` — `producer.ProviderImpl(...)(drop2.EnvCommandTopic)(...)`
  (drop-meso, request-pickup, cancel-reservation): unbuffered command emits
  to the **drop** service's topic; this service owns no drop-storage DB
  write to be atomic with (D7 command-to-another-service carve-out).
- `skill/processor.go:42,46` — `producer.ProviderImpl(...)(skill2.EnvCommandTopic)(...)`
  (create/update skill), called from `RequestDistributeSp` after its own tx
  commits: unbuffered command emits to the **skill** service's topic, same
  D7 carve-out.

### Notes

- **Deliberately dropped emit (behavior change, not a pure refactor)**:
  `RequestDistributeAp`'s `GetById`-failure branch
  (`character/processor.go:910-912`) previously emitted a `STAT_CHANGED`
  event via the direct producer with `channel.NewModel(c.WorldId(), 0)` —
  but `c` is the zero-value `Model` on this path (`GetById` failed, so `c`
  was never populated), meaning `c.WorldId()` dereferenced a zero-value
  model and the emitted event carried garbage (effectively `WorldId() == 0`
  in that case, not a real world id). This is a latent nil/garbage-data bug
  in the original code, not a behavior worth preserving; there is no way to
  build a `rejectEmit` closure here with a real world id, and the brief
  explicitly rules out substituting `channel.NewModel(0, 0)`. The branch now
  simply `return err`s with no emit (Task 8).
- **`database.ExecuteTransaction` atomicity is currently LATENT, fleet-wide,
  until task-119 lands.** `libs/atlas-database/transaction.go`'s
  `isTransaction(db)` check (`db.Statement != nil && db.Statement.ConnPool
  != nil`) is true for essentially every `*gorm.DB` — confirmed empirically
  in this task (see `outbox_acceptance_test.go`) that it is true even for a
  freshly-`gorm.Open`'d in-memory sqlite handle with no prior query,
  because `gorm.Open` itself populates `Statement.ConnPool` on the root
  handle. `ExecuteTransaction` therefore takes the `isTransaction==true`
  branch and calls `fn(db)` directly instead of `db.Transaction(fn)`, so
  **no real BEGIN/COMMIT/ROLLBACK wraps the enqueue+write today**, in
  production (Postgres) or in this module's own tests (sqlite). All of the
  Tasks 7–9 migrations above use the *correct seam* — the enqueue and the
  domain write are issued inside the same `ExecuteTransaction` closure, so
  they will become genuinely atomic the moment task-119's `TxCommitter` fix
  lands in `libs/atlas-database` — but until then, a crash between the
  enqueue and the write (or vice versa) is not actually prevented by this
  migration. See `docs/tasks/task-114-outbox-adoption/inventory.md` (this
  file) and project memory `bug_execute_transaction_noop.md`. This is why
  `TestOutbox_RollbackDiscardsEnqueuedEvents` in `outbox_acceptance_test.go`
  is currently `t.Skip`'d — it fails against real behavior today (a
  "rolled-back" transaction still leaves the enqueued row committed) and
  will pass once task-119 lands.
- No net-new test coverage was added in Tasks 8 or 9 for the specific
  methods they touched (`RequestChangeFame`, `RequestDistributeAp`, and the
  23 `*AndEmit` sites + `RequestDistributeSp`) beyond what already existed;
  Task 7's `meso_outbox_test.go` and this task's
  `outbox_acceptance_test.go` are the only outbox-row-count assertions in
  this module.

## atlas-inventory

Module: `services/atlas-inventory/atlas.com/inventory`. Line numbers below
reflect `compartment/processor.go`, `inventory/processor.go`, and
`asset/processor.go` as of the Task 11 commit, updated for the D7
review-fix pass (see "Fix pass (D7 review)" at the end of this section).
Wiring: `main.go` appends `outboxlib.Migration` to
`database.SetMigrations(...)` and boots the drainer right after
`database.Connect(...)`, using the existing
`tdm := service.GetTeardownManager()` var (this service's local name for
the teardown manager, not `lifecycle`).

**Counts**: 22 `*AndEmit` call sites migrated to the outbox (Pattern A;
unchanged by the D7 fix pass — no site was un-migrated). 7 left-direct
entries: 3 whole-method sites with no DB write (`RequestReserveAndEmit`,
`CancelReservationAndEmit`, `inventory/processor.go`'s `CreateAndEmit`
rollback branch), plus 4 failure-path sub-sites *inside* already-migrated
methods (`Accept`, `Release`, `AttemptEquipmentPickUp`,
`AttemptItemPickUp`) that were incorrectly riding into the outbox buffer
and are now routed direct per the fix pass below.

### Migrated

Each site now wraps its own `database.ExecuteTransaction` (a new outer one
for sites that previously had none, or the site's pre-existing one) and
enqueues its buffer via `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))`
instead of `message.Emit(p.producer)` / `message.Emit(producer.ProviderImpl(p.l)(p.ctx))`,
calling `p.WithTransaction(tx).Method(mb)(...)` so the wrapped method's own
(often nested/reentrant) `ExecuteTransaction` and DB writes run against the
same `tx`.

- `inventory/processor.go:68` `CreateAndEmit` — Pattern A.
- `inventory/processor.go:124` `DeleteAndEmit` — Pattern A.
- `asset/processor.go:268` `ChangeTemplateAndEmit` — Pattern A.
- `asset/processor.go:297` `DeleteAndEmit` — Pattern A; the wrapped
  `Delete` had no `ExecuteTransaction` of its own (bare `deleteById` call),
  so this site gains a **new** outer transaction wrapping the write.
- `compartment/processor.go:236` `EquipItemAndEmit` — Pattern A.
- `compartment/processor.go:350` `RemoveEquipAndEmit` — Pattern A.
- `compartment/processor.go:408` `MoveAndEmit` — Pattern A (wraps
  `MoveAndLock`, which locks then delegates to `Move`).
- `compartment/processor.go:610` `IncreaseCapacityAndEmit` — Pattern A.
- `compartment/processor.go:647` `DropAndEmit` — Pattern A.
- `compartment/processor.go:816` `ConsumeAssetAndEmit` — Pattern A.
- `compartment/processor.go:878` `DestroyAssetAndEmit` — Pattern A.
- `compartment/processor.go:929` `ExpireAssetAndEmit` — Pattern A.
- `compartment/processor.go:989` `CreateAssetAndEmit` — Pattern A (wraps
  `CreateAssetAndLock`, which locks then delegates to `CreateAsset`).
  **Corrected in the D7 review-fix pass**: the original note here claimed
  the failure branch's `CreationFailedEventStatusProvider` "rides along
  into the outbox" — that is **wrong** for this call path.
  `CreateAsset`'s failure branch puts the rejection on `mb` *and then
  `return txErr`* (non-nil); since `CreateAssetAndLock` just forwards that
  error up to `CreateAssetAndEmit`'s `message.Emit(outbox.EmitProvider(...))`
  closure, `Emit`'s `f(b)` returns non-nil and the buffer is **discarded,
  never flushed** — a pre-existing no-op emit on this path, in both the old
  (direct-producer) and new (outbox) code. Behavior unchanged by Task 11.
  Left uncorrected in code (no fix needed here). NOTE: this no-op claim
  holds only for *this* direct entry point — see the Notes-section caveat
  below for the residual case where `CreateAsset` is invoked from
  `AttemptItemPickUp` instead.
- `compartment/processor.go:1093` `AttemptEquipmentPickUpAndEmit` — Pattern
  A. Success path (`RequestPickUp`, bundled with the committed
  `CreateFromModel` write) is correctly migrated and **unchanged** by the
  D7 fix pass. The failure branch (`CancelReservation`, a COMMAND to
  atlas-drop after a rolled-back write) was incorrectly riding into the
  outbox and is now routed DIRECT — see Left direct below.
- `compartment/processor.go:1167` `AttemptItemPickUpAndEmit` — Pattern A.
  Success path (pickup-consume command + `RequestPickUp`, bundled with the
  committed `CreateAsset`/`UpdateQuantity` writes) is correctly migrated
  and **unchanged** by the D7 fix pass. The failure branch
  (`CancelReservation`) was incorrectly riding into the outbox and is now
  routed DIRECT — see Left direct below and the residual-nuance Note.
- `compartment/processor.go:1280` `RechargeAssetAndEmit` — Pattern A.
- `compartment/processor.go:1338` `MergeAndCompactAndEmit` — Pattern A.
- `compartment/processor.go:1346` `CompactAndSortAndEmit` — Pattern A.
- `compartment/processor.go:1467` `AcceptAndEmit` — Pattern A. Success path
  unchanged. **Fixed in the D7 review-fix pass**: the failure branch's
  `ErrorEventStatusProvider` (compartment `AcceptCommandFailed`) — a
  rejection reflecting no committed state change — was riding along into
  the outbox on the same shared `mb`; it is now captured in a `rejectEmit
  func() error` closure and fired via `producer.ProviderImpl(p.l)(p.ctx)`
  on the DIRECT path after the inner tx returns, then `Accept` still
  returns `nil` to the caller (unchanged contract). See Left direct below.
- `compartment/processor.go:1580` `ReleaseAndEmit` — Pattern A. Same fix as
  `Accept` above for the `ReleaseCommandFailed` rejection.
- `compartment/processor.go:1782` `ModifyEquipmentAndEmit` — Pattern A
  (wraps a method whose body already was `return
  database.ExecuteTransaction(...)`; now the outer transaction is opened
  first so `outbox.EmitProvider` can bind to `tx`, and the inner call
  nests/re-enters against the same handle).
- `compartment/processor.go:1805` `ChangeTemplateAndEmit` — Pattern A, same
  shape as `ModifyEquipmentAndEmit`.

### Left direct

- `compartment/processor.go:743` `RequestReserveAndEmit` — wrapped method
  `RequestReserve` performs **no DB write**: it only reads the compartment/
  asset (gorm reads), mutates the Redis-backed `ReservationRegistry`
  (`AddReservation`), and puts a `ReservedEventStatusProvider` event. Kept
  on `p.producer` (struct-field direct provider) despite the pre-existing
  (superfluous) `database.ExecuteTransaction` wrapper around the read+
  registry-write; verified by reading, not assumed from the tx wrapper's
  presence.
- `compartment/processor.go:790` `CancelReservationAndEmit` — wrapped
  method `CancelReservation` has no `ExecuteTransaction` at all: reads the
  compartment (gorm read), removes the reservation from the Redis-backed
  registry, and puts the cancellation event. No DB write. Kept on
  `p.producer`.
- `inventory/processor.go:109` `CreateAndEmit`'s rollback-failure branch —
  already structurally its own `message.Emit(producer.ProviderImpl(...))`
  call on a **fresh** buffer (separate from the tx-scoped one used for the
  success path), firing `CreationFailedEventStatusProvider` only after the
  inner `ExecuteTransaction` has already rolled back/failed. This is the
  textbook rejection-event shape from the recipe (own dedicated emit call,
  no state change) and needed no restructuring.

**Added by the D7 review-fix pass** (failure-path sub-sites inside
already-migrated `*AndEmit` methods; the outer `AndEmit` wrapper itself
stays Pattern A/migrated for its success path):

- `compartment/processor.go:1467` `Accept`'s failure branch
  (`AcceptCommandFailed`) — a rejection reflecting no committed state
  change (inner tx rolled back). Captured as a `rejectEmit func() error`
  closure and fired via `producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvEventTopicStatus)(...)`
  on the direct path, then `Accept` returns `nil` (unchanged contract).
- `compartment/processor.go:1580` `Release`'s failure branch
  (`ReleaseCommandFailed`) — same shape as `Accept` above.
- `compartment/processor.go:1093` `AttemptEquipmentPickUp`'s failure branch
  — `dropProcessor.CancelReservation`, a cross-service COMMAND to
  atlas-drop reflecting a failed/rolled-back pickup. Fired via
  `message.Emit(producer.ProviderImpl(p.l)(p.ctx))(...)` on a fresh,
  throwaway buffer, then `AttemptEquipmentPickUp` returns `nil` (unchanged
  contract). The success path (`RequestPickUp`, bundled with the committed
  `CreateFromModel` write) is untouched.
- `compartment/processor.go:1167` `AttemptItemPickUp`'s failure branch —
  same `CancelReservation` fix as `AttemptEquipmentPickUp` above. The
  success path (pickup-consume command + `RequestPickUp`, bundled with the
  committed `CreateAsset`/`UpdateQuantity` writes) is untouched. See the
  residual-nuance Note below re: `CreateAsset`'s own `CreationFailedEventStatusProvider`
  when reached through this call path.

### Notes

- **Struct-init stored provider (`compartment/processor.go:97,109`,
  `p.producer producer.Provider`)**: traced every method that read
  `p.producer`. Fourteen `message.Emit(p.producer)` call sites existed
  before this migration; twelve write to the DB and were restructured to
  Pattern A (they no longer read `p.producer` — they build
  `outbox.EmitProvider(p.l, p.ctx, tx)` fresh inside their own
  `ExecuteTransaction`). The remaining two (`RequestReserveAndEmit`,
  `CancelReservationAndEmit`, listed above under Left Direct) do no DB
  write, so they keep reading the struct-bound `p.producer` field
  unchanged — matching the recipe's carve-out ("Methods on that struct
  whose flow does NO DB write may keep the direct provider"). The field
  itself, its initialization at `NewProcessor`, and its propagation through
  `WithTransaction`/`WithAssetProcessor` were left untouched since it is
  still live for those two methods.
- **Bundled drop/pickup commands — split by success/failure (D7 review-fix
  pass)**: `AttemptEquipmentPickUpAndEmit` and `AttemptItemPickUpAndEmit`
  each emit, in addition to their compartment/asset state-change events,
  `COMMAND_TOPIC_DROP` commands (`CancelReservation`/`RequestPickUp` via
  `drop.Processor`) and, on the consume-on-pickup branch (no DB write at
  all — out of scope for this pass, see below),
  `pickupMsg.EnvCommandTopic`. The original Task 11 migration bundled
  *all* of these onto the same migrated `mb`, reasoning by analogy to the
  `character/processor.go` Task 9 `AwardExperienceAndEmit` precedent
  (a call site's own bundled follow-on effect migrates with the rest of
  the buffer). A subsequent D7 review found that analogy doesn't hold for
  the FAILURE branch: `RequestPickUp` (success, bundled with the committed
  write) is a legitimate follow-on effect of a state change that actually
  happened, but `CancelReservation` (failure) is a COMMAND fired *because*
  the write did NOT happen — D7 explicitly lists "COMMAND emits to OTHER
  services" as leave-direct, and this command exists specifically to
  signal a rollback, so it must not wait on (or ride along with) an
  outbox flush that has nothing to commit. Fixed: `CancelReservation` now
  fires via a fresh, throwaway buffer on the direct producer path (see
  "Added by the D7 review-fix pass" above); `RequestPickUp` (success) and
  the consume-on-pickup branch's `pickupMsg` command are untouched.
  `TestAttemptItemPickUpInventoryFull` was updated to assert
  `CancelReservation`'s *absence* from the outbox-bound `mb` instead of
  its presence; `TestAttemptItemPickUpConsumeOnPickup` (success/no-write
  branch) needed no change.
- **Residual nuance, FIXED in a follow-up review pass (Task 11 code
  review, 2026-07-02)**: `CreateAsset`'s own
  `CreationFailedEventStatusProvider` rejection is a genuine no-op when
  reached via its direct entry point (`CreateAssetAndEmit` →
  `CreateAssetAndLock` → `CreateAsset`, see the corrected Migrated-list
  note above) because `CreateAsset` returns the error non-nil, which
  propagates all the way to `CreateAssetAndEmit`'s `Emit` call and
  discards the whole buffer. `CreateAsset` is *also* called from inside
  `AttemptItemPickUp`'s own tx closure (the merge/overflow and
  fresh-create branches), which did not propagate that error the same
  way: `AttemptItemPickUp` catches it and returns `nil` from its failure
  branch (unchanged contract) — previously that meant `CreateAsset`'s
  `CreationFailedEventStatusProvider` (and any other event optimistically
  buffered by the rolled-back inner tx before the failure, e.g. a
  same-branch `UpdateQuantity` Put) stayed sitting in the outbox-bound
  `mb` and flushed via the outbox on the swallowed-nil return — the exact
  D7 "rejection event reflecting no state change" pattern.
  **Fix**: `AttemptItemPickUp` now writes its inner tx's events to a local
  scratch buffer (`innerMb`, `compartment/processor.go`) instead of the
  caller-supplied `mb` directly. On success, `innerMb`'s contents are
  folded into `mb` before `RequestPickUp` (success path unchanged — same
  events end up in the same outbox-bound buffer). On failure, `innerMb`
  is **not** folded into `mb`; instead its contents ride along with
  `CancelReservation` on the existing DIRECT producer path (a fresh,
  throwaway buffer via `producer.ProviderImpl`), since atlas-channel's
  `handleCompartmentCreationFailedEvent` consumer
  (`services/atlas-channel/.../kafka/consumer/compartment/consumer.go`)
  needs `CREATION_FAILED` to reach the client to render the
  "inventory full" status message — it is not safe to simply drop the
  event. This preserves the caller-visible contract (`AttemptItemPickUp`
  still returns `nil` on a handled pickup failure) with no restructuring
  of `CreateAsset` or its other callers, and does not touch the
  still-latent `ExecuteTransaction`-no-op atomicity gap (task-119).
  `TestAttemptItemPickUpInventoryFull` was updated to assert
  `CREATION_FAILED`'s *absence* from `mb` (mirroring the existing
  `CancelReservation` absence assertion) instead of its presence with the
  correct error code. A new `TestAttemptItemPickUpSuccess` regression test
  guards the merge-on-success side: the inner `CreateAsset`'s `CREATED`
  asset event must still land in the outbox-bound `mb` alongside
  `REQUEST_PICK_UP`.
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see the `atlas-character` section above and project
  memory `bug_execute_transaction_noop.md`); this task's migrations use the
  correct seam and become atomic for free once task-119 lands.
- No pre-existing test asserted DIRECT-path (non-outbox) emission for any
  now-migrated flow in this module as of the original Task 11 commit — all
  `*AndEmit` wrapper methods were untested in isolation; every existing
  test exercised the un-wrapped `Method(mb)(...)` form directly against a
  manually constructed `message.Buffer`.
- **D7 review-fix pass test changes**: `TestAttemptItemPickUpInventoryFull`
  (`compartment/processor_test.go`) previously asserted that the
  `CancelReservation` drop command landed in the *same* buffer as the
  `CREATION_FAILED` compartment event; updated to assert its *absence*
  instead (it now fires via the direct producer, invisible to this test's
  buffer). Two new tests, `TestAcceptCommandFailedRoutesDirect` and
  `TestReleaseCommandFailedRoutesDirect`, were added to cover `Accept`/
  `Release`'s failure-path rejections, which had no prior test coverage.
  `TestMain` now calls `producertest.InstallNoop()` (from
  `github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest`)
  so these failure-path direct-producer calls succeed instantly in tests
  instead of retrying against an unreachable broker for ~42s.
  `TestAttemptItemPickUpConsumeOnPickup` (success/no-write branch)
  required no change. `go test -race ./...` — all packages `ok`.
