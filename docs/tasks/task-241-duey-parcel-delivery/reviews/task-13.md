# Review: Task 13 — Orchestrator parcel custody dispatch package and status consumer

Commit range: `1a7cc20d6..47fdbf191` (1 commit, `47fdbf191`)
Brief: `.superpowers/sdd/plan/task-13-brief.md`
Report: `.superpowers/sdd/plan/task-13-report.md`

## Scope confirmed

Diff touches exactly the 7 files the report lists (6 of the brief's 6 named
files, plus the disclosed `saga/event_acceptance.go` addition). No unrelated
churn. `git diff --stat` matches the review package. Scope matches what was
asked, with one disclosed expansion addressed under item 1 below.

## Findings

### 1. Disclosed scope expansion (`saga/event_acceptance.go`) — VERIFIED CORRECT

- `saga/event_acceptance.go:826-829` adds `EventKindParcelCustodyAccepted/
  Released/Error`; `:852-856` adds `acceptanceTable` rows for
  `TransferToParcel`/`WithdrawFromParcel` (both `{}`, composite) and
  `AcceptToParcel`/`ReleaseFromParcel` (real event lists); `:880-882` adds the
  matching `outcomeTable` rows.
- Genuinely required: `kafka/consumer/parcel/custody/consumer.go:75,92,109`
  calls `p.AcceptEvent(txId, saga.EventKindParcelCustody*)`, and
  `AcceptEvent` (`saga/processor.go:459`) gates on
  `StepAcceptsEvent(step.Action(), kind)`, which reads `acceptanceTable`.
  Without the new rows, `AcceptEvent` would never resolve to a pending step
  and every ack would silently no-op — confirmed by reading
  `saga/processor.go:422-488`.
- Row correctness verified against usage, not just shape: `AcceptToParcel`/
  `ReleaseFromParcel` are the exact `sharedsaga.Action` constants
  (`libs/atlas-saga/model.go:234-237`) that `expandTransferToParcel`/
  `expandWithdrawFromParcel` (`saga/processor.go:2162,2299`) emit as step
  actions — confirmed the action strings used in `NewStep[any]("accept_to_parcel",
  ..., AcceptToParcel, ...)` and `NewStep[any]("release_from_parcel", ...,
  ReleaseFromParcel, ...)` match the acceptanceTable keys added.
- Before/after check: `git show 1a7cc20d6:./saga/event_acceptance.go | grep
  Parcel` returns nothing — confirms no prior Parcel rows existed and none
  were overwritten.
- No behaviour changed for existing non-parcel actions: diff to
  `event_acceptance.go` is purely additive (`git diff --stat` shows `17
  insertions(+), 0 deletions(-)`); every other table entry is untouched.
- Verdict: correct and necessary, matches the MTS/Trade custody precedent
  (composite actions map to `{}`, atomic actions map to their real event
  list).

### 2. `main.go` registration — VERIFIED, all three sites present, nothing else disturbed

- Import: `main.go:282` (`parcelCustody "atlas-saga-orchestrator/kafka/consumer/parcel/custody"`).
- `InitConsumers`: `main.go:304` (`parcelCustody.InitConsumers(l)(cmf)(consumerGroupId)`), placed immediately after `mtsCustody.InitConsumers` before `tradeCustody.InitConsumers` — same alphabetical-by-custody-family grouping the file already uses.
- `InitHandlers`: `main.go:326-328`, placed immediately after the `mtsCustody.InitHandlers` block.
- Diff to `main.go` is exactly these three additions (`+5` lines total across
  the whole file per `git diff --stat`); no reordering or incidental edits to
  any other consumer registration. Build succeeds
  (`go build ./...` clean).

### 3. World-0 defaulting — VERIFIED, no seam in this diff

- `AcceptToParcelPayload.WorldId` (`libs/atlas-saga/payloads.go:1020`) is
  `world.Id` with tag `json:"worldId"`.
- `parcel.AcceptToParcelParams.WorldId` (`parcel/processor.go:371`) is also
  `world.Id`; `parcelCustody.AcceptToParcelCommandBody.WorldId`
  (`kafka/message/parcel/custody/kafka.go:173`) is also `world.Id` tagged
  `json:"worldId"` — same type, same tag, all three points in the chain.
- `AcceptToParcelProvider` (`parcel/producer.go:724`) copies
  `params.WorldId` straight into the body field with no intermediate
  conversion or default.
- Confirmed there is no call site in this diff that constructs
  `AcceptToParcelParams` from `AcceptToParcelPayload` (`grep -rn
  "AcceptToParcelAndEmit\|case AcceptToParcel:"` across `saga/*.go` and the
  module returns nothing) — the payload→params wiring genuinely does not
  exist yet, so there is no seam in this commit where a `WorldId` could be
  silently dropped to 0. This is consistent with the brief's statement that
  handler wiring is Task 14's job.
- `world.Id` is `byte` (`libs/atlas-constants/world/*.go:3`); JSON encoding
  of a byte-alias and a plain `byte` (as MTS's own `WorldId byte` field uses)
  are identical on the wire, so no cross-type marshal mismatch either.

## Other checks

- **Same package as Task 12's REST client**: `parcel/processor.go`,
  `producer.go`, `processor_test.go` are all `package parcel`, alongside
  Task 12's `parcel/rest.go` and `parcel/requests.go` (also `package
  parcel`). No `Processor`/`NewProcessor` name collision — Task 12's files
  define `RestModel`/`AssetData`/`RequestParcel` only. `rest.go`/
  `requests.go` are untouched by this diff (not in `git diff --stat`
  output).
- **Envelope vs. saga step payloads, field by field**:
  `AcceptToParcelCommandBody` (`kafka/message/parcel/custody/kafka.go:170-213`)
  matches `AcceptToParcelPayload` (`libs/atlas-saga/payloads.go:1016-1062`)
  field-for-field and tag-for-tag (both carry the same 39 fields with
  identical JSON tags; `TransactionId` lives on the envelope only, correctly
  omitted from the body). `ReleaseFromParcelCommandBody` matches
  `ReleaseFromParcelPayload` the same way (`ParcelId`, `RecipientId`,
  `TransactionId` on the envelope). No drift found.
- **MTS mirroring**: `kafka/message/parcel/custody/kafka.go`,
  `parcel/processor.go`, `parcel/producer.go`,
  `kafka/consumer/parcel/custody/consumer.go` all structurally mirror their
  MTS custody twins (envelope shape, `Processor` interface shape, pure
  `Buffer` methods + `AndEmit` wrappers, `InitConsumers`/`InitHandlers`
  shape). No direct producer calls found inside any `Buffer` method
  (`parcel/processor.go:451-491` — each `Buffer` method only calls
  `mb.Put`).
- **`RESTORE_PARCEL`/`REMOVE_PARCEL` doc comments**: state why they exist,
  matching the required content (`kafka/message/parcel/custody/kafka.go:147-154`
  — un-resolve a release whose downstream accept failed / hard-delete a
  still-pending row from a late accept after compensation). See Finding A
  below for a gap on the idempotency half of this requirement.
- **`libs/atlas-constants` reuse**: `world.Id` reused throughout, no new
  domain type/alias invented.
- **Test honesty**: `parcel/processor_test.go` targets a brand-new package;
  without the change, `go test ./parcel/...` fails to compile
  (package does not exist), so the test genuinely fails without the change.
  `go build ./... && go test ./parcel/... ./saga/...` reproduced locally —
  clean build, all relevant packages `ok`.

## Finding A (blocking): idempotency contract not documented in code

`kafka/message/parcel/custody/kafka.go:147-154` states why
`CommandRestoreParcel`/`CommandRemoveParcel` exist but never states, in the
doc comment itself, that both are idempotent (0 rows affected is success) —
this is explicit brief text (task-13-brief.md Step 3: "Both are idempotent: 0
rows affected is success") that the implementer's report
(`task-13-report.md` line 18) paraphrases into prose but did not carry into
the source doc comment. This is the only place in the codebase where this
contract will be documented for whoever implements the atlas-parcel side of
`RESTORE_PARCEL`/`REMOVE_PARCEL` later — the MTS twin's `kafka.go` also
doesn't say it, but the brief's binding constraint for this task explicitly
required it here, so the MTS precedent doesn't excuse the omission for this
file. Low functional impact today (atlas-parcel's custody consumer doesn't
exist yet), but it is a specific brief requirement that was dropped and is a
one-line fix.

## Not evaluable

- atlas-parcel's actual custody consumer implementation (does not exist yet
  in this branch) — cannot verify the wire contract against a live
  consumer; the diff explicitly documents that this is a "contract to
  honor in a later task."
- The compensator's real use of `RestoreParcelAndEmit`/`RemoveParcelAndEmit`
  and the handler.go dispatch of `AcceptToParcelAndEmit`/
  `ReleaseFromParcelAndEmit` — out of scope per the brief's Files list
  (Task 14 and later), not present in this diff.

## Verification run

```
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go build ./...                          # clean
go test ./parcel/... ./saga/...         # ok (cached, re-confirmed clean build)
```
