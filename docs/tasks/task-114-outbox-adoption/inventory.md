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
- **Further scope fix (2026-07-02, same day)**: the fix above still forwarded
  `innerMb` *wholesale* on failure (looping over `innerMb.GetAll()` and
  `Put`-ing every topic's messages onto the direct-path buffer). In the
  split-overflow branch this meant a buffered *success* event —
  `UpdateQuantity`'s `QuantityChanged` (asset topic), fired when the
  existing stack is topped up to `slotMax` — rode along with
  `CREATION_FAILED` on the direct path even though that write belongs to
  the same rolled-back inner tx. Harmless today only because
  `ExecuteTransaction` is the task-119 no-op (the quantity update is not
  actually undone), but wrong in principle and wrong for real once task-119
  lands. **Fix**: `AttemptItemPickUp`'s failure branch now filters
  `innerMb` down to just the `compartment.EnvEventTopicStatus` topic before
  forwarding — `CreateAsset` buffers `CreationFailedEventStatusProvider`
  under that exact topic only when the creation step itself is the one
  that fails, so the filter reproduces the intended semantic exactly:
  `CREATION_FAILED` still reaches atlas-channel on a creation failure, and
  any other buffered writes (asset-topic success events from steps that
  ran before the failing step) are discarded, never forwarded. Failures
  upstream of `CreateAsset` (e.g. `GetByCharacterAndType`) never populate
  that topic in `innerMb`, so they still forward nothing but the
  `CancelReservation` command, matching the pre-existing semantic. A new
  test, `TestAttemptItemPickUpSplitOverflowThenFail`
  (`compartment/processor_test.go`), reproduces the split-then-fail
  scenario (capacity-1 compartment, existing stack topped to `slotMax`,
  remainder create fails on no free slot) using a capturing producer
  writer (`installCapturingProducer`, swaps the process-wide manager
  singleton in place of `producertest.InstallNoop()` for the duration of
  the test) to assert the direct path carries `CREATION_FAILED` and
  `CANCEL_RESERVATION` but *not* `QuantityChanged`, while the outbox-bound
  `mb` stays empty. `TestAttemptItemPickUpInventoryFull` and
  `TestAttemptItemPickUpSuccess` continue to pass unchanged.
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

## atlas-cashshop

Module: `services/atlas-cashshop/atlas.com/cashshop`. Task 12. Line numbers
below reflect the module as of this commit.

### Migrated

Pattern A/B sites now enqueue their success-path event(s) via
`message.Emit`/`message.EmitWithResult` wired to
`outbox.EmitProvider(p.l, p.ctx, tx)` inside the same
`database.ExecuteTransaction` closure that performs the DB write.

- `wallet/processor.go:91` `CreateAndEmit` — Pattern B.
- `wallet/processor.go:115` `UpdateAndEmit` — Pattern B.
- `wallet/processor.go:141` `UpdateAndEmitWithTransaction` — Pattern B.
- `wallet/processor.go:205` `DeleteAndEmit` — Pattern A (`Delete` was a bare
  `deleteEntity` write with no existing `ExecuteTransaction`; now wrapped).
- `wishlist/processor.go:71` `AddAndEmit` — Pattern B (no `WithTransaction`
  on this processor; rebuilt via `NewProcessor(p.l, p.ctx, tx)`).
- `wishlist/processor.go:90` `DeleteAndEmit` — Pattern A, `NewProcessor`
  fallback.
- `wishlist/processor.go:107` `DeleteAllAndEmit` — Pattern A, `NewProcessor`
  fallback.
- `cashshop/inventory/asset/processor.go:119` `CreateAndEmit` — Pattern A,
  `NewProcessor` fallback (no `WithTransaction` on this processor).
- `cashshop/inventory/asset/processor.go:173`
  `CreateWithCashIdAndEmit` — Pattern A, `NewProcessor` fallback.
- `cashshop/inventory/asset/processor.go:246` `ExpireAndEmit` — Pattern A,
  `NewProcessor` fallback. `Expire`'s bare `deleteById` (previously
  unwrapped) plus its conditional replacement `Create` call are now both
  inside the enclosing tx.
- `cashshop/inventory/compartment/processor.go:260` `AcceptAndEmit` —
  Pattern A, uses the processor's existing `WithTransaction(tx)`.
- `cashshop/inventory/compartment/processor.go:298` `ReleaseAndEmit` —
  Pattern A, uses the existing `WithTransaction(tx)`.
- `cashshop/inventory/compartment/processor.go:131` `CreateAndEmit` —
  **fix-pass migration** (was routed through a hand-rolled
  `mb := message.NewBuffer(); ...; for t, ms := range mb.GetAll() { p.p(t)(...) }`
  flush loop over the struct-stored `p.p producer.Provider` field, not
  `message.Emit`, so it evaded the recipe's enumeration grep on the initial
  pass). Now Pattern A: `database.ExecuteTransaction` builds `tx`,
  `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))` wraps
  `p.WithTransaction(tx).Create(buf)(...)`, replacing the hand-rolled loop
  entirely (not left alongside it).
- `cashshop/inventory/compartment/processor.go:161` `UpdateCapacityAndEmit`
  — same fix-pass treatment: hand-rolled flush loop → Pattern A around
  `UpdateCapacity`.
- `cashshop/inventory/compartment/processor.go:194` `DeleteAndEmit` — same
  fix-pass treatment: hand-rolled flush loop → Pattern A around `Delete`.
- `cashshop/inventory/compartment/processor.go:230`
  `DeleteAllByAccountIdAndEmit` — same fix-pass treatment, plus two
  additional fixes to the wrapped `DeleteAllByAccountId` (line 202): (1)
  swapped its raw `p.db.WithContext(p.ctx).Transaction(...)` for
  `database.ExecuteTransaction(p.db.WithContext(p.ctx), ...)`, consistent
  with the rest of the module and safely re-entrant when
  `DeleteAllByAccountIdAndEmit`'s outer `ExecuteTransaction` already holds
  `tx`; (2) fixed the pre-existing bug where the inner per-compartment
  `deleteEntity(p.db.WithContext(p.ctx), ccm.Id())` call ignored the
  closure's own `tx` parameter in favor of the outer `p.db` — now
  `deleteEntity(tx.WithContext(p.ctx), ccm.Id())`, threading the actual
  transaction handle through.
- `cashshop/inventory/processor.go:150` `CreateAndEmit` — same fix-pass
  treatment. `createDefaultCompartments` (helper called from `Create`) was
  rewritten to take the shared `mb *message.Buffer` and call the
  compartment sub-processor's buffered `Create(mb)(...)` three times
  (Explorer/Cygnus/Legend) instead of its previous `CreateAndEmit(...)`
  (which opened its own independent `ExecuteTransaction`/outbox-provider
  bind per call); all three compartment writes plus the inventory-level
  `CreateStatusEventProvider` now share the single outer `tx` and the
  single outer `mb`/outbox flush.
- `cashshop/inventory/processor.go:183` `DeleteAndEmit` — same fix-pass
  treatment. `Delete` now calls the compartment sub-processor's buffered
  `DeleteAllByAccountId(mb)(accountId)` instead of
  `DeleteAllByAccountIdAndEmit(accountId)`, so the per-compartment deletes
  and the inventory-level `DeleteStatusEventProvider` share the one outer
  `tx`/`mb`.
- `cashshop/processor.go:78` `PurchaseAndEmit` — Pattern A, `NewProcessor`
  fallback (top-level `ProcessorImpl` has no `WithTransaction`). See Notes
  for the `INVENTORY_FULL` failure-path split inside `Purchase`.
- `cashshop/processor.go:199` `PurchaseInventoryIncreaseByItemAndEmit` /
  `cashshop/processor.go:210` `PurchaseInventoryIncreaseByTypeAndEmit` —
  Pattern A, `NewProcessor` fallback. Also reclassified the
  `InventoryCapacityIncreasedStatusEventProvider` emit inside
  `PurchaseInventoryIncrease` (originally a bare direct `producer.ProviderImpl`
  call immediately after a successful `ExecuteTransaction`) from
  left-direct to migrated per D7 ("immediately after" a committed tx = a
  state-asserting event): it is now `mb.Put` inside the tx closure and
  flows through the same outbox provider as the wallet/capacity writes it
  reports on.

### Left direct

- `cashshop/processor.go` (`Purchase`'s `INVENTORY_FULL` branch) — rejection
  event, no state change committed (the check runs before any write in the
  transaction). **Failure-path pitfall #1 applied**: the branch previously
  did `mb.Put(...); return nil`, which would have leaked the rejection into
  the outbox once `PurchaseAndEmit` was wrapped in Pattern A. Fixed by
  capturing a `rejectEmit func() error` closure (fires
  `producer.ProviderImpl(p.l)(p.ctx)(...)` directly) set inside the tx
  closure, returning the new internal sentinel `errPurchaseRejected` to
  abort the closure, then checking `rejectEmit != nil` *before* `txErr` in
  `Purchase`'s post-tx logic — fires the rejection on the direct path and
  returns `nil`, preserving the original external contract (this branch
  never returned a Go error to callers). All other `mb.Put` + `return err`
  branches in `Purchase` (UNKNOWN_ERROR ×5, NOT_ENOUGH_CASH) return a
  non-nil error today, so `message.Emit`'s `f(b)` error short-circuit
  already discards their buffered event before any flush — behavior is
  identical pre/post migration for those branches (pre-existing: those
  status events never actually publish today; not a regression introduced
  here, out of scope to fix).
- `cashshop/processor.go` (`PurchaseInventoryIncrease`'s `UNKNOWN_ERROR`
  branch, fired via `producer.ProviderImpl` right after `txErr != nil`) —
  rejection with no committed state change (the tx rolled back / never
  wrote); already lives outside the `ExecuteTransaction` closure, matching
  the recipe's prescribed remedy shape exactly, so no restructuring needed
  beyond adding an explanatory comment.
- `cashshop/inventory/asset/processor.go:198` `DeleteAndEmit` — wrapped
  method `Delete(_ *message.Buffer)` discards its `mb` argument and never
  calls `Put`; the `message.Emit` around it always flushes an empty buffer
  today. No event to migrate.
- `cashshop/inventory/asset/processor.go:211` `ReleaseAndEmit` — same
  reason; `Release(_ *message.Buffer)` also discards `mb`.
- `kafka/consumer/cashshop/consumer.go:93,102,111` — `RequestStorageIncrease`
  / `RequestStorageIncreaseByItem` / `RequestCharacterSlotIncreaseByItem`
  command handlers are unconditional "not implemented" stubs: they always
  fire `ErrorStatusEventProvider(..., "UNKNOWN_ERROR")` with no DB access
  at all. No state change, no tx to enqueue into.

### Notes

- The brief's `cashshop/processor.go:234,239` (pre-edit line numbers) were
  reviewed independently against D7 rather than both being taken as
  left-direct: `:234` (`UNKNOWN_ERROR` after `txErr != nil`) is a rejection
  with no state change → left direct; `:239`
  (`InventoryCapacityIncreasedStatusEventProvider`, fired unconditionally
  right after a successful tx) asserts a committed state change → migrated
  (see Migrated list). The brief flagged both for classification rather
  than dictating the outcome; this is the applied result.
- **Fix pass (resolves the prior "discovered gap" note)**: the 6 sites
  previously flagged as tx-coupled, state-asserting emits left unmigrated
  because they routed through a hand-rolled
  `mb := message.NewBuffer(); ...; for t, ms := range mb.GetAll() { p.p(t)(...) }`
  flush loop over the struct-stored `producer.Provider` field (evading the
  recipe's enumeration grep) have now been migrated to Pattern A — see the
  Migrated list above (`cashshop/inventory/processor.go:150,183` and
  `cashshop/inventory/compartment/processor.go:131,161,194,230`). The
  hand-rolled loops were replaced outright, not left alongside the outbox
  path. Per the recipe's "struct-init stored providers" guidance, the
  struct-stored `p` field on both `compartment.ProcessorImpl` and
  `inventory.ProcessorImpl` is no longer read by any of these methods —
  each now builds its own `outbox.EmitProvider(p.l, p.ctx, tx)` from the
  tx it owns.
- `cashshop/inventory/compartment/processor.go`'s `WithTransaction(tx)`
  previously copied `astP` (the nested `asset.Processor`) unchanged — it
  did not rebind the sub-processor to `tx`, so `Accept`/`Release`'s calls
  into `p.astP.CreateWithCashId`/`Release` opened their own independent
  `ExecuteTransaction` against the asset processor's original
  construction-time `db`, not the outer `tx` used for the
  compartment-level enqueue. **Fixed in this pass**: `WithTransaction` now
  rebinds via `astP: asset.NewProcessor(p.l, p.ctx, tx)` (matching
  `asset.NewProcessor`'s actual 3-arg constructor signature — the `asset`
  package exposes no `WithTransaction` method of its own), so
  `AcceptAndEmit`/`ReleaseAndEmit`'s asset write now joins the same `tx` as
  their compartment-level enqueue.
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see the `atlas-character` section above and project
  memory `bug_execute_transaction_noop.md`); this task's migrations use
  the correct seam and become atomic for free once task-119 lands.
- No pre-existing test in this module asserted direct-path (non-outbox)
  emission for any now-migrated flow — all touched `*AndEmit` methods were
  untested in isolation; the module's existing tests (`wallet/rest_test.go`,
  `wallet/provider_test.go`, `wallet/model_test.go`,
  `wishlist/rest_test.go`, `cashshop/inventory/rest_test.go`,
  `cashshop/inventory/asset/rest_test.go`,
  `cashshop/inventory/compartment/{rest,model}_test.go`,
  `cashshop/inventory/asset/reservation/cache_test.go`) exercise REST/model/
  provider layers only and do not reference `AndEmit`, `producer.`, or
  `message.Emit`. No test changes were required; `go test -race ./...` —
  all packages `ok`.

## atlas-fame

Module: `services/atlas-fame/atlas.com/fame`. Task 13. Line numbers below
reflect `fame/processor.go` and `character/processor.go` as of this commit.

### Migrated

- `fame/processor.go:166` `RequestChangeAndEmit` — Pattern A. Was a
  struct-init-shaped inversion problem (recipe's `:120` local-var-capture
  pitfall, pre-edit line numbers): `producerProvider :=
  producer.ProviderImpl(p.l)(p.ctx)` was captured into a local var
  *outside* any transaction, then handed to `message.Emit`, whose closure
  called the tx-opening `RequestChange(mb)(...)` — so the buffer flush rode
  the direct producer while the actual DB write (`create()`) happened
  inside `RequestChange`'s own, separate `database.ExecuteTransaction`. Per
  the recipe's guidance the fix moves the *whole* `message.Emit` inside a
  new outer `ExecuteTransaction` in `RequestChangeAndEmit` itself, built
  from `outbox.EmitProvider(p.l, p.ctx, tx)`, and rebinds via a new
  `WithTransaction(tx)` method (added to `Processor`) so
  `RequestChange`'s own nested `ExecuteTransaction` call re-enters the same
  tx (`database.ExecuteTransaction` is documented re-entrant). The
  resulting flow: validation reads, the `create()` fame-log write, and the
  follow-on `REQUEST_CHANGE_FAME` command to atlas-character
  (`character/processor.go:54` `RequestChangeFame`, called *without* `Emit`
  from inside `fame/processor.go:150`) now all commit and enqueue as one
  atomic unit via the outbox-bound `mb`.
- `fame/processor.go:77` `RequestChange` (the tx-opening building block
  wrapped by the above) — required an additional D7 fix beyond the Pattern
  A inversion itself: five early-return validation branches (character not
  found, target not found, below minimum level, already famed today,
  already famed this target this month) originally did
  `return mb.Put(EnvEventTopicFameStatus, errorEventStatusProvider(...))`
  — since `mb.Put` returns `nil` on success, these branches returned `nil`
  from the tx closure *before any write occurred*, which after the Pattern
  A inversion would have flushed the rejection event through the
  newly-outbox-bound `mb` — a textbook instance of the recipe's
  failure-path pitfall #1 (rejection event riding an empty, no-op commit
  into the outbox as if it reflected a state change). Fixed by restructuring
  around a `rejectEmit func() error` var (declared before the
  `ExecuteTransaction` call, matching the `atlas-cashshop`
  `Purchase`/`errPurchaseRejected` precedent): each validation branch now
  sets `rejectEmit` to a closure that fires
  `producer.ProviderImpl(p.l)(p.ctx)(...)` directly and returns the new
  package-level sentinel `errFameChangeRejected` to abort the tx; after
  `ExecuteTransaction` returns, `if rejectEmit != nil { rejectEmit();
  return nil }` fires the rejection on the direct path and swallows the
  sentinel (it never escapes `RequestChange`). The `create()`-write failure
  branch (`StatusEventErrorTypeUnexpected`, also no committed state) was
  converted the same way. The success branch (`create()` succeeds) is
  unchanged — it still returns
  `characterProcessor.RequestChangeFame(mb)(...)`, which buffers the
  command into the outbox-bound `mb`, since at that point a real write
  *did* commit.

### Left direct

- `character/processor.go:69` `RequestChangeFameAndEmit` — pure relay: the
  wrapped `RequestChangeFame` (`character/processor.go:54`) only builds a
  `REQUEST_CHANGE_FAME` command provider and `mb.Put`s it
  (`character/producer.go`'s `requestChangeFameCommandProvider`); it never
  touches this service's DB (`ProcessorImpl` here has no local write
  methods at all — `GetById`/`ByIdProvider` are read-only HTTP calls to
  atlas-character via `character/requests.go`). A COMMAND emit to another
  service with no local DB write, per the classification rule. Grepped for
  callers of `RequestChangeFameAndEmit` — none exist in this module besides
  the interface/mock declarations; the only live path to a
  `REQUEST_CHANGE_FAME` command is the already-migrated
  `fame/processor.go:150` call to the non-`Emit` `RequestChangeFame`
  variant, which shares the caller's (now outbox-bound) `mb`.
- `fame/processor.go:174` `DeleteByCharacterId` — bare
  `ExecuteTransaction` wrapping a `Delete`; no Kafka emit of any kind on
  this path (unchanged, nothing to migrate).

### Notes

- Enumeration: base grep (`message.Emit`, `producer.ProviderImpl`,
  `database.ExecuteTransaction`) found exactly the 3 sites named in the
  brief (`fame/processor.go:73` tx, `fame/processor.go:120-121` emit pre-edit
  → now `:166-171`, `fame/processor.go:127` tx pre-edit → now `:174-177`,
  `character/processor.go:70-71` emit). The extended enumeration grep
  (`.GetAll()`, `NewBuffer()`, `p.p(`, `producer.Provider\b`, `AndEmit(`)
  surfaced no additional hand-rolled flush loops or struct-stored-provider
  shapes — `ProcessorImpl` in both `fame` and `character` packages holds no
  provider field; every emit goes through `message.Emit`/`producer.ProviderImpl`
  directly. No hidden sites.
- Added `fame.Processor.WithTransaction(tx *gorm.DB) Processor` (new
  interface method + impl, mirrors the `atlas-monster-book`/`atlas-cashshop`
  pattern: `&ProcessorImpl{l: p.l, ctx: p.ctx, db: tx, t: p.t}`). No
  sub-processor fields exist on `fame.ProcessorImpl` to rebind — the
  `character.NewProcessor(p.l, p.ctx, tx)` call inside `RequestChange`
  already constructs a fresh, correctly-tx-bound instance per call (it was
  never a stored field), so no separate-transaction-write risk applies
  here.
- Added two tests to `fame/processor_test.go` (previously no
  `TestMain`/producer setup existed in this package) exercising the fixed
  failure path end-to-end via a `capturingWriter` (ported from the
  `atlas-inventory` test helper) installed over the process-wide producer
  singleton, plus an `httptest` stub for `CHARACTERS_SERVICE_URL`:
  - `TestProcessor_RequestChangeAndEmit_RejectsOnCharacterNotFound` —
    asserts (a) the rejection fires on the direct path (captured under
    `EnvEventTopicFameStatus`), (b) no fame log is created, (c)
    `outbox_entries` has 0 rows.
  - `TestProcessor_RequestChangeAndEmit_SuccessEnqueuesOutbox` — asserts
    (a) nothing fires on the direct path, (b) the fame log is created, (c)
    exactly one `outbox_entries` row exists with `Topic ==
    EnvCommandTopic` (the character command).
  Both are new tests (no pre-existing test in this module referenced
  `AndEmit`, `producer.`, or `message.Emit`, so nothing needed updating for
  the new outbox contract). `go test -race ./...`, `go vet ./...`, `go
  build ./...` all clean; `docker buildx bake atlas-fame` succeeded;
  `tools/redis-key-guard.sh` clean (this service doesn't touch Redis).
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see `bug_execute_transaction_noop.md`); this task's
  migration uses the correct seam and becomes atomic for free once
  task-119 lands.

## atlas-buddies

Module: `services/atlas-buddies/atlas.com/buddies`. All line numbers below
reflect `list/processor.go` as of the Task 14 commit.

### Migrated

- `list/processor.go:108` `DeleteAndEmit` → wraps `Delete` (Pattern A,
  clean — `Delete` already propagates `txErr` unswallowed on failure and
  puts no rejection event, so no D7 fix was needed here).
- `list/processor.go:143` `RequestAddBuddyAndEmit` → wraps `RequestAddBuddy`
  (Pattern A **+ D7 fix**, see Notes: the pure method previously logged
  `txErr` and returned `nil` — i.e. it *swallowed* the tx failure — so any
  rejection `ErrorStatusEventProvider` `mb.Put` during a rolled-back inner
  tx would otherwise flush into the same outbox-bound buffer the outer
  wrapper now enqueues on success. Restructured to a scratch `innerMb`:
  merged into the caller buffer on success, fired on the direct producer
  path on failure).
- `list/processor.go:276` `RequestDeleteBuddyAndEmit` → wraps
  `RequestDeleteBuddy` (Pattern A + same D7 scratch-buffer fix as above;
  also swallowed `txErr` to `nil` pre-migration).
- `list/processor.go:361` `AcceptInviteAndEmit` → wraps `AcceptInvite`
  (Pattern A + same D7 scratch-buffer fix; also swallowed `txErr` to `nil`
  pre-migration).
- `list/processor.go:471` `DeleteBuddyAndEmit` → wraps `DeleteBuddy`
  (Pattern A, clean — propagates `txErr`, puts no rejection event on
  failure).
- `list/processor.go:509` `UpdateBuddyChannelAndEmit` → wraps
  `UpdateBuddyChannel` (Pattern A, clean — same shape).
- `list/processor.go:547` `UpdateBuddyShopStatusAndEmit` → wraps
  `UpdateBuddyShopStatus` (Pattern A, clean — same shape).
- `list/processor.go:624` `IncreaseCapacityWithTransactionAndEmit` (and
  `:620` `IncreaseCapacityAndEmit`, which delegates straight to it with
  `transactionId = uuid.Nil`) → wraps `IncreaseCapacityWithTransaction`
  (Pattern A). This one DOES `mb.Put` a rejection event on each failure
  branch, but the pre-existing wrapper already propagated `txErr`
  unswallowed, and `message.Emit`'s closure only flushes the buffer when
  the wrapped function returns `nil` (see `kafka/message/message.go`) — so
  the rejection event was already silently discarded (never published,
  pre- and post-migration) whenever this method fails. That is a
  pre-existing behavior gap unrelated to D7 (nothing rides the outbox
  incorrectly); left as-is, out of this task's scope, and not a new
  regression.

### Left direct

- `list/resource.go:69` `handleCreateBuddyList` — a REST POST handler that
  emits a `CREATE` COMMAND (`list2.EnvCommandTopic`) to trigger buddy-list
  creation asynchronously via the `list2` Kafka consumer
  (`kafka/consumer/list/consumer.go`, which calls `list.Create`, a plain DB
  write with no emit of its own). The handler itself performs no DB write —
  per D7 (REST-handler emit with no local state change), left on the
  direct producer path.
- `invite/processor.go:32` `Create` — pure COMMAND emit
  (`invite2.EnvCommandTopic`) to the separate `atlas-invites` service (per
  the brief); `invite.ProcessorImpl` has no `db` field at all, so there is
  no DB write to be atomic with. Called synchronously (not via the shared
  `message.Buffer`) from `list/processor.go`'s `RequestAddBuddy`, inside
  the inner tx, but bypasses `mb`/`innerMb` entirely — already architecturally
  separate from the migrated buffer-flush path.
- `invite/processor.go:37` `Reject` — same shape as `Create` above, a
  COMMAND emit to `atlas-invites` with no local DB write, called from
  `list/processor.go`'s `RequestDeleteBuddy`.

### Notes

- **Enumeration.** Base grep (`message.Emit`, `producer.ProviderImpl`,
  `database.ExecuteTransaction`) found the 8 `AndEmit`/tx pairs in
  `list/processor.go` named in the brief, plus the two direct
  `producer.ProviderImpl` calls in `invite/processor.go` and the one in
  `list/resource.go:69`. The extended enumeration grep (`.GetAll()`,
  `NewBuffer()`, `p.p(`, `producer.Provider\b`, `AndEmit(`) surfaced the
  `list/processor.go:74` (pre-edit line) struct-init stored provider
  (`p producer.Provider`, set to `producer.ProviderImpl(l)(ctx)` in
  `NewProcessor` and copied unchanged in `WithTransaction`) referenced by
  all 8 `AndEmit` wrappers via `message.Emit(p.p)(...)` — i.e. exactly the
  "struct-init stored provider" case flagged in the brief, but every one of
  its 8 call sites was already covered by the base grep (all 8 are
  `message.Emit(p.p)(...)`, not a hand-rolled per-topic flush loop). No
  additional hidden hand-rolled flush loop was found. Read every
  `*AndEmit` method and its wrapped pure method in full before migrating.
- **`list/processor.go:74` struct-init stored provider.** Per the recipe's
  "Struct-init stored providers" guidance, the `p producer.Provider` field
  was removed entirely from `ProcessorImpl` (struct definition, `NewProcessor`,
  and `WithTransaction`) rather than kept — after migrating all 8 call
  sites off `message.Emit(p.p)(...)` to `message.Emit(outbox.EmitProvider(...))`
  built fresh from `tx` per the recipe, the field had zero remaining
  readers; a lingering unused-but-still-populated field would have been
  misleading (looks live, isn't). The three D7-fix reject-emit closures use
  `producer.ProviderImpl(p.l)(p.ctx)` directly instead (matches the
  pre-existing convention in `invite/processor.go` and `list/resource.go`).
- **`list/resource.go:69` classification.** `handleCreateBuddyList` is a
  REST POST handler (not GET, despite the brief's generic "REST GET
  handler" phrasing — read the actual method and confirmed it's a POST).
  It performs no DB write of its own (`producer.ProviderImpl(...)(list2.EnvCommandTopic)(list3.CreateCommandProvider(...))`
  is the entire handler body besides status-code plumbing); the actual DB
  write (`list.create`) happens later, in the `list2` consumer, on a
  separate request/transaction with no emit of its own. Left direct per
  D7 (no local state change to be atomic with).
- **Failure-path pitfall (3 sites).** `RequestAddBuddy`, `RequestDeleteBuddy`,
  and `AcceptInvite` all had the exact "swallow txErr to nil" shape flagged
  in the recipe: the pure method's outer `if txErr != nil { p.l.Errorf(...); return nil }`
  discarded the tx failure and returned success to the caller — meaning
  the pre-migration behavior already always flushed whatever was in `mb`
  (rejection event on failure, success event(s) on success) via the direct
  producer, since `message.Emit`'s wrapping closure only sees the
  swallowed `nil`. Migrating naively (Pattern A with no other change) would
  have made the OUTER (new, outbox-bound) `ExecuteTransaction`+`message.Emit`
  wrapper see that same swallowed `nil` and commit the rejection event to
  the outbox as if it reflected a committed state change — a direct D7
  violation. Fixed identically in all three methods: introduced a scratch
  `innerMb := message.NewBuffer()`, redirected every `mb.Put` inside the
  method to `innerMb.Put`, and after the (pre-existing) inner
  `ExecuteTransaction` returns: on `txErr != nil`, fire `innerMb`'s
  contents on the direct producer path
  (`message.Emit(producer.ProviderImpl(p.l)(p.ctx))(...)`) and return
  `nil`; on success, merge `innerMb.GetAll()` into the caller-supplied `mb`
  via `mb.Put(t, model.FixedProvider(ms))` per topic. Verified each single
  invocation of these methods puts either only-rejection or only-success
  events (no interleaving within one call — every failure branch is an
  immediate `mb.Put`+`return`), so no cross-branch leakage is possible.
  Pattern ported from the `atlas-inventory` D7 fix (commit `b820a3db7`).
- **Rebind nested sub-processors.** `list.ProcessorImpl` holds two
  sub-processor fields, `cp character.Processor` and `ip invite.Processor`,
  set in `NewProcessor` but — pre-migration — **not copied** in
  `WithTransaction` (silently dropped to their zero value, `nil`, on any
  `p.WithTransaction(tx)`-derived instance). This was latent/dormant
  pre-migration because every existing internal call site invoked `p.cp`/
  `p.ip` on the ORIGINAL (non-`WithTransaction`'d) receiver. This task's
  migration introduces the first `p.WithTransaction(tx).RequestAddBuddy(...)`
  /`.RequestDeleteBuddy(...)`/`.AcceptInvite(...)` call from each new
  `*AndEmit` wrapper — inside those methods, `p.cp.GetById(...)` and
  `p.ip.Create(...)`/`p.ip.Reject(...)` are now called on the
  `WithTransaction`-derived receiver, which would have nil-pointer-panicked
  without a fix. Fixed by adding `cp: p.cp, ip: p.ip` to `WithTransaction`'s
  return value (neither sub-processor has a `db` field to further rebind to
  `tx` — `character.Processor` is REST-only, `invite.Processor` is a pure
  command-emitter — so a plain field copy, not a tx-bound reconstruction,
  is correct here).
- **Tests.** No pre-existing test in this module exercised any `AndEmit`
  method or the pure buffer-taking methods directly (`list/processor_test.go`
  only exercises the `updateCapacity` administrator function against a raw
  DB, explicitly avoiding "processor tenant context complexity" per its own
  comment) — nothing needed updating for the new outbox contract. Added
  `list/processor_outbox_test.go` (new file, `package list`) with a
  `TestMain` installing `producertest.InstallNoop()` and a
  `capturingWriter`/`installCapturingProducer` helper (ported from the
  `atlas-inventory` D7-fix test pattern) to guard the D7 fix:
  - `TestRequestDeleteBuddyMissingListRoutesRejectDirect` — no buddy list
    exists for the character (fast, deterministic, DB-only failure, no
    HTTP mocking needed); asserts (a) the caller-supplied buffer is empty
    (the rejection did NOT ride the would-be-outbox path), (b) exactly one
    `ERROR` status event was captured on the DIRECT producer path.
  - `TestRequestDeleteBuddySuccessMergesIntoCallerBuffer` — full success
    path (character + target both have list rows, target is a buddy);
    asserts (a) exactly one `BUDDY_REMOVED` event lands in the
    caller-supplied buffer, (b) nothing fires on the direct path. Also
    surfaced and fixed a pre-existing test-infra gap: `setupProcessorTestDB`'s
    raw-SQL `buddies` table (in `list/processor_test.go`) predates
    `buddy.Entity.TenantId` and has no `tenant_id` column, which silently
    breaks tenant-scoped `Preload("Buddies")` reads for any test that
    inserts a buddy row with a real tenant in context; worked around
    locally in the new test via `db.AutoMigrate(&buddy.Entity{})` (adds the
    missing column without touching other tests' schema).
  - `TestAcceptInviteMissingListRoutesRejectDirect` — same shape as the
    `RequestDeleteBuddy` failure test, guarding `AcceptInvite`'s identical
    fix.
  `RequestAddBuddy`'s own failure branches were verified by code
  inspection only (not a dedicated test): its first failure branch
  requires `character.Processor.GetById` — a real HTTP call with no mock
  seam in this package — to fail, which happens deterministically in this
  sandboxed test environment (no `atlas-character` reachable) but isn't a
  hermetic/portable test fixture; the fix is structurally identical
  (byte-for-byte the same scratch-buffer pattern) to the two tested
  methods, and `go test -race`/`go vet`/`go build` all pass with the fix in
  place.
- `go test -race ./...`, `go vet ./...`, `go build ./...` all clean;
  `docker buildx bake atlas-buddies` succeeded; `tools/redis-key-guard.sh`
  clean (this service doesn't touch Redis).
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see `bug_execute_transaction_noop.md`); this task's
  migration uses the correct seam and becomes atomic for free once
  task-119 lands.

## atlas-guilds

Module: `services/atlas-guilds/atlas.com/guilds`. Two processors migrated:
`guild/processor.go` (13 `*AndEmit` sites) and `thread/processor.go` (5
`*AndEmit` sites). Line numbers reflect the state after this task's commit.

### Migrated

`guild/processor.go`:

- `guild/processor.go:298` `CreateAndEmit` — Pattern A. `Create` already
  wrapped its own `database.ExecuteTransaction` (guild + `member.AddMember`
  + `title.CreateDefaults`) and put the success event via `mb.Put` AFTER
  that inner tx returned but BEFORE the outer wrapper flushed — the outer
  wrapper previously used the direct producer. Now `CreateAndEmit` opens the
  outer tx + `outbox.EmitProvider(tx)` and calls
  `p.WithTransaction(tx).Create(mb)(...)`; the inner nested
  `ExecuteTransaction` re-enters the same (currently no-op) seam per the
  recipe's re-entrancy note, and the success `mb.Put` now lands inside the
  outbox-bound closure.
- `guild/processor.go:361` `CreationAgreementResponseAndEmit` — Pattern A.
  Calls `p.Create(mb)(...)` internally (DB write) and, for non-leader
  requesters, `member.NewProcessor(p.l, p.ctx, p.db).AddMember(...)` (a
  second, previously untethered DB write) before putting the created event.
  No sub-processor struct fields exist on `guild.ProcessorImpl` — every
  internal call constructs `member.NewProcessor`/`title.NewProcessor`/
  `character.NewProcessor` fresh from `p.db` per call, so invoking this
  method via `p.WithTransaction(tx).CreationAgreementResponse(mb)(...)`
  correctly rebinds every one of those fresh constructions to the shared
  `tx` (see "Sub-processor rebind" note below) with zero body edits.
- `guild/processor.go:393` `ChangeEmblemAndEmit` — Pattern A (bare
  `updateEmblem(p.db...)` write, not previously wrapped in any tx; now
  explicitly wrapped per the classification rule).
- `guild/processor.go:423` `UpdateMemberOnlineAndEmit` — Pattern A around an
  already-tx-wrapped inner method (`UpdateMemberOnline` opens its own
  `ExecuteTransaction`); outer tx now supplies the outbox provider so the
  `member.UpdateStatus` write and the `MEMBER_STATUS` event share one tx.
- `guild/processor.go:449` `ChangeNoticeAndEmit` — Pattern A (bare
  `updateNotice(p.db...)` write, explicitly wrapped).
- `guild/processor.go:481` `LeaveAndEmit` — Pattern A (bare
  `member.NewProcessor(p.l,p.ctx,p.db).RemoveMember(...)` call at the guild
  layer — `RemoveMember` itself opens its own inner tx — now threaded
  through the outer tx via `WithTransaction`).
- `guild/processor.go:542` `JoinAndEmit` — Pattern A (bare
  `member.NewProcessor(...).AddMember(...)` call, same shape as Leave).
- `guild/processor.go:574` `ChangeTitlesAndEmit` — Pattern A (bare
  `title.NewProcessor(...).Replace(...)` call — `Replace` opens its own
  inner tx — now threaded through the outer tx).
- `guild/processor.go:609` `ChangeMemberTitleAndEmit` — Pattern A around an
  already-tx-wrapped inner method (`ChangeMemberTitle` opens its own
  `ExecuteTransaction` around `member.UpdateTitle`).
- `guild/processor.go:645` `RequestDisbandAndEmit` — Pattern A around an
  already-tx-wrapped inner method (`RequestDisband` opens its own
  `ExecuteTransaction` around the member-removal loop, `title.Clear`, and
  `deleteGuild`).
- `guild/processor.go:675` `RequestCapacityIncreaseAndEmit` — Pattern A
  (bare `updateCapacity(p.db...)` write, explicitly wrapped).

`thread/processor.go`:

- `thread/processor.go:107` `CreateAndEmit` — Pattern A (bare `create(p.db...)`
  write, not previously wrapped; now explicitly wrapped).
- `thread/processor.go:163` `UpdateAndEmit` — Pattern A around an
  already-tx-wrapped inner method (`Update` opens its own
  `ExecuteTransaction` around the thread row update).
- `thread/processor.go:219` `DeleteAndEmit` — Pattern A around an
  already-tx-wrapped inner method (`Delete` opens its own
  `ExecuteTransaction` covering the reply-cascade delete via
  `reply.NewProcessor(p.l,p.ctx,tx).Delete(...)` plus the thread row
  delete).
- `thread/processor.go:270` `ReplyAndEmit` — Pattern A around an
  already-tx-wrapped inner method (`Reply` opens its own
  `ExecuteTransaction` around `reply.NewProcessor(p.l,p.ctx,tx).Add(...)`).
- `thread/processor.go:324` `DeleteReplyAndEmit` — Pattern A around an
  already-tx-wrapped inner method (`DeleteReply` opens its own
  `ExecuteTransaction` around `reply.NewProcessor(p.l,p.ctx,tx).Delete(...)`).

### Left direct

- `guild/processor.go:226` `RequestCreateAndEmit` — no DB write in this
  flow at all: the method only queries (character/party HTTP reads),
  validates, and calls `coordinator.GetRegistry().Initiate(...)`, an
  in-process registry (not a DB write). Every `mb.Put` here (both the
  error/rejection events on each validation failure and the success
  `REQUEST_AGREEMENT` event) reflects no committed DB state, so per D7 the
  whole method stays on the direct producer path. Verified by reading the
  full method body — zero `p.db`/`tx` references anywhere in it.
- `guild/processor.go:510` `RequestInviteAndEmit` — the wrapped method,
  `RequestInvite`, takes its `mb` parameter as `_` (explicitly unused) and
  never calls `mb.Put`; the actual invite creation delegates to
  `invite.NewProcessor(p.l, p.ctx).Create(...)`, which emits its own
  COMMAND event directly (see next bullet). The `message.Emit(...)` wrapper
  around `RequestInvite` is therefore a no-op flush of an always-empty
  buffer — left unchanged (no behavior to migrate).
- `invite/processor.go:29` `Create` — a COMMAND emit to another service
  (`producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopic)(...)`), not a status
  event reflecting a local committed state change; `invite.ProcessorImpl`
  has no `db` field at all (it's a pure command-emitter, per the brief).
  Per D7, command emits to other services stay direct. Left unchanged.

### Notes

- **Enumeration.** Ran both recipe greps (`message.Emit|EmitWithResult|
  producer.ProviderImpl|database.ExecuteTransaction` and
  `.GetAll()|NewBuffer()|\bp\.p(|producer.Provider\b|AndEmit(`) from the
  module dir. No hand-rolled buffer-flush-over-stored-provider sites were
  found (`kafka/message/message.go`'s own `Emit`/`EmitWithResult`
  definitions and `guild/task.go:44` / the four `kafka/consumer/*`
  packages calling into the now-migrated `*AndEmit` methods are the only
  other grep hits — none are additional emit sites to migrate). Also read
  `thread/reply/processor.go`, `guild/title/processor.go`,
  `guild/member/processor.go`, and `guild/character/processor.go` in full:
  none of them independently emit Kafka messages — they are pure
  DB-write sub-processors invoked by the two migrated processors.
- **Sub-processor rebind.** Unlike the `atlas-cashshop`/`atlas-buddies`
  shape (a long-lived sub-processor field set once in `NewProcessor` and
  silently NOT copied by `WithTransaction`), `guild.ProcessorImpl` and
  `thread.ProcessorImpl` hold **no** sub-processor struct fields at all —
  every call site constructs `member.NewProcessor(p.l, p.ctx, p.db)` /
  `title.NewProcessor(...)` / `character.NewProcessor(...)` /
  `reply.NewProcessor(...)` fresh, inline, per call, always passing the
  CURRENT receiver's `p.db`. This means the correct rebind mechanism here
  is simply: every migrated `*AndEmit` wrapper must invoke the pure method
  on `p.WithTransaction(tx)`, never on the original `p`. All 16 migrated
  wrappers (11 in `guild`, 5 in `thread`) do this consistently, so every
  fresh sub-processor constructed inside the wrapped method inherits `tx`
  automatically via the receiver's `p.db` field — no struct-field changes
  were needed in either `ProcessorImpl`.
- **Failure-path pitfalls.** Audited every migrated method for the
  "rejection/command-only branch that `return nil`s after an `mb.Put`"
  shape described in the recipe. None exists in this service: every
  early-return-`nil` branch in both processors (e.g.
  `UpdateMemberOnline`'s `GetByMemberId` failure, `ChangeMemberTitle`'s
  `GetByMemberId` failure) occurs strictly BEFORE any `mb.Put` in that
  call, so nothing is queued that could leak to the wrong path. All other
  failure branches propagate the real error (`return err`) rather than
  swallowing it to `nil`, so no scratch-buffer/D7 restructuring was
  needed — a plain Pattern A wrap was sufficient everywhere.
- **Unused import cleanup.** `thread/processor.go` no longer references
  `atlas-guilds/kafka/producer` after all 5 sites migrated off
  `producer.ProviderImpl`; the import was removed. `guild/processor.go`
  still uses `producer.ProviderImpl` directly in the two left-direct sites
  (`RequestCreateAndEmit`, `RequestInviteAndEmit`), so its import was kept.
- **Tests.** No pre-existing test in `guild/processor_test.go` or
  `thread/processor_test.go` exercised any `*AndEmit` method (both files
  only cover read paths, builders, and `WithTransaction` identity), so
  nothing needed updating for the new outbox contract. Added
  `guild/processor_outbox_test.go` and `thread/processor_outbox_test.go`
  (new files, in-package `guild`/`thread`) covering: (1) a
  previously-bare-write site committing exactly one outbox row
  (`ChangeEmblemAndEmit` / `CreateAndEmit`), (2) an
  already-nested-tx site committing exactly one outbox row
  (`UpdateMemberOnlineAndEmit` / `ReplyAndEmit`), and (3) the generic
  rollback-discards-enqueued-events seam test using gorm's `db.Transaction`
  directly (per the recipe's no-op-`ExecuteTransaction` caveat), ported
  from `atlas-character/character/outbox_acceptance_test.go`.
- `go test -race ./...`, `go vet ./...`, `go build ./...` all clean;
  `docker buildx bake atlas-guilds` succeeded; `tools/redis-key-guard.sh`
  clean (this service doesn't touch Redis directly).
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see `bug_execute_transaction_noop.md`); this task's
  migration uses the correct seam and becomes atomic for free once
  task-119 lands.

## atlas-notes

Module: `services/atlas-notes/atlas.com/notes`. One processor migrated
(`note/processor.go`, all 5 `*AndEmit` sites); the four tx-coupled writes
live in `note/administrator.go`, which is re-entered through the shared
`database.ExecuteTransaction` seam rather than edited directly. Line
numbers reflect the state after this task's commit.

### Migrated

- `note/processor.go:99` `CreateAndEmit` — Pattern B. Opens
  `database.ExecuteTransaction`, builds `outbox.EmitProvider(p.l, p.ctx,
  tx)`, and calls `p.WithTransaction(tx).Create(mb)(...)`, which in turn
  calls `createNote(tx.WithContext(ctx), ...)` in `administrator.go:9`.
  `createNote` opens its own `database.ExecuteTransaction(db, ...)`
  internally; since `db` here is already the outer `tx`, `isTransaction`
  is true and the administrator's `tx.Create(&entity)` re-enters the same
  transaction (recipe re-entrancy guarantee) rather than opening a second
  one. The result (`Model`) is captured in a `var result Model` outside
  the `ExecuteTransaction` closure per Pattern B.
- `note/processor.go:145` `UpdateAndEmit` — Pattern B, same shape as
  Create. Calls `p.WithTransaction(tx).Update(mb)(...)` →
  `updateNote(tx.WithContext(ctx), ...)` in `administrator.go:23`, which
  re-enters the outer tx for its `tx.Where(...).Updates(&entity)` call
  (and then re-reads the row via `getByIdProvider` on the same `db`/`tx`
  handle before returning).
- `note/processor.go:177` `DeleteAndEmit` — Pattern A. Calls
  `p.WithTransaction(tx).Delete(mb)(id)` → `deleteNote(tx.WithContext(ctx),
  id)` in `administrator.go:40`, re-entering the outer tx.
- `note/processor.go:206` `DeleteAllAndEmit` — Pattern A. Calls
  `p.WithTransaction(tx).DeleteAll(mb)(characterId)` →
  `deleteAllNotes(tx.WithContext(ctx), characterId)` in
  `administrator.go:46`, re-entering the outer tx. `DeleteAll` buffers one
  `NOTE_STATUS` delete event per existing note (read via
  `ByCharacterProvider` before the bulk delete) — all buffered events flush
  through the single outbox provider built from the one outer `tx`.
- `note/processor.go:302` `DiscardAndEmit` — Pattern A (call-site-shaped
  closure, since `Discard` takes `mb` plus three more curried args). Wraps
  `p.WithTransaction(tx).Discard(mb)(ch)(characterId)(noteIds)` in the
  outer `ExecuteTransaction`; each loop iteration's `deleteNote(tx...)`
  call re-enters the same outer tx (same seam as `DeleteAndEmit`, just
  looped).

`ProcessorImpl` gained a `WithTransaction(tx *gorm.DB) Processor` method
(added to the `Processor` interface) that copies `l`/`ctx`/`t`/`sagaP` and
swaps `db` for the given `tx`. The struct's former `producer
producer.Provider` field (constructed once in `NewProcessor` via
`producer.ProviderImpl(l)(ctx)`) was the sole reason none of the five
`*AndEmit` sites previously threaded a transaction through their
administrator calls — every DB write in `Create`/`Update`/`Delete`/
`DeleteAll`/`Discard` used `p.db` directly with no tx-scoping, and the
buffered events flushed through that stored direct producer strictly
*after* each administrator write had already committed on its own.
Because all five emit sites are now migrated and nothing else in the
package reads a stored provider (the classification rule's "methods with
no DB write may keep the direct provider" carve-out doesn't apply to any
method here), the field was deleted outright rather than left dead,
along with its now-unused `atlas-notes/kafka/producer` import in
`processor.go`.

### Left direct

- `saga/processor.go:27` `Create` — `producer.ProviderImpl(p.l)(p.ctx)(msgsaga.EnvCommandTopic)(CreateCommandProvider(s))`
  is a COMMAND emit to the separate `atlas-saga-orchestrator` service (a
  saga-kickoff command, not a notes-domain status event) with no local DB
  write in this function — per the D7 classification rule, COMMAND emits
  to other services stay on the direct path. **Post-review fix (task-114
  review pass):** this call site is now reached only from
  `note/processor.go:302` `DiscardAndEmit`'s post-commit firing loop, not
  from inside the transaction. The original migration left
  `awardFameToSender` firing this command unconditionally, synchronously,
  from inside `Discard`'s per-note loop — but that loop now runs inside
  `DiscardAndEmit`'s single shared `ExecuteTransaction` closure (this
  task's own change), so a *later* note's failure in the same call could
  roll back an *earlier* note's delete after its fame-award command had
  already been irrevocably sent — a non-atomic side effect this migration
  introduced (it did not exist pre-migration, when each note's delete
  committed in its own immediately-committed mini-tx). Fixed by splitting
  `awardFameToSender` into a pure `buildFameAwardSaga` (called from inside
  `Discard`'s loop; only builds and collects a `pendingFameAward`, no
  side effect) and `fireFameAwardSaga` (called from `DiscardAndEmit`,
  once per collected item, only after `database.ExecuteTransaction`
  returns nil). `Discard`'s signature changed from returning `error` to
  returning `([]pendingFameAward, error)` so the collected-but-unfired
  sagas can cross the transaction-closure boundary. The command itself
  (topic, payload, one-per-successfully-discarded-non-self-non-system-note
  semantics) is unchanged — only *when* it fires moved from mid-loop to
  post-commit. Verified by
  `note/processor_fame_award_test.go`'s
  `TestDiscardAndEmit_FameAwardNotFiredWhenDiscardFails` (proves 0 saga
  commands fire when a later note id in the same call fails) and
  `TestDiscardAndEmit_FameAwardFiresAfterSuccess` (proves exactly one
  command per discarded note fires on the happy path); both tests were
  confirmed to fail against the pre-fix inline-fire behavior before being
  confirmed to pass against the fix.

### Notes

- Enumeration: the step-2 grep plus the extra
  `GetAll()`/`NewBuffer()`/`p.p(`/`producer.Provider`/`AndEmit(` sweep
  found exactly the 5 `note/processor.go` `*AndEmit` sites (3 `Emit`, 2
  `EmitWithResult`) plus the 1 `saga/processor.go` command emit called out
  in the brief — no hand-rolled buffer-flush loops or struct-stored
  provider fan-outs beyond the single `producer` field already accounted
  for above. `note/mock/processor.go`'s `ProcessorMock` does not
  implement the current `note.Processor` interface (its `Discard`/
  `DiscardAndEmit` signatures are missing the `ch channel.Model` param
  that the real interface has carried since before this task, and it now
  also lacks `WithTransaction`); it isn't referenced anywhere in the
  module or asserted against the interface, so it compiles as inert dead
  code both before and after this change — pre-existing drift, left
  untouched as out of scope for this task.
- No pre-existing test asserted direct-path emission for a migrated flow
  (`processor_test.go` only exercises the `mb`-taking `Create`/`Update`/
  `Delete`/`DeleteAll`/`Discard` methods directly, never the `*AndEmit`
  wrappers), so no test needed updating to assert outbox rows instead.
  **Post-review update:** `processor_test.go`'s two `Discard(...)` call
  sites were updated for the new `([]pendingFameAward, error)` return
  shape, and a new internal (`package note`) test file
  `note/processor_fame_award_test.go` was added specifically to exercise
  `DiscardAndEmit` — the first test in this module to call an `*AndEmit`
  method directly — using a `fakeSagaProcessor` injected via a
  package-internal `ProcessorImpl{...}` literal (the mock file at
  `note/mock/processor.go` couldn't be reused since it's the pre-existing
  stale/inert mock noted above). This required also migrating
  `libs/atlas-outbox`'s `Entity` table (`outbox.Migration(db)`) in the new
  file's test-db setup, which no prior test in this module needed.
- `go test -race ./...`, `go vet ./...`, `go build ./...` all clean;
  `docker buildx bake atlas-notes` succeeded; `tools/redis-key-guard.sh`
  clean (this service doesn't touch Redis directly). Re-verified clean
  after the post-review fame-award-ordering fix.
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see `bug_execute_transaction_noop.md`); this task's
  migration uses the correct seam and becomes atomic for free once
  task-119 lands.

## atlas-pets

Module: `services/atlas-pets/atlas.com/pets`. All line numbers below
reflect `pet/processor.go` as of the Task 17 commit.

### Migrated

Every `*AndEmit` site now wraps its own outer
`database.ExecuteTransaction(p.db.WithContext(p.ctx), ...)`, builds the
outbox provider from that outer `tx` via
`outbox.EmitProvider(p.l, p.ctx, tx)`, and calls the underlying
`mb`-taking method via `p.With(WithTransaction(tx))` so the method's own
internal `ExecuteTransaction` call re-enters the same outer transaction
(re-entrant per design). All 12 emit call sites in the brief (11
`message.Emit` + 1 `message.EmitWithResult`) were migrated; none were
left direct.

- `pet/processor.go:234` `CreateAndEmit` — Pattern B (`EmitWithResult`).
  Captures `result Model` outside the closure; `Create` itself already
  wraps its write + `EnvStatusEventTopic` `createdEventProvider` put in
  its own `ExecuteTransaction`, which now re-enters the outer tx.
- `pet/processor.go:283` `DeleteOnRemoveAndEmit` — Pattern A. Wraps
  `DeleteOnRemove`, which does pure reads (character/inventory lookup)
  then delegates to the already-migrated `Delete` write.
- `pet/processor.go:312` `DeleteForCharacterAndEmit` — Pattern A. Wraps
  `DeleteForCharacter`, which loops `GetByOwner` results and calls
  `p.With(WithTransaction(tx)).Delete(mb)(...)` per pet — all deletes and
  their `deletedEventProvider` events now share the one outer tx/buffer.
- `pet/processor.go:386` `SpawnAndEmit` — Pattern A. `Spawn` covers the
  egg-hatch-on-summon branch (in-place template swap + `ip.ChangeTemplate`
  command), the lead-migration slot-shift loop, and the final
  `spawnEventProvider` — all inside the one re-entrant tx/buffer.
- `pet/processor.go:541` `DespawnAndEmit` — Pattern A. Wraps
  `Despawn`/`defaultDespawn`'s slot-shift loop and `despawnEventProvider`.
- `pet/processor.go:622` `AttemptCommandAndEmit` — Pattern A. `AttemptCommand`
  calls `p.With(WithTransaction(tx)).AwardCloseness(mb)(...)` internally,
  which re-enters the same outer tx a second level deep — still one
  physical transaction/buffer.
- `pet/processor.go:683` `EvaluateHungerAndEmit` — Pattern A.
  `EvaluateHunger` loops spawned pets, updates fullness, and conditionally
  calls `p.With(WithTransaction(tx)).Despawn(mb)(...)` on hunger-triggered
  despawn — nested re-entrant call, same tx/buffer.
- `pet/processor.go:761` `AwardClosenessWithTransactionAndEmit` — Pattern A.
  (`AwardClosenessAndEmit` is a thin `uuid.Nil` wrapper around this and
  needed no separate change.) `AwardClosenessWithTransaction` calls
  `p.With(WithTransaction(tx)).AwardLevel(mb)(...)` internally on
  level-up — nested re-entrant call.
- `pet/processor.go:837` `EvolveAndEmit` — Pattern A. `Evolve` performs the
  template-roll write + `ip.ChangeTemplate` command +
  `evolvedEventProvider` inside its own re-entrant tx, then (outside that
  inner tx but still inside the outer one, since `p` here is already
  tx-rebound) best-effort calls `p.Despawn`/`p.Spawn` for the
  summoned-pet appearance refresh, swallowing their errors as warnings —
  unchanged control flow, now folded into the single outer transaction
  instead of three separately-committed ones (see Notes).
- `pet/processor.go:918` `AwardFullnessAndEmit` — Pattern A.
- `pet/processor.go:959` `AwardLevelAndEmit` — Pattern A.
- `pet/processor.go:1000` `SetExcludeAndEmit` — Pattern A.

`inv.Processor.ChangeTemplate` (called from `Spawn`'s egg-hatch branch and
from `Evolve`) buffers an `EnvCommandTopic` `CHANGE_TEMPLATE` command to
atlas-inventory into the same shared `mb` as the status event — this is
the method's own follow-on effect of the state change it just made (not a
cross-service command emitted independently of any local write), so per
the atlas-character precedent it migrates with the rest of the buffer
rather than being left direct.

### Left direct

None. `ClearPositions` (`pet/processor.go:737`) has its own
`ExecuteTransaction` (the 13th tx site counted in the brief) but performs
no Kafka emit at all — it only clears the Redis-backed temporal-position
registry — so there is nothing to migrate there.

### Notes

- **Struct-init stored provider (`:108`, pre-migration).** `ProcessorImpl`
  held a `kp producer.Provider` field constructed once in `NewProcessor`
  via `producer.ProviderImpl(l)(ctx)` and bound to the *original*
  construction-time context — every one of the 12 `*AndEmit` sites read
  through this single field. Since all 12 tx-coupled sites migrated and no
  method needed to keep a direct-producer flow (there's no method here
  whose flow does no DB write among the `*AndEmit` set), the field, its
  `NewProcessor` initializer, and the now-dead `atlas-pets/kafka/producer`
  import were deleted outright rather than left dead — same call as
  atlas-notes made for its equivalent stored field.
- Enumeration: the step-2 grep (`message.Emit`/`EmitWithResult`/
  `producer.ProviderImpl`/`database.ExecuteTransaction`) plus the extra
  `GetAll()`/`NewBuffer()`/`p.kp(`/`producer.Provider`/`AndEmit(` sweep
  both landed on exactly the 12 `*AndEmit` sites and 13
  `ExecuteTransaction` sites called out in the brief — no hand-rolled
  buffer-flush loop over `p.kp` and no additional stored-provider fan-out
  anywhere else in the module (`character`, `inventory`, `data/pet`,
  `data/position`, `skill` sub-processors are all REST-only clients with
  no local `db` field, so `WithTransaction`'s existing behavior of only
  swapping `p.db` needed no rebind of those fields).
- No rejection/error event topic exists in this service's message
  package (`kafka/message/pet` defines only `EnvStatusEventTopic` and
  `EnvCommandTopic`) — every `mb.Put` in every migrated method represents
  a genuine committed state change, so none of the failure-path
  pitfalls (rejection-then-`return nil`, or shared-buffer-forwarded-on-
  failure) apply here; no `rejectEmit` closures were needed anywhere in
  this migration.
- `EvolveAndEmit`'s post-evolve despawn/respawn appearance-refresh calls
  (`Evolve`'s `wasSummoned` branch) already swallowed despawn/spawn errors
  as warnings pre-migration and always returned `nil` from `Evolve`
  regardless. Post-migration those nested calls re-enter the *same*
  physical transaction as the main evolve write (previously each ran in
  its own separately-committed transaction), so a swallowed despawn/spawn
  failure now shares fate with the evolve commit instead of failing
  independently. This is an accepted side effect of the recipe's
  re-entrant-nested-tx design (same as every other migrated service where
  an `AndEmit` method calls into another already-tx-wrapped method) and
  not a new rejection-path bug — there is no separate rejection event on
  this branch, and the existing log-and-continue semantics are unchanged.
  No behavior change is user-visible today since `ExecuteTransaction` is
  still a no-op pending task-119.
- No batching-per-item-command concern (per the atlas-notes lesson):
  `DeleteForCharacter` and `EvaluateHunger` loop over pets and call
  `Delete`/`Despawn` per item, but neither buffers a *direct-path*
  side-effecting command outside `mb` — everything they emit goes through
  the shared buffer, which only flushes once, after the single outer
  `ExecuteTransaction` commits. There is no separate immediate-fire
  command needing post-commit collection here.
- No pre-existing test asserted direct-path emission for a migrated flow:
  every test in `pet/processor_test.go` calls the `mb`-taking methods
  (`Create`, `Delete`, `Spawn`, `Despawn`, `AttemptCommand`,
  `EvaluateHunger`, `AwardCloseness*`, `AwardFullness`, `AwardLevel`,
  `SetExclude`, `Evolve`) directly with an explicit `message.NewBuffer()`
  and asserts against the buffer contents — never the `*AndEmit`
  wrappers — so no test needed updating to assert outbox rows instead.
- `go test -race ./...`, `go vet ./...`, `go build ./...` all clean;
  `docker buildx bake atlas-pets` succeeded; `tools/redis-key-guard.sh`
  clean (exit 0 across the whole repo).
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see `bug_execute_transaction_noop.md`); this task's
  migration uses the correct seam and becomes atomic for free once
  task-119 lands.

## atlas-skills

Module: `services/atlas-skills/atlas.com/skills`. Line numbers below
reflect `skill/processor.go` and `macro/processor.go` as of the Task 18
commit.

### Migrated

- `skill/processor.go:140` `CreateAndEmit` — Pattern A. `Create` (mb-taking,
  `:109`) already wraps its existence-check + row insert in its own
  `ExecuteTransaction` (`:113`); `CreateAndEmit` now opens an outer
  `ExecuteTransaction`, builds `outbox.EmitProvider(p.l, p.ctx, tx)`, and
  calls `p.WithTransaction(tx).Create(buf)(...)` so the insert and the
  `EnvStatusEventTopic` `StatusEventTypeCreated` enqueue share one tx (the
  inner `ExecuteTransaction` re-enters the outer one, per the recipe's
  re-entrancy guarantee).
- `skill/processor.go:188` `UpdateAndEmit` — Pattern A, same shape as
  `CreateAndEmit`. `Update` (`:153`) wraps its existence-check + dynamic
  update in its own `ExecuteTransaction` (`:157`); `UpdateAndEmit` now
  wraps that in an outer tx and enqueues `StatusEventTypeUpdated` through
  `outbox.EmitProvider`.
- `skill/processor.go:301` `DeleteForSagaCompensationAndEmit` — Pattern A,
  with an added explicit transaction. Pre-migration, the underlying
  `DeleteForSagaCompensation` (`:280`) called `deleteSkill` directly
  against `p.db` with **no** `ExecuteTransaction` wrap at all (a bare
  write) and then buffered the DELETED event outside any tx. Per the
  classification rule ("wrap bare Save/Create/Update writes in an
  explicit ExecuteTransaction with the enqueue"), `DeleteForSagaCompensationAndEmit`
  now opens `database.ExecuteTransaction` itself and calls
  `p.WithTransaction(tx).DeleteForSagaCompensation(buf)(...)` inside it, so
  the delete (or idempotent no-op) and the `StatusEventTypeDeleted`/synthetic
  event enqueue commit atomically.
- `macro/processor.go:108` `UpdateAndEmit` — Pattern A. `Update` (`:69`)
  wraps its delete-then-recreate loop in its own `ExecuteTransaction`
  (`:73`); `UpdateAndEmit` now opens an outer `ExecuteTransaction` and
  enqueues `EnvStatusEventTopic` `StatusEventTypeUpdated` via
  `outbox.EmitProvider`. `macro.ProcessorImpl` had no `WithTransaction`
  method before this task (only `skill.ProcessorImpl` had one) — added a
  `WithTransaction(tx *gorm.DB) *ProcessorImpl` matching the skill
  package's shape (rebinds `db`, keeps `l`/`ctx`/`t`) so `Update` can run
  against the outer tx.

### Left direct

- `skill/processor.go:232` `ExpireCooldowns(l, ctx)` — package-level
  function invoked from `tasks/expiration.go:29` as a background ticker.
  Read in full: it iterates `GetRegistry().GetAll(ctx)` (a Redis-backed
  `atlas.TenantRegistry`, see `skill/cooldown_registry.go`), calls
  `GetRegistry().Clear(...)` (Redis `Remove`, not a Postgres write), and
  emits `StatusEventTypeCooldownExpired` directly via
  `producer.ProviderImpl`. No `gorm.DB`/`ExecuteTransaction` call appears
  anywhere in this function — genuinely no DB write, confirmed by reading
  both this function and `cooldown_registry.go` end to end.
- `skill/processor.go:221` `SetCooldownAndEmit` (and its mb-taking
  `SetCooldown` at `:201`) — same reasoning as `ExpireCooldowns`.
  `SetCooldown` calls only `GetRegistry().Apply(...)` (Redis `Put` +
  `Set.Add`) and `p.ByIdProvider` (a read); it performs no Postgres write,
  so the `StatusEventTypeCooldownApplied` event asserts no DB state
  change. Left on the direct `message.Emit(producer.ProviderImpl(...))`
  path. Not called out by name in the task brief (which only named
  `ExpireCooldowns` as the registry-only example), but it is the same
  Redis-only cooldown registry with the identical no-DB-write
  justification, so it is classified the same way.
- `skill/processor.go:268` `RequestCreate` and `skill/processor.go:273`
  `RequestUpdate` — command emits (`CommandTypeRequestCreate` /
  `CommandTypeRequestUpdate`, `EnvCommandTopic`) to the atlas-skills
  service's own command consumer (a self-loop used by the REST layer,
  `skill/resource.go:53,91`), not a DB-state-asserting event; both methods
  are one-line direct `producer.ProviderImpl` calls with no DB read or
  write at all. Left direct per the "COMMAND emits to other services"
  classification rule.
- `macro/processor.go:121` `Delete(characterId)` — has its own
  `ExecuteTransaction` (the 2nd of the 2 tx sites counted in the brief)
  but emits no Kafka event at all (no `message.Emit`, no `mb.Put`); there
  is nothing to migrate here, it is a pure DB delete used by the
  saga/cleanup path.

### Notes

- **Enumeration.** Both the base grep (`message.Emit`/`EmitWithResult`/
  `producer.ProviderImpl`/`database.ExecuteTransaction`) and the extra
  sweep (`GetAll()`/`NewBuffer()`/`p.p(`/`producer.Provider`/`AndEmit(`)
  were run from the module root. They landed on exactly the 4 `message.Emit`
  + 3 `ExecuteTransaction` sites in `skill/processor.go` and the 1
  `message.Emit` + 2 `ExecuteTransaction` sites in `macro/processor.go`
  named in the brief — no hand-rolled stored-provider flush loop and no
  additional `*AndEmit` method anywhere else in the module. Every
  `*AndEmit` method in both packages was read in full (`CreateAndEmit`,
  `UpdateAndEmit`, `SetCooldownAndEmit`, `DeleteForSagaCompensationAndEmit`
  in `skill/`; `UpdateAndEmit` in `macro/`).
- **No sub-processor fields.** Neither `skill.ProcessorImpl` nor
  `macro.ProcessorImpl` holds a nested sub-processor field (both are
  `{l, ctx, db, t}`), so the "rebind nested sub-processors in
  WithTransaction" pitfall does not apply — `WithTransaction` only needed
  to swap `db`.
- **No rejection/error event topic.** `kafka/message/skill` and
  `kafka/message/macro` each define only `EnvStatusEventTopic` and
  `EnvCommandTopic` (skill) / `EnvStatusEventTopic` (macro) — there is no
  rejection/error topic in this service's message packages, so none of
  the failure-path pitfalls (rejection-then-`return nil`, or
  shared-buffer-forwarded-on-failure) apply; no `rejectEmit` closures were
  needed.
- **No batching-per-item-command concern.** `macro.Update` loops over
  `macros` and calls `create` per item, but every `create` call is a plain
  DB write inside the shared `tx` — no per-item direct-path command or
  side effect fires mid-loop, so the atlas-notes batching pitfall does not
  apply.
- **Pre-existing tests untouched.** No test in `skill/processor_test.go`,
  `macro/processor_test.go`, `kafka/consumer/skill/consumer_test.go`, or
  `kafka/consumer/macro/consumer_test.go` calls any `*AndEmit` method —
  every test exercises the `mb`-taking methods (`Create`, `Update`,
  `SetCooldown`, `DeleteForSagaCompensation`, macro `Update`) directly with
  an explicit `message.NewBuffer()` and asserts against buffer contents or
  DB state. None asserted direct-path emission for a now-migrated flow, so
  no test needed updating to assert outbox rows instead.
- `go mod tidy`, `go build ./...`, `go vet ./...`, and `go test -race
  ./...` all clean in `services/atlas-skills/atlas.com/skills`;
  `docker buildx bake atlas-skills` succeeded; `tools/redis-key-guard.sh`
  clean (exit 0 across the whole repo).
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see `bug_execute_transaction_noop.md`); this task's
  migration uses the correct seam and becomes atomic for free once
  task-119 lands.

## atlas-merchant

Module: `services/atlas-merchant/atlas.com/merchant`. All line numbers below
reflect `shop/processor.go` pre-migration (Task 19).

### Migrated

Each site now opens (or reuses) `database.ExecuteTransaction` and enqueues its
event(s) via `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))`, calling
`p.WithTransaction(tx).<Method>(buf)(...)` instead of the direct
`message.Emit(p.producer)(...)` path. `shop.ProcessorImpl` had no
`WithTransaction` method before this task — added
`WithTransaction(tx *gorm.DB) Processor` (rebinds `db`, keeps
`l`/`ctx`/`t`/`producer`) and added it to the `Processor` interface (and to
`shop/mock/processor.go`'s `ProcessorMock`, which asserts `var _
shop.Processor = (*ProcessorMock)(nil)` and otherwise had no reason to
change).

- `shop/processor.go:986` `OpenShopAndEmit` — Pattern A. `OpenShop` (`:257`)
  transitions Draft→Open via its own `ExecuteTransaction`; the
  `StatusEventShopOpenedProvider` now enqueues in the same outer tx
  (re-entrant with the inner one).
- `shop/processor.go:992` `CloseShopAndEmit` — Pattern A. `CloseShop`
  (`:386`) transitions the shop to Closed via its own `ExecuteTransaction`,
  then (post-tx in the pre-migration code, now inside the outer tx) ejects
  visitors (Redis, buffers `VISITOR_EJECTED` events), conditionally calls
  `storeToFrederick` (which constructs `frederick.NewProcessor(p.l, p.ctx,
  p.db)` — now `p.db == tx` after rebinding, so `frederick.storeItems`'s own
  `ExecuteTransaction` (the "1 tx" in `frederick/administrator.go:15`) is
  re-entrant against the same outer tx), and buffers `AcceptAssetCommand`
  for returned listings before the final `StatusEventShopClosedProvider`.
  All of these events/commands are consequences of the same close
  operation and are now enqueued atomically with the shop-state write and
  the Frederick storage write. This resolves the brief's "check frederick's
  caller for a coupled emit" instruction: the emit is coupled (via
  `CloseShop`'s final status event), so `storeItems`'s tx rides along
  instead of needing its own separate Pattern A treatment.
- `shop/processor.go:998` `EnterMaintenanceAndEmit` — Pattern A.
  `EnterMaintenance` (`:303`) transitions Open→Maintenance via its own
  `ExecuteTransaction`; ejection events + `StatusEventMaintenanceEnteredProvider`
  now enqueue in the outer tx.
- `shop/processor.go:1004` `ExitMaintenanceAndEmit` — Pattern A.
  `ExitMaintenance` (`:335`) transitions Maintenance→Open or →Closed (if
  empty) via its own `ExecuteTransaction`; the resulting
  `StatusEventShopClosedProvider`/`StatusEventMaintenanceExitedProvider` now
  enqueue in the outer tx.
- `shop/processor.go:1022` `AddListingAndEmit` — Pattern A (result captured
  via outer `var result listing.Model` + closure, since this service has no
  `EmitWithResult`). `AddListing` (`:500`) creates the listing row via its
  own `ExecuteTransaction`; the `ReleaseAssetCommand` now enqueues in the
  outer tx.
- `shop/processor.go:1032` `RemoveListingAndEmit` — Pattern A (same
  result-capture shape). `RemoveListing` (`:553`) deletes the listing row
  via its own `ExecuteTransaction`; the `AcceptAssetCommand` (item return)
  now enqueues in the outer tx.
- `shop/processor.go:1042` `PurchaseBundleAndEmit` — Pattern A (same
  result-capture shape). `PurchaseBundle` (`:625`) updates/deletes the
  listing and optionally closes the shop via its own `ExecuteTransaction`;
  the meso-debit command, asset-grant command, meso-credit command,
  `ListingEventPurchasedProvider`, and (if sold out)
  `StatusEventShopClosedProvider` all now enqueue atomically with the
  purchase write in the outer tx. The version-conflict/insufficient-bundles
  error paths return from inside the inner `ExecuteTransaction` before any
  `mb.Put` call, so no event is ever buffered on a failed purchase (nothing
  new needed here — already correct pre-migration, just carried through).
- `shop/processor.go:1052` `SendMessageAndEmit` — Pattern A. `SendMessage`
  (`:891`) calls `msg.NewProcessor(p.l, p.ctx, p.db).SendMessage(...)`
  (`message/processor.go:35`, a bare `create(...)` with no
  `ExecuteTransaction` of its own) to persist the chat message; wrapped the
  whole `AndEmit` in a new `ExecuteTransaction` so the persisted message row
  and `StatusEventMessageSentProvider` commit/enqueue together (this is the
  "wrap bare Save/Create/Update writes in an explicit ExecuteTransaction"
  classification case — `SendMessage`'s DB write had no tx at all
  pre-migration).
- `shop/processor.go:1058` `RetrieveFrederickAndEmit` — Pattern A.
  `RetrieveFrederick` (`:925`) reads Frederick items/mesos, buffers
  `AcceptAssetCommand`/`ChangeMesoCommand` per item/mesos-total, then calls
  `fp.ClearItems`/`ClearMesos`/`ClearNotifications` (`frederick/processor.go`
  — bare deletes, no `ExecuteTransaction`). Wrapped the whole `AndEmit` in a
  new `ExecuteTransaction`; `frederick.NewProcessor(p.l, p.ctx, p.db)` picks
  up `tx` automatically since it is constructed fresh from `p.db` on every
  call (no stored sub-processor field to rebind), so the clears and the
  command enqueue now commit together.

### Left direct

- `shop/processor.go:1010` `EnterShopAndEmit` — `EnterShop` (`:763`) only
  reads the shop row (`getById`, no write) and mutates the Redis visitor
  registry (`visitor.GetRegistry().AddVisitor`/`GetVisitors`); no Postgres
  write. `StatusEventVisitorEnteredProvider` (and the capacity-full
  rejection `StatusEventCapacityFullProvider`) asserts no DB state change.
  Left on the direct `message.Emit(p.producer)` path.
- `shop/processor.go:1016` `ExitShopAndEmit` — `ExitShop` (`:803`) only
  touches the Redis visitor registry (`RemoveVisitor`); no Postgres write.
  `StatusEventVisitorExitedProvider` left direct for the same reason.
- `kafka/consumer/merchant/consumer.go:198` — `handlePurchaseBundleCommand`
  builds `producer.ProviderImpl(l)(ctx)` outside any tx to fire
  `StatusEventPurchaseFailedProvider` when `PurchaseBundleAndEmit` returns
  an error (version conflict / insufficient bundles / other). This is the
  D7 rejection-event case: the purchase did not commit, so there is no DB
  state change to couple the emit to. Already correctly on the direct path
  pre-migration (fired from the caller, after the failed `ExecuteTransaction`
  returned) — no change needed, confirmed by reading `PurchaseBundle` end
  to end (its error returns happen before any `mb.Put`).
- `shop/processor.go:605` `UpdateListing` — has its own `ExecuteTransaction`
  (counted as 1 of the brief's "8 tx" in `shop/processor.go`, alongside
  `UpdateFields`) but emits no Kafka event at all (no `message.Emit`, no
  `mb.Put`, no `*AndEmit` variant in the `Processor` interface). Nothing to
  migrate.
- `shop/processor.go:163` `CreateShop` — writes the shop entity directly
  (no `message.Emit`/`mb.Put` anywhere in the method or its callers'
  `handlePlaceShopCommand`); not part of the 11-site enumeration, no change.

### Notes

- **Enumeration.** Ran both the base grep
  (`message.Emit(\|message.EmitWithResult\|producer.ProviderImpl\|database.ExecuteTransaction`)
  and the extra sweep (`\.GetAll()\|NewBuffer()\|\bp\.p(\|producer.Provider\b\|AndEmit(`)
  from the module root. They landed on exactly the 11 `message.Emit`
  (`AndEmit` wrappers) + 8 `database.ExecuteTransaction` sites in
  `shop/processor.go` and the 1 `database.ExecuteTransaction` site in
  `frederick/administrator.go` named in the brief, plus the 1
  `producer.ProviderImpl` site in `kafka/consumer/merchant/consumer.go:198`
  — no hand-rolled stored-provider flush loop anywhere. Every `*AndEmit`
  method was read in full, including the two (`EnterShopAndEmit`,
  `ExitShopAndEmit`) whose underlying method has no DB write at all — these
  do not appear in the "8 tx" count, confirming the brief's implicit
  classification.
- **No EmitWithResult.** Confirmed by reading `kafka/message/message.go`:
  only `NewBuffer`/`Put`/`GetAll`/`Emit` exist, no `EmitWithResult`. The
  three result-returning `AndEmit` methods (`AddListingAndEmit`,
  `RemoveListingAndEmit`, `PurchaseBundleAndEmit`) already captured their
  result via an outer `var result T` + closure assignment even
  pre-migration (a hand-rolled Pattern-B shape built on plain `Emit`), so
  the migration kept that shape and just swapped the producer + added the
  outer `ExecuteTransaction`.
- **No sub-processor fields.** `shop.ProcessorImpl` is `{l, ctx, db, t,
  producer}` — no stored sub-processor field, so the "rebind nested
  sub-processors" pitfall doesn't apply structurally. However, `CloseShop`
  and `RetrieveFrederick` construct `frederick.NewProcessor(p.l, p.ctx,
  p.db)` and `SendMessage` constructs `msg.NewProcessor(p.l, p.ctx, p.db)`
  fresh on every call using the (now possibly tx-bound) `p.db` — this has
  the same effect as rebinding, without needing a stored-field rebind in
  `WithTransaction`.
- **No shared-buffer failure-forwarding pitfall.** Checked every migrated
  method for a handled-failure branch that `mb.Put`s then `return nil`s:
  none exists. All error returns in `OpenShop`/`CloseShop`/
  `EnterMaintenance`/`ExitMaintenance`/`AddListing`/`RemoveListing`/
  `PurchaseBundle`/`SendMessage`/`RetrieveFrederick` happen before any
  `mb.Put` call for that invocation, so no `rejectEmit`-closure
  restructuring was needed. The one genuine rejection event in this module
  (`StatusEventPurchaseFailedProvider`) is already fired from the consumer
  caller on the direct path, never through the shared buffer.
- **No batching-per-item-command concern.** `shop/task.go`'s
  `ExpirationTask.Run()` loops over expired shops and calls
  `CloseShopAndEmit` once per shop — each call is an independent, fully
  committed `ExecuteTransaction` (not one shared outer tx wrapping the
  loop), so the atlas-notes per-item-command-timing pitfall does not apply.
  Same reasoning for `kafka/consumer/character/consumer.go`'s
  `handleLogout`, which loops over a character's open shops calling
  `CloseShopAndEmit` per shop.
- **Frederick's "1 tx" is not migrated separately.** Per the brief's
  instruction to check the caller: `frederick/administrator.go:15`'s
  `storeItems` `ExecuteTransaction` is only ever invoked (via
  `frederick.ProcessorImpl.StoreItems`) from `shop.storeToFrederick`, which
  is only called from the now-migrated `CloseShop`. Since
  `frederick.NewProcessor` is constructed with `p.db` (== `tx` after
  `WithTransaction` rebinding) at call time, `storeItems`'s own
  `ExecuteTransaction(db, ...)` runs against `tx` re-entrantly — no
  separate Pattern A wrap needed on the frederick side.
- **Pre-existing tests untouched.** No test in `shop/processor_test.go`,
  `frederick/processor_test.go`, `frederick/notification_test.go`, or
  `message/processor_test.go` calls any `*AndEmit` method — every test
  exercises the `mb`-taking methods (`OpenShop`, `CloseShop`,
  `EnterMaintenance`, `ExitMaintenance`, `AddListing`, `RemoveListing`,
  `PurchaseBundle`, `UpdateListing`) directly against a hand-built
  `message.Buffer`/`testBuffer()`, or the plain frederick/message processor
  methods against a test DB. None asserted direct-path Kafka emission for a
  now-migrated flow, so no test needed updating to assert outbox rows
  instead.
- `go mod tidy`, `go build ./...`, `go vet ./...`, and `go test -race ./...`
  all clean in `services/atlas-merchant/atlas.com/merchant`; `docker buildx
  bake atlas-merchant` succeeded; `tools/redis-key-guard.sh` clean (exit 0
  across the whole repo).
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see `bug_execute_transaction_noop.md`); this task's
  migration uses the correct seam and becomes atomic for free once
  task-119 lands.

### Fix pass: propagate swallowed Frederick errors (post-review)

Review flagged that `RetrieveFrederick` and `CloseShop`'s `storeToFrederick`
helper logged-and-continued on Frederick DB-write failures instead of
returning the error, which defeated the atomicity contract this task
introduced (a failed clear/store would still let the tx "commit" and the
outbox event enqueue — duplication risk on retrieve, silent item loss on
close). Fixed in `shop/processor.go`:

- `RetrieveFrederick` (~:981-993): `ClearItems`/`ClearMesos`/`ClearNotifications`
  now `return err` (log preserved) on first failure instead of continuing.
- `storeToFrederick` (~:473-517): signature changed from `void` to
  `error`; `StoreItems`/`StoreMesos`/`CreateNotification` now `return err`
  (log preserved) on first failure. Sole caller `CloseShop` (~:459-463)
  updated to propagate the error (`if err := p.storeToFrederick(...); err
  != nil { return err }`).

Added two tests in `shop/processor_test.go` (`TestRetrieveFrederick_ClearFailure_SkipsOutbox`,
`TestCloseShop_FrederickStoreFailure_SkipsOutbox`) using table-drop as the
failure-injection seam (`frederick.Processor` has no mock/fake in this
module; `shop.ProcessorImpl` constructs it directly, so there's no
injectable interface at the shop-processor level). Both assert the error
propagates through `RetrieveFrederickAndEmit`/`CloseShopAndEmit` and that
zero rows land in `outbox_entries`. Per the pre-existing
`bug_execute_transaction_noop` limitation (task-119), the tests do **not**
assert DB-row rollback (e.g. shop staying Open, cleared item reappearing) —
`ExecuteTransaction` never actually starts a real transaction today, so the
already-executed writes before the failure point are not undone at the SQL
level; only the outbox-enqueue suppression and error propagation are
verified, which is what this fix pass delivers and what is testable given
the current transaction infra.

## atlas-npc-shops

**Zero tx-coupled emit sites; no code change to any emit/tx pairing.**
Wiring (go.mod + main.go outbox drainer boot) applied per the recipe
template so the drainer runs, but no call site needed migration.

- **Migrated**: none.
- **Left direct**:
  - `shops/processor.go:268` (`EnterAndEmit`/`Enter`) — `GetRegistry().AddCharacter`
    is a Redis-backed registry write (`shops/registry.go`), not a Postgres
    DB write inside `p.db`; the only Kafka event is `EnvStatusEventTopic`
    `enteredEventProvider`. No SQL state to couple atomically with.
  - `shops/processor.go:287` (`ExitAndEmit`/`Exit`) — same reasoning;
    `GetRegistry().RemoveCharacter` is Redis-only, event is `exitedEventProvider`.
  - `shops/processor.go:369` (`BuyAndEmit`/`Buy`) — no local DB write at all;
    success path only enqueues `RequestChangeMeso`/`RequestCreateItem`
    COMMAND events on `character.EnvCommandTopic`/`compartment.EnvCommandTopic`
    (relayed to other services, which own the actual state change and their
    own outbox); failure paths only `mb.Put` rejection/error events
    (`EnvStatusEventTopic`, `errorEventProvider`/`reasonErrorEventProvider`)
    reflecting no state change in this service.
  - `shops/processor.go:480` (`SellAndEmit`/`Sell`) — same reasoning as Buy:
    COMMAND relay (`RequestChangeMeso`, `RequestDestroyItem`) plus
    rejection-only status events, no local DB write.
  - `shops/processor.go:570` (`RechargeAndEmit`/`Recharge`) — same reasoning:
    COMMAND relay (`RequestChangeMeso`, `RequestRechargeItem`) plus
    rejection-only status events, no local DB write.
  - `shops/processor.go:92` (struct-init `kp: producer.ProviderImpl(l)(ctx)`)
    — kept as-is per the recipe's "struct-init stored providers" guidance:
    every method that flushes through `p.kp` (the five `*AndEmit` methods
    above) does no DB write, so the direct provider is the correct choice
    for all of them; there is no tx-coupled use to restructure.
  - `shops/processor.go:194` (`UpdateShop`, `ExecuteTransaction`) and
    `shops/processor.go:312` (`DeleteAllShops`, `ExecuteTransaction`) —
    pure DB writes via `commodities.Processor`/local entity calls, no Kafka
    emit anywhere in either method or in their callers
    (`shops/resource.go:186,304`, plain REST CRUD handlers with no emit).
  - `shops/administrator.go:69` (`BulkCreateShops`, `ExecuteTransaction`)
    and `commodities/administrator.go:86` (`BulkCreateCommodities`,
    `ExecuteTransaction`) — pure DB writes, sole caller is
    `shops/subdomain.go:91` `ShopSubdomain.BulkCreate` (seeder subdomain
    ingestion), which has no coupled emit anywhere in its call chain.
- **Notes**: Verified by reading every method reachable from the step-2 and
  the hand-rolled-flush greps (`.GetAll()`/`NewBuffer()`/`p.p(`/`AndEmit(`)
  — no hidden per-topic loop over `p.kp` exists (the five `*AndEmit`
  methods are the only consumers, and each already goes through
  `message.Emit(p.kp)(...)`). `shops.ProcessorImpl` has no `WithTransaction`
  method (unlike its `cp commodities.Processor` field, which does); none
  was needed since no method required rebinding a tx-bound outbox provider.
  `main.go` wiring (import, `outboxlib.Migration` appended as 4th
  `SetMigrations` arg after the seeder func, drainer boot block using
  `tdm := service.GetTeardownManager()`) applied per the recipe template so
  the drainer/publisher run in this service even though no emit path uses
  the outbox yet — consistent with how other unaffected sites are wired
  service-wide per the plan.

## atlas-tenants

All 12 tx-coupled emit sites (3 `message.Emit` + 6 `message.EmitWithResult`
in `configuration/processor.go`, 1 `message.Emit` + 2
`message.EmitWithResult` in `tenant/processor.go`) migrated to the outbox.
Enumeration matched the brief exactly — the step-2 grep and the
hand-rolled-flush grep (`.GetAll()`/`NewBuffer()`/`p.p(`/`AndEmit(`) turned
up no extra sites beyond the 12 listed.

- **Migrated**:
  - `tenant/processor.go:113` (`CreateAndEmit`) — Pattern B (EmitWithResult).
  - `tenant/processor.go:183` (`UpdateAndEmit`) — Pattern B (EmitWithResult).
  - `tenant/processor.go:245` (`DeleteAndEmit`) — Pattern A (Emit).
  - `configuration/processor.go:210` (`CreateRouteAndEmit`) — Pattern B.
  - `configuration/processor.go:299` (`UpdateRouteAndEmit`) — Pattern B.
  - `configuration/processor.go:332` (`DeleteRouteAndEmit`) — Pattern A.
  - `configuration/processor.go:456` (`CreateVesselAndEmit`) — Pattern B.
  - `configuration/processor.go:545` (`UpdateVesselAndEmit`) — Pattern B.
  - `configuration/processor.go:578` (`DeleteVesselAndEmit`) — Pattern A.
  - `configuration/processor.go:693` (`CreateInstanceRouteAndEmit`) — Pattern B.
  - `configuration/processor.go:778` (`UpdateInstanceRouteAndEmit`) — Pattern B.
  - `configuration/processor.go:810` (`DeleteInstanceRouteAndEmit`) — Pattern A.
- **Left direct**: none — every emit site asserts a DB state change
  (route/vessel/instance-route/tenant create-update-delete), so all 12
  migrated. No rejection/error-only or cross-service COMMAND emit paths
  exist in either processor.
- **Notes**:
  - Neither `tenant.ProcessorImpl` nor `configuration.ProcessorImpl`
    exposes a `WithTransaction` method, so every migrated site follows the
    recipe's fallback: `NewProcessor(p.l, p.ctx, tx)` constructs a
    tx-scoped processor and the wrapped `Foo(mb)(args...)` call runs
    against it, while the enqueue itself uses
    `outbox.EmitProvider(p.l, p.ctx, tx)` built directly from the same
    `tx` — enqueue and domain write share one transaction.
  - Struct-init stored providers at `tenant/processor.go:64`
    (`p: producer.ProviderImpl(l)(ctx)`, originally line 62 in the brief
    before the new `database`/`outbox` imports shifted it) and
    `configuration/processor.go:110` (same, originally line 108) are now
    dead in the migrated
    `*AndEmit` paths — no site reads `p.p` any more (confirmed via
    `grep -n "p\.p\b"` returning empty in both files after migration). The
    field is left in place (still set by `NewProcessor` for API-compat and
    because Go does not flag unused struct fields); it has zero remaining
    call sites doing a direct-path emit, so there is no lingering
    tx-vs-direct-provider hazard.
  - `tenant/administrator.go` (`CreateTenant`/`UpdateTenant`/`DeleteTenant`)
    and `configuration/administrator.go`
    (`CreateConfiguration`/`UpdateConfiguration`/`DeleteConfiguration`)
    each open their own `database.ExecuteTransaction(db, ...)`. Because the
    migrated `*AndEmit` methods pass `tx` through as the `db` field of a
    freshly-constructed processor, these administrator calls now run
    `database.ExecuteTransaction(tx, ...)` — re-entrant nesting inside the
    outer tx, exactly as the recipe describes; no administrator.go edits
    were needed.
  - No sub-processor fields exist on either `ProcessorImpl` (no
    `WithTransaction` rebind concern applies).
  - No rejection/error-only emit branches exist in either processor (every
    `mb.Put` in both files sits on a success path immediately after a
    completed write); no `rejectEmit` closure pattern was needed.
  - The three `Seed*` methods (`SeedRoutes`, `SeedInstanceRoutes`,
    `SeedVessels`) loop calling `p.CreateRouteAndEmit` /
    `p.CreateInstanceRouteAndEmit` / `p.CreateVesselAndEmit` per item; each
    call now opens its own `ExecuteTransaction`, preserving the
    pre-migration per-item-commit semantics (no shared-loop-tx batching
    pitfall from the atlas-notes case applies here — there is no
    cross-service command to defer, only same-service status events per
    item).
  - Existing tests (`tenant/processor_test.go`, `configuration/processor_test.go`)
    exercise the buffer-based `Create`/`Update`/`Delete` methods directly
    via a local `testProcessor` shim (bypassing Kafka/producer entirely) —
    none of them call the `*AndEmit` methods, so no test needed updating
    for the new outbox-enqueue behavior.
  - Config-status projection consumers (login/channel) read the emitted
    event payloads unchanged — `CreateRouteStatusEventProvider` /
    `CreateVesselStatusEventProvider` / `CreateInstanceRouteStatusEventProvider`
    / `CreateStatusEventProvider` (tenant) are untouched; only the delivery
    path (outbox vs. direct-after-return) changed, not the header/body
    construction, so byte-parity for the projection is preserved.

## atlas-mounts

Module: `services/atlas-mounts/atlas.com/mounts`. This is a Pattern C
(call-site wrapping) service: `mount/processor.go`'s three tx-bearing
methods (`ApplyTick`, `ApplyFeedAndEmit`, `EmitSet` — via its
`GetByCharacterId` default-create path) take an already-open `*message.Buffer`
and never call `producer.ProviderImpl` or open their own emit; the
`database.ExecuteTransaction` + `message.Emit(...)` wiring lives entirely at
the three call sites outside the processor (a 60s tiredness ticker and two
Kafka consumers), so the processor itself needed zero changes beyond
accepting `tx` as its `db` via `NewProcessor(l, ctx, tx)`.

### Migrated

- `mount/task.go:29` (`applyTick` seam, called from `TirednessTask.Run`) —
  Pattern C. Wraps `NewProcessor(l, ctx, tx)` + `ApplyTick` (persists the
  tiredness increment and `LastTirednessTickAt`, buffers the `TICK` status
  event) in one `database.ExecuteTransaction(db.WithContext(ctx), ...)`,
  enqueuing via `mountmessage.Emit(outbox.EmitProvider(l, ctx, tx))`.
- `kafka/consumer/buff/consumer.go:33` (`emitSet` seam, called from
  `handleBuffApplied` after a tamed-mount buff activation) — Pattern C.
  Wraps `NewProcessor(l, ctx, tx)` + `EmitSet` the same way. `EmitSet` calls
  `GetByCharacterId`, which default-creates a mount row inside its own
  `ExecuteTransaction` on first read (re-entrant into the same outer `tx`)
  before buffering the `SET` status event — a genuine DB write, not a pure
  read, on first activation.
- `kafka/consumer/food/consumer.go:24` (`applyFeed` seam, called from
  `handleTamingMobFood`) — Pattern C. Wraps `NewProcessor(l, ctx, tx)` +
  `ApplyFeedAndEmit` (persists level/exp/tiredness from the feed math,
  buffers the `FEED` status event) the same way.

All three call sites already had `l`, `ctx`, and `db` in scope pre-migration
(the ticker seam takes them as explicit params from `TirednessTask.Run`;
both consumer seams take them as explicit params from their `handle*`
functions, which receive `db` via `InitHandlers(l)(db)(...)` closures) — no
new plumbing was needed to reach a DB handle at any of the three sites.

### Left direct

None. All three tx-coupled emit sites in the brief were migrated; no
rejection/error event topic exists in this service's message package
(`kafka/message/mount` defines only the one `EnvStatusEventTopic` used for
`TICK`/`FEED`/`SET` bodies), so none of the failure-path pitfalls
(rejection-then-`return nil`, shared-buffer-forwarded-on-failure) apply.

### Notes

- Enumeration: the step-2 grep (`message.Emit`/`EmitWithResult`/
  `producer.ProviderImpl`/`database.ExecuteTransaction`) plus the extra
  `GetAll()`/`NewBuffer()`/`p.p(`/`producer.Provider`/`AndEmit(` sweep both
  landed on exactly the 3 call sites named in the brief plus
  `processor.go`'s 3 `ExecuteTransaction` sites (`GetByCharacterId`,
  `ApplyTick`, `ApplyFeedAndEmit` — all pre-existing, unchanged bodies) —
  no additional hand-rolled buffer-flush loop anywhere else in the module.
- `ProcessorImpl` has an unused `kp producer.Provider` field (set in
  `NewProcessor` via `producer.ProviderImpl(l)(ctx)` but never read anywhere
  in the module — confirmed via `grep -rn "\.kp\b"`). It predates this task,
  is dead code independent of the migration, and is out of this task's
  scope (Task 22 is call-site wrapping only); left as-is.
- No pre-existing test asserted direct-path emission for a migrated flow:
  `mount/task_test.go`, `kafka/consumer/buff/consumer_test.go`, and
  `kafka/consumer/food/consumer_test.go` all override the `applyTick`/
  `emitSet`/`applyFeed` function-seam variables with fakes that record their
  arguments, never exercising the real `ExecuteTransaction`/producer wiring
  — so no test needed updating to assert outbox rows instead.
- `go test -race ./...`, `go vet ./...`, `go build ./...` all clean;
  `docker buildx bake atlas-mounts` succeeded; `tools/redis-key-guard.sh`
  clean (exit 0 across the whole repo).
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see `bug_execute_transaction_noop.md`); this task's
  migration uses the correct seam and becomes atomic for free once
  task-119 lands.

## atlas-quest

Divergent case: no `message.Emit`/`EmitWithResult` call sites and no local
`producer.ProviderImpl`; emission ran through an `EventEmitter` interface
(`quest/event_emitter.go`) whose `KafkaEventEmitter` impl published directly,
with all 8 call sites running AFTER their associated `ExecuteTransaction`
block(s) had already committed. Migration added a per-transaction
`OutboxEventEmitter` (`quest/outbox_event_emitter.go`) and restructured the
5 methods that own the 8 sites so each event enqueues inside the same
transaction as the domain write(s) it reports on.

### Migrated

- `quest/processor.go` `startWithDefinition` (EmitQuestStarted, was line 216)
  — restructured: `startCore` lost its own internal `ExecuteTransaction` and
  now takes `tx *gorm.DB` directly; `startWithDefinition` wraps
  `startCore` + `processStartActions` (which also now takes `tx`) + the
  post-write reload (`p.WithTransaction(tx).GetByCharacterIdAndQuestId`) +
  `p.txEmitter(tx).EmitQuestStarted(...)` in one outer
  `database.ExecuteTransaction`, capturing `updated` outside the closure
  (Pattern-B style) and returning it after.
- `quest/processor.go` `StartChained` (EmitQuestStarted, was line 361) —
  same restructure: `startChainedCore` takes `tx` directly (no more own
  tx), wrapped with `processStartActions(tx, ...)` + reload + emit in one
  outer `ExecuteTransaction`.
- `quest/processor.go` `Complete` (EmitQuestCompleted, was line 461) — same
  shape: `completeCore` takes `tx` directly, wrapped with
  `processEndActions(tx, ...)` + emit in one outer `ExecuteTransaction`;
  `nextQuestId` captured outside the closure.
- `quest/processor.go` `Forfeit` (EmitQuestForfeited, was line 535) — emit
  moved literally inside the existing `ExecuteTransaction` closure, right
  after `forfeitQuest(tx, ...)`.
- `quest/processor.go` `SetProgress` (EmitProgressUpdated ×2, was lines
  580/586, one per reload-success/reload-failure branch) — emit(s) and the
  post-write reload moved inside the existing `ExecuteTransaction` closure,
  reading via `p.WithTransaction(tx).GetByCharacterIdAndQuestId` so the
  reload sees the just-written row through the same tx handle.
- `quest/processor.go` `processStartActions` (EmitSaga, was line 866) —
  Pattern B: no longer opens its own transaction (it never had a DB write
  of its own — only builds and emits a saga command); now takes `tx` from
  its caller (`startWithDefinition`/`StartChained`) and calls
  `p.txEmitter(tx).EmitSaga(s)`, so the reward-saga command enqueues
  atomically with the quest-start write it accompanies. `awardedItems` is
  still returned as the first return value regardless of the emit outcome
  (unchanged from pre-migration).
- `quest/processor.go` `processEndActions` (EmitSaga, was line 936) — same
  restructure as `processStartActions`, called from `Complete` with the
  same outer `tx`.

### Left direct

None — all 8 emit sites named in the brief were migrated. No
rejection/error-only event exists on this service's `EventEmitter`
interface (every method reports a state change that already committed, or
a saga command that accompanies one), so none of the failure-path pitfalls
(rejection-then-`return nil`, shared-buffer-forwarded-on-failure) apply.

### Notes

- `startCore`, `startChainedCore`, `completeCore`, `processStartActions`,
  and `processEndActions` are all private, single-call-site helpers (each
  called from exactly one public method), so changing their signatures to
  accept `tx *gorm.DB` directly (instead of opening their own
  `database.ExecuteTransaction`) was a safe, contained refactor — no other
  caller exists (`CheckAutoStart`/`CheckAutoComplete` call the *public*
  `startWithDefinition`/`Complete`, not the private core helpers).
- The `ProcessorImpl.eventEmitter` field is still populated (by
  `NewProcessor` via `NewKafkaEventEmitter` and by
  `NewProcessorWithDependencies` via the injected mock) but is no longer
  read anywhere in `processor.go` — every former `p.eventEmitter.EmitX(...)`
  call site now goes through `p.txEmitter(tx).EmitX(...)`. It's kept
  because `NewProcessorWithDependencies`'s signature (test-facing, used by
  `quest/processor_test.go` and 4 consumer test files) still takes an
  `EventEmitter` param, and `txEmitter` is defined in terms of it
  (`func(*gorm.DB) EventEmitter { return eventEmitter }`) so injected test
  mocks keep observing every emit unchanged.
- `NewProcessor`'s `txEmitter` defaults to
  `func(tx *gorm.DB) EventEmitter { return NewOutboxEventEmitter(l, ctx, tx) }`;
  `WithTransaction` carries the field forward unchanged so a
  `WithTransaction(tx)`-derived processor still resolves to the same
  emitter strategy (mock or outbox) as its parent.
- No pre-existing test asserted direct-path (Kafka) emission for a migrated
  flow — `quest/processor_test.go` and `quest/set_progress_cap_test.go` both
  inject `test.NewMockEventEmitter()` via `NewProcessorWithDependencies`
  and assert against its recorded `StartedEvents`/`CompletedEvents`/
  `ForfeitedEvents`/`ProgressEvents`/`SagaEvents` slices — these assertions
  are agnostic to whether the emit happened via direct Kafka publish or an
  outbox enqueue, since the mock's `EmitX` methods just append to a slice
  and return nil regardless of caller. No test file was modified.
- `go test -race ./...`, `go vet ./...`, `go build ./...` all clean;
  `docker buildx bake atlas-quest` succeeded; `tools/redis-key-guard.sh`
  clean (exit 0 across the whole repo).
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see `bug_execute_transaction_noop.md`); this task's
  migration uses the correct seam and becomes atomic for free once
  task-119 lands. Until then, nesting `startCore`/`processStartActions`/etc
  calls inside one outer `ExecuteTransaction` closure is behaviorally
  identical to the pre-migration separate-transaction sequence (since
  `ExecuteTransaction` just invokes its callback directly today).
