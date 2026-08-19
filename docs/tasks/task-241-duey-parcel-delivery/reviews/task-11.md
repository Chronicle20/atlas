# Review: Task 11 — Saga library: parcel custody actions, types and payloads

Commit range: `d78646ab0..bc69a8a76` (1 commit: `bc69a8a76`)
Brief: `.superpowers/sdd/plan/task-11-brief.md`
Reviewer: atlas-reviewer (Sonnet 5)

## Scope

Reviewed the full diff (`libs/atlas-saga/model.go`, `payloads.go`,
`unmarshal.go`, `unmarshal_test.go`, `world_transfer_test.go`) against the
brief's field-level interface table, plus `AcceptToMtsListingPayload` and
`TransferToMtsPayload` (read-only, in-repo) as the structural twin this task
is required to match. Ran `go build ./...`, `go test ./...`, `go vet ./...`,
`gofmt -l .` from `libs/atlas-saga` — all clean. Checked line endings
(no CRLF introduced) and `libs/atlas-constants` for pre-existing
type/constant duplication (none: only `world.Id`, `uuid.UUID`, `time.Time`,
and Go primitives are used, all pre-existing imports in this file).

`scope_confirmed`: the diff matches the brief's stated file list exactly —
`model.go`, `payloads.go`, `unmarshal.go`, `unmarshal_test.go`,
`world_transfer_test.go`. No unscoped files touched.

## Findings

### PASS — Type and Action constants (model.go)

- `ParcelSend Type = "parcel_send"`, `ParcelReceive Type = "parcel_receive"`
  added after `RemoteNpcUse`, each with an explanatory comment in the house
  style (`model.go:31-36` per diff hunk).
- `TransferToParcel`, `AcceptToParcel`, `ReleaseFromParcel`,
  `WithdrawFromParcel` added immediately after the MTS action block
  (`model.go:58-64`), with the exact design §4.2 comment wording: "transfer_to_parcel
  is a COMPOSITE expanded into release_from_character + accept_to_parcel, the
  same shape as transfer_to_mts." Matches brief's placement and wording
  instruction verbatim.
- String values match the brief exactly: `transfer_to_parcel`,
  `accept_to_parcel`, `release_from_parcel`, `withdraw_from_parcel`.

### PASS — Payload field sets, field-for-field (payloads.go)

- `TransferToParcelPayload`: all 17 fields present, in the brief's order,
  correct types (`uuid.UUID` x2, `uint32` x8, `world.Id`, `byte`, `string`
  x2, `bool`, `time.Time` x2). Matches `TransferToMtsPayload`'s shape
  (carries `SourceInventoryType`/`AssetId`/`Quantity` since the asset has not
  yet been deleted at this point in the composite).
- `AcceptToParcelPayload`: delivery-parameter fields plus `HasItem bool` plus
  the 24-field item-snapshot block. Snapshot block is byte-identical
  (field names, order, types, json tags) to `AcceptToMtsListingPayload`'s
  snapshot block (`payloads.go:889-911` pre-existing) — verified line by
  line. `HasItem` is a plain `bool` (not pointer), snapshot fields are
  concrete non-pointer types (`uint32`/`uint16`/`byte`/`string`), matching
  RISK-2's requirement that the zero-valued snapshot be representable
  without nils. Correctly omits `SourceInventoryType`/`AssetId`/`Quantity`-
  as-asset-qty (the asset is already gone by accept time) — same shape as
  `AcceptToMtsListingPayload` omitting them, so this is not a dropped field,
  it is the established pattern.
- `ReleaseFromParcelPayload`: `TransactionId`, `ParcelId`, `RecipientId` —
  matches brief exactly; matches `ReleaseFromMtsHoldingPayload`'s
  "carries only the row id" shape.
- `WithdrawFromParcelPayload`: `TransactionId`, `ParcelId`, `CharacterId`,
  `WorldId`, `InventoryType` — matches brief exactly.

### PASS — Unmarshal arms (unmarshal.go)

Four `case` arms added for `TransferToParcel`, `AcceptToParcel`,
`ReleaseFromParcel`, `WithdrawFromParcel`, each copying the existing
`case`-arm shape (unmarshal into the concrete payload type, wrap error with
the action name, `any(payload).(T)`). Structurally identical to the
surrounding MTS arms.

### PASS — Unmarshal tests (unmarshal_test.go)

Four new tests (`TestUnmarshalTransferToParcelStep`,
`TestUnmarshalAcceptToParcelStep`, `TestUnmarshalReleaseFromParcelStep`,
`TestUnmarshalWithdrawFromParcelStep`), each modelled on
`TestUnmarshalTransferToMtsStep`: builds a JSON step envelope, unmarshals,
type-asserts `step.Payload`, asserts exactly the fields the brief's table
specifies (`ParcelId`, `CharacterId`, `AssetId`, `MesoAmount`, `FeePaid`,
`Quick`, `ReceivableAt` for transfer; `ParcelId`, `RecipientId`,
`TemplateId`, `HasItem`, `Owner` for accept; etc.). Ran green.

### PASS — the `knownActions` hoist (world_transfer_test.go)

This is the item flagged for specific judgment. Verified by diffing the
pre-edit `otherActions` local slice (99 lines removed) against the new
package-level `knownActions` var (35 lines added) element-by-element:

- Every constant present in the old `otherActions` slice appears in
  `knownActions`, in the same order, with no omissions or reordering.
- The only content change is one inserted line —
  `TransferToParcel, AcceptToParcel, ReleaseFromParcel, WithdrawFromParcel,`
  — placed immediately after the MTS action group
  (`TransferToMts, ..., MtsBidEscrow,`) and before the guild-action group,
  mirroring exactly where the four constants were placed in `model.go`.
  This is the "plus exactly the four new parcel actions" the task called
  for — confirmed byte-for-byte against the diff hunk, not eyeballed.
- `TestWorldTransferActionConstantsAreUnique` asserts the identical property
  over the identical data: it still builds `worldTransferActions`
  (unchanged), still checks no two collide with each other, and still
  iterates `knownActions` (formerly `otherActions`) checking no collision
  with any pre-existing action. The rename to `knownActions` and reference
  swap is the only change to this test's body — behavior unchanged.
- `grep` across `libs/atlas-saga/` confirms `knownActions` appears only in
  `world_transfer_test.go` and `otherActions` no longer appears anywhere —
  no leakage into or duplication with other test files in the package.
- `TestParcelActionsAreKnown` is honest about what it can and cannot prove:
  because `knownActions` is a hand-written literal in the same file, the
  test is necessarily self-referential once the literal compiles — it
  cannot fail without also failing to compile (`TransferToParcel` etc. would
  be undefined) or without a human deleting the line by hand later. That is
  the same shape as the brief explicitly asked for ("assert all four actions
  appear in the action list ... enumerates") and the authorized resolution
  ("test against whatever enumeration mechanism the file actually uses").
  Noted as a non-blocking observation below, not a defect — the assignment
  itself has limited teeth by construction, and the implementer did not
  invent a stronger mechanism that wasn't authorized.

Verdict on the hoist: faithful. No content drift, no weakening of the
existing test, no leakage.

### PASS — repo conventions

- No new domain type/constant defined; only pre-existing `world.Id`,
  `uuid.UUID`, `time.Time` and Go primitives reused (self-review claim
  verified — `libs/atlas-constants` checked, nothing overlaps since all
  non-primitive fields are already-imported types in this file).
- Line endings: `payloads.go`, `model.go`, `unmarshal.go`,
  `unmarshal_test.go`, `world_transfer_test.go` all LF-only, no CRLF
  introduced (checked with a byte-level `\r\n` count, all zero).
- `gofmt -l .` clean, `go vet ./...` clean, `go build ./...` and
  `go test ./...` both green.
- Commit scoped to `libs/atlas-saga` only, matching the report's claim.

## Non-blocking

- `TestParcelActionsAreKnown` (`world_transfer_test.go:586-598`) is
  tautological in the sense described above: it can only fail via a compile
  error or a hand-edit removing a line from the same literal it is checking
  against. This is an accepted consequence of the authorized resolution
  (no parallel registry), not a new defect introduced by this implementer,
  but future readers should not mistake it for evidence that the four
  constants are wired anywhere outside this package — it proves the symbols
  exist and compile, nothing more.

## Not evaluable

None. Full diff was read; MTS payload analogs were read for the shape
comparison; build/test/vet/gofmt were run directly.

## Verdict

APPROVED. Every field set, type, action string, and comment matches the
brief and the MTS structural twin exactly. The authorized `knownActions`
hoist is faithful to its brief — content-identical plus exactly the four new
actions, same assertion preserved, no leakage. No repo-convention
violations found.
