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
`asset/processor.go` as of the Task 11 commit. Wiring: `main.go` appends
`outboxlib.Migration` to `database.SetMigrations(...)` and boots the
drainer right after `database.Connect(...)`, using the existing
`tdm := service.GetTeardownManager()` var (this service's local name for
the teardown manager, not `lifecycle`).

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
  `CreateAssetAndLock`, which locks then delegates to `CreateAsset`). The
  failure branch's `CreationFailedEventStatusProvider` is put on the same
  `mb` as the success path inside `CreateAsset` (not a separate
  `message.Emit` call), so it rides along into the outbox with the rest of
  the site's buffer — consistent with the `character/processor.go`
  precedent (Task 9 inventory note on `AwardExperienceAndEmit`'s bundled
  command) of migrating a call site's whole buffer rather than
  splitting sub-`Put`s out of an already-migrated site.
- `compartment/processor.go:1093` `AttemptEquipmentPickUpAndEmit` — Pattern
  A. Bundles the post-tx `dropProcessor.CancelReservation`/no call in this
  method (success path only calls `RequestPickUp`); see Notes.
- `compartment/processor.go:1167` `AttemptItemPickUpAndEmit` — Pattern A.
  Bundles the pickup-consume command and both
  `dropProcessor.RequestPickUp`/`CancelReservation` outcomes into the same
  migrated buffer; see Notes.
- `compartment/processor.go:1280` `RechargeAssetAndEmit` — Pattern A.
- `compartment/processor.go:1338` `MergeAndCompactAndEmit` — Pattern A.
- `compartment/processor.go:1346` `CompactAndSortAndEmit` — Pattern A.
- `compartment/processor.go:1467` `AcceptAndEmit` — Pattern A. The failure
  branch's `ErrorEventStatusProvider` (compartment `AcceptCommandFailed`)
  is put on the same shared `mb` inside `Accept`, riding along into the
  outbox with the rest of the buffer for the same reason as `CreateAsset`
  above.
- `compartment/processor.go:1580` `ReleaseAndEmit` — Pattern A. Same
  shared-buffer rejection pattern (`ReleaseCommandFailed`) as `Accept`.
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
- **Bundled drop/pickup commands ride the migrated buffer, not split out**:
  `AttemptEquipmentPickUpAndEmit` and `AttemptItemPickUpAndEmit` each emit,
  in addition to their compartment/asset state-change events,
  `COMMAND_TOPIC_DROP` commands (`CancelReservation`/`RequestPickUp` via
  `drop.Processor`) and, on the consume-on-pickup branch,
  `pickupMsg.EnvCommandTopic`. These are commands to the separate
  atlas-drop-domain, and D7 lists "COMMAND emits to OTHER services" as a
  leave-direct category — but here they are `mb.Put` calls sharing the
  *same* buffer/`message.Emit` call as the migrated state-change writes,
  not separate `message.Emit` sites of their own. Splitting them out would
  require threading a second buffer (or an out-of-band `rejectEmit`-style
  closure) through `AttemptEquipmentPickUp`/`AttemptItemPickUp`, which are
  exercised directly (bypassing `*AndEmit`) by
  `TestAttemptItemPickUpInventoryFull` and
  `TestAttemptItemPickUpConsumeOnPickup` — both assert the drop/pickup
  commands land in the *same* buffer passed to the un-wrapped method. Per
  the `character/processor.go` Task 9 precedent (`AwardExperienceAndEmit`'s
  bundled command note), a command that is a call site's own bundled
  follow-on effect migrates with the rest of that site's buffer rather than
  being split out. Flagging as a concern for review: these two commands
  will now be delayed until outbox drain instead of firing immediately,
  which is a latency change (not a correctness change) for the drop
  service's reservation cancel/pickup-finalize signal.
- `database.ExecuteTransaction` atomicity is still latent fleet-wide
  pending task-119 (see the `atlas-character` section above and project
  memory `bug_execute_transaction_noop.md`); this task's migrations use the
  correct seam and become atomic for free once task-119 lands.
- No pre-existing test asserted DIRECT-path (non-outbox) emission for any
  now-migrated flow in this module — all `*AndEmit` wrapper methods were
  untested in isolation; every existing test in `compartment/processor_test.go`,
  `asset/processor_test.go` exercises the un-wrapped `Method(mb)(...)` form
  directly against a manually constructed `message.Buffer`, so none needed
  updating. `go test -race ./...` passes unchanged.
