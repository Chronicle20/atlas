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
