# Review: Task 12 — Orchestrator composite expansion for parcel custody

Range: `bc69a8a76..3c658eb49` (2 commits)
- `4ce4c4548` feat(orchestrator): expand transfer_to_parcel and withdraw_from_parcel
- `3c658eb49` fix(orchestrator): resolve parcel item snapshot for withdraw_from_parcel

Brief: `.superpowers/sdd/plan/task-12-brief.md`
Report: `.superpowers/sdd/plan/task-12-report.md`
Diff package: `.superpowers/sdd/plan/review-bc69a8a76..3c658eb49.diff`

## Scope confirmed

Files touched match the brief plus the ordered fix round:
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/parcel_expansion_test.go` (new)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/rest.go` (new, fix round)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/requests.go` (new, fix round)

No files outside `atlas-saga-orchestrator` were touched. `services/atlas-parcel` was read
(as required to verify the report's "no transformer change needed" claim) but not
modified — matches the report. Working tree is otherwise clean; the unrelated
uncommitted docs artifacts noted in the task brief were ignored as instructed.

## Requirement-by-requirement (brief + binding constraints)

1. **Mirrors the MTS/storage/cash-shop composite family** — PASS.
   `expandTransferToParcel` (`saga/processor.go:2161`) is a structural line-for-line
   mirror of `expandTransferToMts` (`saga/processor.go:1954`): same
   `compartment.RequestCompartment` call, same `fmt.Sscanf` linear-search idiom, same
   error message shape, `release_from_character` built first and `accept_to_parcel`
   second. `expandWithdrawFromParcel` (`saga/processor.go:2228`) mirrors
   `expandWithdrawFromMts`'s external-lookup-then-two-steps shape, adjusted for the
   ordered fix (see item 1 below).
   `isExpandableAction` (`saga/processor.go:1621`) and the dispatch switch in
   `expandAndProcessStep` (`saga/processor.go:1244`) both got the two new composite
   actions added — confirmed both by reading the diff and by `go build`.

2. **Meso-only transfer is exactly one step (RISK-2)** — PASS.
   `saga/processor.go:2165` guards the entire compartment lookup behind
   `payload.AssetId == 0`, returning a single `accept_to_parcel` step
   (`HasItem: false`) with no `TemplateId`/`Quantity`/snapshot fields set (i.e. the
   Go zero value), and never emits `release_from_character`. Pinned by
   `TestExpandTransferToParcel/meso_only` (`saga/parcel_expansion_test.go:372-401`),
   which asserts `require.Len(t, steps, 1)`, `acc.HasItem` false, and zero
   `TemplateId`/`Strength`. Ran locally: PASS.

3. **Item transfer emits two steps, snapshot from compartment by AssetId, "asset
   missing" is an error with no steps** — PASS.
   `saga/processor.go:2200-2222` linear-searches `comp.Assets` by `AssetId`, and on a
   miss returns `nil, fmt.Errorf(...)` — no steps. Pinned by
   `TestExpandTransferToParcel/item_parcel` and `/asset_missing`. Ran locally: PASS.

4. **`libs/atlas-constants` check, no cross-layer reach, line endings** — PASS.
   No new domain type/alias/numeric constant is introduced; the new `parcel.AssetData`
   and `parcel.RestModel` are the orchestrator's own REST-view types, following the
   exact same pattern as `mts.HoldingRestModel` and `compartment.CompartmentRestModel`
   (not a shared library candidate, and not reached into another layer's internals —
   the request client reads `atlas-parcel`'s already-public REST surface, same as every
   other cross-service REST client in this package). New files use LF only (verified via
   `grep -cP '\r'`, zero matches); no existing file's line endings were touched (diff
   is pure addition inside changed regions of existing files).

## Judgment items (fix round, not in the original brief)

### 1. New `parcel/rest.go` + `parcel/requests.go` REST client

**Verdict: correct and matches sibling shape.**

- `parcel/requests.go:40-57` (`RequestParcel`) is a field-for-field structural match
  of `compartment/requests.go`'s `RequestCompartment` (`getBaseRequest` →
  `requests.RootUrlFor(ctx, "<DOMAIN>")`, `fmt.Sprintf(root+resource, ...)`,
  `requests.GetRequest[RestModel](url)(l, ctx)`), not the paginated `mts.RequestHoldings`
  shape — correctly justified since this is a lookup-by-id, not a list scan.
- Tenant/span/env header handling is identical to every sibling client: `GetRequest`
  (`libs/atlas-rest/requests/decorated.go:9-16`) applies `SpanHeaderDecorator`,
  `TenantHeaderDecorator`, `EnvHeaderDecorator` unconditionally — the new client gets
  this for free by using the same `requests.GetRequest[A]` helper every other client in
  this package uses. No new HTTP idiom was invented.
- `parcel/rest.go`'s `AssetData` (lines 71-104) is byte-for-byte field-and-tag identical
  to `services/atlas-parcel/atlas.com/parcel/parcel/asset_data.go`'s `AssetData` — every
  field name, JSON tag, and type verified by direct comparison.
  `parcel/rest.go`'s `RestModel` (lines 113-137) matches
  `services/atlas-parcel/atlas.com/parcel/parcel/rest.go`'s `RestModel` on every field
  the orchestrator needs (`Id`, `WorldId`, sender/recipient identity, `Message`,
  `MesoAmount`, `FeePaid`, `ItemId *uint32`, `ItemType`, `Quantity`, `ItemSnapshot`,
  `Status`/`Quick`/`Returned`) — the orchestrator's copy correctly omits
  `CreatedAt`/`ReceivableAt`/`ExpiresAt`/`ResolvedAt`/`LastNotified`, which the
  expansion doesn't need; that's not a mismatch, api2go/JSON:API decoding tolerates
  extra server-side fields the client struct doesn't declare.
- `GetName`/`GetID`/`SetID`/`SetToOneReferenceID`/`SetToManyReferenceIDs` are the same
  five-method api2go boilerplate `mts.HoldingRestModel` and
  `compartment.CompartmentRestModel` both carry — correctly copied, not invented.
- **Endpoint verified against the real handler.** `GET /parcels/{parcelId}` is
  registered at `services/atlas-parcel/atlas.com/parcel/parcel/resource.go:41` and
  handled by `handleGetParcel` (lines 143-166), which calls `GetById` then
  `model.Map(Transform)` and marshals a single `RestModel`. `Transform`
  (`services/atlas-parcel/atlas.com/parcel/parcel/rest.go:59-82`) populates `ItemId`,
  `Quantity`, and `ItemSnapshot` directly from the domain `Model` — confirming the
  report's claim that **no atlas-parcel transformer change was needed**; I read the
  transformer myself rather than trusting the report, and it already exposes every
  field the fix consumes.
- `expandWithdrawFromParcel`'s lookup-error path (`saga/processor.go:2239-2241`) wraps
  and returns no steps, matching `expandTransferToParcel`'s/`expandWithdrawFromMts`'s
  sibling error shape exactly. Pinned by
  `TestExpandWithdrawFromParcel/withdraw_parcel_lookup_fails`. Ran locally: PASS.
- `withdraw_with_item`'s test assertion on real `TemplateId 1302000` and
  `Strength/Owner/Quantity` (not zeros) is a genuine regression pin for the coordinator's
  original finding — it would fail against the pre-fix code (zero-valued
  `TemplateId`/`AssetData`), confirmed by reading the diff of what changed.

No defects found in this half of the fix round.

### 2. Meso-only withdraw emits only `[release_from_parcel]`

**Verdict: correct against design.md §4.3, and a sound engineering call.**

- Design.md §4.3 (`docs/tasks/task-241-duey-parcel-delivery/design.md:266-273`)
  specifies `ParcelReceive` as a **two-step top-level saga**:
  `withdraw_from_parcel` (composite → `release_from_parcel` + `accept_to_character`)
  **followed by a separate, unconditional `award_mesos` step**. `award_mesos` is not
  part of the `withdraw_from_parcel` composite this task implements — it is a sibling
  saga step registered by whatever built the `ParcelReceive` saga's step list (out of
  this task's file set). Because `award_mesos` always runs regardless of what
  `withdraw_from_parcel` expands to, **the meso reaches the character on every
  withdraw, item or not** — omitting `accept_to_character` for a meso-only parcel
  does not silently drop the meso.
- `release_from_parcel` — the step design.md says is where "the parcel row
  transitions to `received` ... inside atlas-parcel's own transaction" — is emitted
  **unconditionally**, before the `pm.ItemId == nil` check
  (`saga/processor.go:2244-2253`), so the custody-release/status-transition fact still
  happens for a meso-only withdraw exactly as design.md requires.
  `TestExpandWithdrawFromParcel/withdraw_meso_only` pins this.
- The implementer's stated reasoning (no `HasItem`-style escape hatch exists on
  `AcceptToCharacterPayload`, and there is no sane `InventoryType`/`TemplateId` to
  hand a zero-item accept without inventing data) is sound: unlike
  `AcceptToParcelPayload`, which was purpose-built with `HasItem` for exactly this
  RISK-2 case, `AcceptToCharacterPayload` (`saga/model.go:998-1005`) has no such field,
  and it is reused across four other expansions (storage/cash-shop/MTS withdraws) where
  inventing an escape hatch would be out of scope for this task.
- **Step-accounting / compensation check**: nothing in this diff's scope assumes a
  fixed step count for `withdraw_from_parcel`'s expansion. The only place step count is
  referenced is a documentation comment (`saga/processor.go:1954`, `:2059`, "N=2
  steps... record for timeout scaling") on other composites; `expandWithdrawFromParcel`
  carries no such comment and none of `AtomicUpdateSaga`'s step-splicing logic
  (`saga/processor.go:1244-1276`) depends on the expanded slice having a specific
  length — it just replaces the composite step with however many concrete steps come
  back. A 1-step and a 2-step expansion are both handled identically by that splice.
  Compensation wiring for the new leaf actions (`release_from_parcel`,
  `accept_to_character`, `accept_to_parcel`) is not part of this diff and is out of
  scope for this review (Task 12 is expansion-only; handler/compensator registration is
  a later task's surface) — noted under Not evaluable, not held against this range.

No defects found in this half of the fix round either. This ruling is judged correct.

## Findings

### Non-blocking

1. **`TestIsExpandableActionCoversExpansionSwitch` was not updated for the two new
   composite actions.**
   `saga/mts_expansion_test.go:268-282` is the repo's own named regression guard for
   exactly the failure class this task is at risk of (the comment at
   `saga/processor.go:1176-1179` explicitly names this test as "pins the two in
   sync" — `isExpandableAction` vs. the `expandAndProcessStep` switch). The
   `composites` slice in that test still only lists
   `TransferToStorage, WithdrawFromStorage, TransferToCashShop, WithdrawFromCashShop,
   TransferToMts, WithdrawFromMts, MtsSettlePurchase, TradeSettlement` — it was not
   extended to include `TransferToParcel, WithdrawFromParcel` (nor `TransferToTrade`/
   `TradeUnwind`, which is a pre-existing gap from an earlier task and out of this
   range's scope). The actual production code is correct (both switch statements were
   updated together, confirmed by direct diff read and `go build`/`go test` passing),
   so this is not a functional defect — but the specific safety net the repo built to
   catch this exact regression class does not currently cover the two actions this
   task added. A one-line addition (`TransferToParcel, WithdrawFromParcel` to the
   `composites` slice) would close the gap.

## Not evaluable

- Compensation/handler wiring for the four new leaf actions
  (`release_from_parcel`, `accept_to_parcel`, `accept_to_character`,
  `release_from_character` reuse) is not part of this diff; Task 12's brief scopes
  this task to composite expansion only. Whether a mid-saga failure after
  `release_from_parcel` (before `award_mesos`) compensates correctly is a Task
  13/14-era concern that this range's file set cannot answer.
- The `award_mesos` step's own implementation and its registration into the
  `ParcelReceive` saga's step list are outside this diff's file set; I relied on
  design.md §4.3's saga table as the source of truth that it exists and runs
  unconditionally, per the task's own instruction to judge item 2 against §4.3
  rather than against code this range doesn't touch.

## Verification performed

- `go build ./...` (services/atlas-saga-orchestrator/atlas.com/saga-orchestrator) — clean.
- `go test ./saga/... -run Parcel -v` — all 7 subtests PASS (matches report).
- Read `services/atlas-parcel/atlas.com/parcel/parcel/resource.go` and `rest.go` directly
  (not trusting the report) to confirm the `GET /parcels/{parcelId}` handler and
  `Transform` already expose every field the fix needed.
- Read `services/atlas-parcel/atlas.com/parcel/parcel/asset_data.go` and diffed it
  field-by-field against the orchestrator's new `parcel.AssetData`.
- Read `compartment/requests.go` and `mts/requests.go` and `mts/rest.go` as the sibling
  baseline for the new `parcel/requests.go`/`parcel/rest.go`.
- Read `docs/tasks/task-241-duey-parcel-delivery/design.md` §4.3 and §12 (RISK-2) as the
  source of truth for the two judgment items.
- `grep -cP '\r'` on all five changed/new files — zero CRLF matches.
- `git status --porcelain services/atlas-saga-orchestrator services/atlas-parcel` — clean.

## Verdict

APPROVED_WITH_FINDINGS. Both fix-round judgment items are correct against the design
and the sibling pattern. One non-blocking test-completeness gap (finding 1).
