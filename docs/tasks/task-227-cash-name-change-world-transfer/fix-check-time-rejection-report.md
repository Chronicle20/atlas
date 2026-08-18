# Fix: wire the CHECK-time rejection path (step 4, closes OQ-7)

Implements step 4 of the "Suggested fix order" in
`bug-world-transfer-eligibility-reasons.md` — the last outstanding step.
Steps 1, 2, 3, 5 were already landed on this branch.

## What changed

### 1. `services/atlas-character/atlas.com/character/pending_change/processor_eligibility.go`

Split the 11-gate eligibility table into destination-INDEPENDENT gates
(`is_gm`, `banned`, `is_guild_master`, `in_family`, `trade_open`,
`merchant_open`, `mts_listings_open`) and destination-DEPENDENT gates
(`world_same`, `world_unknown`/`world_full`, `no_character_slot`,
`name_taken`), without duplicating any gate's underlying dependency-lookup
logic:

- Every gate's logic moved into its own `checkXxx` method (`checkWorldSame`,
  `checkIsGM`, `checkWorldStatus`, `checkCharacterSlot`, `checkNameTaken`,
  `checkBanned`, `checkGuildMaster`, `checkInFamily`, `checkTradeOpen`,
  `checkMerchantOpen`, `checkMtsHolding`), each returning
  `(reason string, rejected bool, err error)` — a uniform shape so a single
  `runGates` helper can orchestrate any subset in any order without a switch
  or copy-pasted err-handling.
- `evaluateTransferEligibility` (BUY-time, unchanged behavior/order) is now a
  data-shaped slice of all 11 `checkXxx` calls in the original order, fed to
  `runGates`.
- `evaluateTransferEligibilityIndependent` (new) is the same shape, holding
  only the 7 destination-independent gates, in the same relative order.
- `CheckTransferEligibilityIndependent(characterId uint32) (bool, string, error)`
  (new) resolves the character and runs only the independent half, with no
  side effect — the destination-free counterpart to the existing
  `CheckTransferEligibility(characterId, destinationWorldId)`.
- Added to the `Processor` interface (`processor.go`).

### 2. `services/atlas-character/atlas.com/character/pending_change/resource.go`

New route, registered exactly like the existing `transfer-eligibility` route
(outside the `/pending-changes` subrouter prefix, same `registerGet` wiring):

```
GET /characters/{characterId}/transfer-eligibility-independent
```

`handleGetTransferEligibilityIndependent` calls
`CheckTransferEligibilityIndependent` and reuses the existing
`EligibilityRestModel` (`rest.go`) — no new wire shape, since the response
shape (`eligible`, `reason`) is identical to the destination-bearing route.

### 3. `services/atlas-channel/atlas.com/channel/pendingchange/{rest.go,requests.go,processor.go}`

Mirrored the destination-free route on the atlas-channel side, following the
existing REST-client pattern in this package (`requestByCharacterId` /
`GetByCharacterId`):

- `rest.go`: `EligibilityRestModel` (mirrors atlas-character's own).
- `requests.go`: `requestTransferEligibilityIndependentUrl` /
  `requestTransferEligibilityIndependent` (a plain `requests.GetRequest`, no
  special error-body handling needed — this route never returns a rejection
  status, only `eligible: false` in a 200 body).
- `processor.go`: `CheckTransferEligibilityIndependent(characterId uint32) (bool, string, error)`
  added to the `Processor` interface and implemented via
  `requests.Provider[EligibilityRestModel, EligibilityRestModel]`.

### 4. `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_transfer_world_possible.go`

- New seam var `checkPossibleTransferEligibilityIndependentFunc`, following
  this file's existing `checkPossible*Func` pattern, wired to
  `pendingchange.NewProcessor(...).CheckTransferEligibilityIndependent`.
- The handler now calls it **after** the credential-validation and
  PIC-attempt-lockout logic (both left byte-for-byte untouched — the
  birthday-verification crash fix and the license-notice ordering are
  preserved) and **before** the world-list lookup:
  - An infrastructure error refuses via the existing `UNKNOWN_ERROR` body,
    the same fail-closed posture the world-list lookup below it already
    uses.
  - `eligible == false` refuses via
    `cashcb.CheckTransferWorldPossibleResultRejectedBody(characterId, reason, nil)`
    — the previously-dead-code mapper, now wired. `in_family` resolves to
    its own confirmed arm (`IN_FAMILY`, StringPool 5017); every other reason
    resolves to `UNKNOWN_ERROR`, exactly as
    `checkTransferWorldPossibleReasonArms` (libs/atlas-packet, untouched by
    this step) already specifies.
  - Neither branch emits any `POP_UP` world message — a rejected CHECK
    result is the packet's own arm, not a broadcast, so the license-notice
    modal-collision fix from step 1 is unaffected on this path too.
- The handler's doc comment is rewritten: the old paragraph asserting "no
  further gate evaluation" (the design gap this step closes) is replaced
  with the accurate description of the independent-gate wiring, its
  rejection-body routing, and why the BUY-time re-evaluation is still
  necessary (it remains the authoritative check; this one is advisory).
- Mapper file (`libs/atlas-packet/cash/clientbound/check_transfer_world_possible_result.go`)
  is **unchanged** — its reason→arm derivation (including the documented
  reasoning for why `is_guild_master`/`is_gm`/`banned` fold to
  `UNKNOWN_ERROR` rather than their own StringPool ids) was explicitly out
  of scope for this step; only the wiring that calls it changed.

### 5. Tests

`services/atlas-character/atlas.com/character/pending_change/processor_eligibility_test.go`:
- `TestCheckTransferEligibilityIndependentEvaluatesOnlyTheIndependentHalf` —
  unit-level pin: an `in_family` rejection surfaces through
  `CheckTransferEligibilityIndependent`; a character passing every gate
  (including one whose destination-dependent gates were never asked about,
  since this entry point takes no `destinationWorldId`) reports eligible.

`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change_possible_test.go`
(the shared `checkPossibleHandlerEnv` test harness both CHECK handlers use):
- Added `transferEligible`/`transferEligibleReason`/`transferEligibleErr`
  fields, defaulted to `eligible: true` in `newCheckPossibleHandlerEnv` so
  every pre-existing ALLOWED-path test needed no change, plus the
  `checkPossibleTransferEligibilityIndependentFunc` seam swap and two
  builder methods (`withTransferEligibilityIndependentRejected`,
  `withTransferEligibilityIndependentErr`).

`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_transfer_world_possible_test.go`
(new tests, handler-level, exercising the real
`CashShopCheckTransferWorldPossibleHandleFunc`):
- `TestTransferWorldPossibleRejectsOnIndependentGate` — `in_family`
  rejection resolves to the configured `IN_FAMILY` byte.
- `TestTransferWorldPossibleRejectsOnIndependentGateWithoutDedicatedArm` —
  `banned` rejection resolves to `UNKNOWN_ERROR`, proving the wiring is not
  special-cased to `in_family` alone.
- `TestTransferWorldPossibleEligibilityCheckErrorRefuses` — an
  infrastructure error on the eligibility check refuses (`UNKNOWN_ERROR`),
  never a false `ALLOWED`.
- `TestTransferWorldPossibleRejectionNeverEmitsStorageWarning` — a rejected
  CHECK result never triggers the FR-4.7 `POP_UP` warning (step 1's fix
  extends to this new branch too).

`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation_reason_test.go`
(extended per the brief, not duplicated):
- `TestCheckTransferWorldPossibleReasonKeyRoutesInFamilyToItsOwnArm` —
  complements the existing `check_unavailable`-folds-to-`UNKNOWN_ERROR` test
  immediately above it with the NEW-contract counterpart: `in_family`
  resolves to its own `IN_FAMILY` arm, not the generic fallback. Doc comment
  cross-references `TestTransferWorldPossibleRejectsOnIndependentGate` as
  the handler-level exercise of the same seam.

### 6. `docs/tasks/task-227-cash-name-change-world-transfer/design.md`

- The OQ table's OQ-7 row is rewritten from "deferred" to "Resolved",
  explaining that the wire-code derivation itself was already correct and
  landed (dead code); what was missing was the wiring, closed by this step.
- New `### 6.1 Destination-independent vs. destination-dependent gates
  (closes OQ-7)` subsection after §6 (Reason taxonomy): documents the split,
  which routes carry which half, why the CHECK op cannot reach the
  destination-dependent half, and that BUY time is unchanged (full table,
  `errors`-table path, still authoritative).

## Cross-service seam, traced by hand (CLAUDE.md's rule)

atlas-character's new `CheckTransferEligibilityIndependent` →
`GET .../transfer-eligibility-independent` → atlas-channel's
`pendingchange.CheckTransferEligibilityIndependent` →
`checkPossibleTransferEligibilityIndependentFunc` →
`CashShopCheckTransferWorldPossibleHandleFunc` →
`cashcb.CheckTransferWorldPossibleResultRejectedBody` → the wire byte the
client renders. Every hop is exercised by a test that asserts the NEW
contract, not just that the old ALLOWED path still works:

- atlas-character hop: `TestCheckTransferEligibilityIndependentEvaluatesOnlyTheIndependentHalf`.
- atlas-channel REST-client hop: covered indirectly by the handler-level
  tests below (the seam var is swapped at that layer, per this codebase's
  established `checkPossible*Func` pattern — the same pattern
  `checkPossibleAccountGetByIdFunc` already uses for the account lookup).
- atlas-channel handler hop, in_family →
  `TestTransferWorldPossibleRejectsOnIndependentGate` (asserts the specific
  `IN_FAMILY` wire byte, not just "not ALLOWED").
- atlas-channel handler hop, a reason with no dedicated arm (banned) →
  `TestTransferWorldPossibleRejectsOnIndependentGateWithoutDedicatedArm`.
- Failure mode (infra error) →
  `TestTransferWorldPossibleEligibilityCheckErrorRefuses`.
- The step-1 regression this touches (no `POP_UP` on any CHECK path) →
  `TestTransferWorldPossibleRejectionNeverEmitsStorageWarning`.
- Mapper-level pin, independent of the handler →
  `TestCheckTransferWorldPossibleReasonKeyRoutesInFamilyToItsOwnArm`.

## Verification

```
$ cd services/atlas-character/atlas.com/character && go build ./...
(no output)
$ go test -count=1 ./pending_change/...
ok  	atlas-character/pending_change	148.940s
$ go test -count=1 ./...
ok  	atlas-character/...   (full module, all packages OK, no FAIL -- see note below)
```

```
$ cd services/atlas-channel/atlas.com/channel && go build ./...
(no output)
$ go test -count=1 ./...
ok (all packages, including atlas-channel/socket/handler and atlas-channel/pendingchange -- no FAIL)
```

```
$ cd libs/atlas-packet && go build ./... && go test -count=1 ./cash/...
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound	0.042s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound	0.043s
```

Note: the atlas-character full-module `go test ./...` run (which includes
several testcontainers-backed suites and takes minutes) was launched in the
background per the tool-call budget; its completion was confirmed with exit
code 0 and no FAIL lines before this report was finalized. The targeted
`pending_change` package run above (which contains every file this step
touched) was run to completion in the foreground and is the primary
evidence.

All output pristine — no warnings, no skips, no `-race` (not part of the
implementer's verification scope).

## Self-review

- Gate logic itself was moved, not rewritten: every `checkXxx` method's body
  is the same dependency call, same error log line, same affirmative-reason
  string as before the split — verified by diffing the extracted bodies
  against the pre-split `evaluateTransferEligibility` (no `check_unavailable`
  string handling changed either, since step 3 already landed that).
- BUY time (`evaluateTransferEligibility`, `RequestWorldTransfer`, the
  `errors`-table path in `cash_shop_operation.go`) is untouched in behavior:
  same 11 gates, same order, same `errors`-table routing.
- The mapper (`check_transfer_world_possible_result.go`) is byte-for-byte
  unchanged — grep confirms no diff in that file on this branch beyond step
  3's already-landed `check_unavailable` addition.
- The CHECK handler emits no `POP_UP` on any path, rejected or allowed —
  confirmed by reading the full handler body: `announceTransferWorldPossible`
  is the only announce call on every branch, and it only ever writes
  `CashShopCheckTransferWorldPossibleResultWriter`, never
  `chatpkt.WorldMessageWriter`.
- Credential validation and PIC-attempt-lockout logic is byte-for-byte
  unchanged (same two `checkPossibleRecordPicAttemptFunc` calls, same
  ordering, same log lines) — the new gate check is inserted strictly after
  both, never interleaved with them.
- `EligibilityRestModel` reused rather than duplicated (both the
  atlas-character response shape and — as a new but structurally identical
  type — the atlas-channel REST-client-side decode target), since the wire
  shape (`eligible`, `reason`) did not change for the new route.
- Followed the Builder-pattern / existing-seam conventions throughout: no
  `*_testhelpers.go` file, no invented atlas-constants type, the new REST
  client method follows `data/cash/processor.go`'s
  `requests.Provider[RestModel, RestModel]` + identity-transform pattern
  used elsewhere in this same service.

## Issues or concerns

- The brief explicitly instructed not to rewrite or re-derive the mapper's
  arms. The diagnosis document's own later "Rendered StringPool text"
  section lists `is_guild_master -> 4004` and `banned -> 4009` as
  CHECK-result-only strings, which on its face looks like it could justify
  routing those two reasons to their own arms as well (not just
  `in_family`). I did not do this — the brief's instruction ("Do not rewrite
  the mapper or re-derive its arms — it was derived correctly. Wire it.") is
  explicit and I followed it literally rather than resolving that tension
  myself. If the controller wants `is_guild_master`/`banned` to also get
  distinct arms, that is a mapper change, not a wiring change, and is a
  natural follow-up rather than something folded into this step.
- `tools/verify.sh` was not run, per the implementer contract — that is the
  controller's `atlas-verifier` dispatch to run.
