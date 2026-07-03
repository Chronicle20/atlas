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
