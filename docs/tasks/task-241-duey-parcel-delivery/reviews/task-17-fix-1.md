# Review: Task 17, fix round 1 (closes B1)

Range: `680a44ff0..3b8de6cc0`
Prior finding closed: B1 — `duey_action_send_test.go` accept-path assertions checked
only `len(f.sagas)` and `sg.SagaType`, never `Steps`, proven by hardcoding
`Quick: false` into `buildParcelSendSaga` and all 13 subtests staying green.
Fix claim: test-only change.

## 1. Production code untouched

Confirmed. `git diff --stat 680a44ff0..3b8de6cc0` shows exactly one file:
`services/atlas-channel/atlas.com/channel/socket/handler/duey_action_send_test.go`
(+79/-4). `git diff 680a44ff0..3b8de6cc0 -- .../duey_action_send.go` and
`.../duey_action.go` both produce zero output — byte-identical to the
pre-fix commit. No stray edit left over from the implementer's RED
experiment.

## 2. Where did the expected values come from

- Step count/order (`award_mesos` → optional `consume_quick_delivery_ticket`
  → `transfer_to_parcel`), `award_mesos` first, `destroy_asset`
  present-iff-quick, and the "no destroy_asset when non-quick" loop all
  trace directly to the brief's table
  (`.superpowers/sdd/plan/task-17-brief.md:82-95`) and its comment citing
  design §4.3 (`duey_action_send_test.go:340-342`).
- `AwardMesosPayload.Amount`: brief row "npc send item and meso" states
  `Amount -(1000+5000)`; "quick send" states `-(1000+0)`. The test computes
  the expectation via `dueyparcel.TotalCost(tc.want.mesoAmount, tc.want.quick)`
  (`duey_action_send_test.go:358`) rather than a hardcoded literal.
  `TotalCost(1000, false)` = 1000 + Fee(1000)=0 + SendSurcharge(5000) = 6000
  = `-(1000+5000)` per the brief; `TotalCost(1000, true)` = 1000 + 0 + 0 =
  `-(1000+0)`. Values match the brief's formula. `TotalCost` is Task 16's
  own unit, independently covered by `parcel/fee_test.go` (`TestFee`,
  `TestTotalCost`), and is **not itself under review in this range** — so
  reusing it in the test does not duplicate untested logic. It does mean
  this specific assertion validates wiring (right field, right sign, right
  inputs threaded through) rather than re-deriving the fee formula from
  scratch; the formula itself is validated in `fee_test.go`, out of this
  unit's scope. Noted, non-blocking.
- `TransferToParcelPayload.{AssetId, Quantity, Quick, Message,
  SourceInventoryType}`: asserted against the case's own input literals
  (`tc.want.assetId`, `.quantity`, `.quick`, `.message`,
  `.sourceInventoryType`), which are set directly from what
  `newActionSendBytes`/the fixture wired in for each subtest
  (`duey_action_send_test.go:230-296`) — not read back from
  `buildParcelSendSaga`'s output. These are genuine "pass the input through
  unchanged" checks, matching the brief's intent stated in the comment at
  `duey_action_send_test.go:211-214`.
- "meso only" / "mailbox at nine" cases aren't literal brief-table rows,
  but their `want` values (`mesoAmount: 1000`, no item) are the direct
  consequence of the wire bytes each subtest already sent
  (`newActionSendBytes(0, 0, 0, 1000, ...)`), not values read off the
  implementation's output.

No assertion was found that matches the code with no independent source
other than the Amount/TotalCost case noted above, which is explainable and
non-blocking given Task 16's separate coverage.

## 3. Does the test bite per field

Reproduced the implementer's RED/GREEN experiment myself (not just trusted
the report):
- Restored `Quick: false` hardcode in `buildParcelSendSaga` → `go test
  ./socket/handler/... -run TestDueyActionSend` → `quick_send` fails at
  `duey_action_send_test.go:394` with `transfer_to_parcel quick = false,
  want true`, all others pass. Confirmed clean revert
  (`git diff --stat` empty after restoring the backup).
- Additional independent mutations, each reverted and confirmed clean:
  - `AssetId: assetId + 1` → 4 subtests fail at `:388` on `assetId`
    mismatch (both item and no-item cases, since `assetId+1` moves 0→1
    too). Confirms `AssetId` is checked independently.
  - Force the `destroy_asset` step unconditionally (`if true` instead of
    `if quick`) → 3 non-quick subtests fail at `:348` on step count
    (3, want 2). Confirms the "absent when non-quick" direction.
  - Suppress the `destroy_asset` step unconditionally (`if false` instead
    of `if quick`) → `quick_send` fails at `:348` on step count (2, want
    3). Confirms the "present when quick" direction.

Both directions of the `DestroyAssetPayload` presence-iff-quick contract
(check 4 in the brief) are independently covered: present-when-quick via
the step-count assertion at `:348` plus the direct `Action`/payload-type
check at `:365-370`; absent-when-not-quick via both the step-count
assertion and the explicit loop at `:372-378` that fails if any step in a
non-quick saga carries `saga.DestroyAsset`.

Assertion style: plain `t.Errorf`/`t.Fatalf` (not a third-party
assert/require library). `t.Fatalf` is used only where continuing would
panic (step-count mismatch before indexing `sg.Steps[...]`, and the two
payload type-assertion failures before dereferencing the concrete type);
every other check is `t.Errorf`, so multiple field mismatches in one
subtest each get reported rather than being masked by an early return.
No `reflect.DeepEqual`-on-a-fixture aggregate assertion is used anywhere in
the new code — each field is compared individually, so a single-field
regression cannot be masked by a passing aggregate.

## 4. DestroyAssetPayload presence-iff-quick

Both directions covered and independently verified by mutation testing
above (item 3). Not one-directional.

## 5. Out-of-scope changes in the diff

None. The diff touches only `duey_action_send_test.go`: one new import
(`dueyparcel "atlas-channel/parcel"`), the `expect` struct extension, six
literal `want` field additions on the pre-existing saga-creating subtests,
and the new step/payload assertion block. No change to fixtures, helpers,
or any other test's inputs/expectations outside the accept-path block.

## Verdict rationale

B1 is closed. Production code is verified byte-identical (not just
claimed). The new assertions bite independently per field, in both
directions of the presence-iff-quick contract, verified by mutation
testing rather than trusting the implementer's report. The one soft spot
— `AwardMesosPayload.Amount` validated via production's own `TotalCost`
rather than a fully independent literal — is explainable (Task 16's fee
formula has its own dedicated tests, out of this range) and non-blocking.
