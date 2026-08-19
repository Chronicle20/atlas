# Review — Task 28 batch 2/8 (gms_v72), commit `238faead6`

Range reviewed: `90c44a7a3..238faead6` (59 files, +1790/-24, per `git diff --stat`).
Brief: `.superpowers/sdd/plan/task-28-batch-gms_v72-brief.md`.
Report: `.superpowers/sdd/plan/task-28-batch-gms_v72-report.md`.

## Scope confirmed

The diff matches the brief exactly: STATUS.md/status.json regeneration, 25
new audit-report pairs under `docs/packets/audits/gms_v72/`, the
`docs/packets/ida-exports/gms_v72.json` splice, two new stacked-marker
fixture files (`parcel/clientbound/v72_test.go`,
`parcel/serverbound/v72_test.go`), and 4 new DUEY_ACTION evidence YAMLs.
No file outside the gms_v72 PARCEL/DUEY_ACTION surface was touched. No
scope mismatch.

## 1. RULING D — equality-vs-derive gate inventory

Enumerated every version gate under `libs/atlas-packet/parcel/`:

- `grep -rn MajorAtLeast libs/atlas-packet/parcel/` — **zero hits**. Neither
  `parcel.go`, `clientbound/parcel.go`, `clientbound/parcel_body.go`, nor
  `serverbound/action*.go` carries any `MajorAtLeast`/region/version
  conditional. Every PARCEL/DUEY_ACTION arm is therefore a legitimate
  equality-assertion target under Ruling D — there is nothing to
  over-apply the ruling to.
- The one adjacent version-gated codec is `model.Asset` (item encoding),
  gated at `MajorAtLeast(72)`/`(79)`/`(84)` (`libs/atlas-packet/model/asset.go:256,260,266,287,324,329,467,470,507,587,590,601,605,609,628`).
  v72 sits exactly on the `MajorAtLeast(72)` boundary and would diverge
  from v83's Asset bytes if an item were attached. Checked whether the
  v72 `Open`/`ParcelArrived` fixtures (`libs/atlas-packet/parcel/clientbound/v72_test.go:120-135,162-179`)
  exercise this path: they do not — `parcel.NewParcel(...)` is never
  followed by `.SetItem(...)`, so `HasItem()` is false and
  `parcel.Parcel.Encode` (`libs/atlas-packet/parcel/parcel.go:155-170`)
  never calls into `model.Asset.Encode`. The version-gate risk is real in
  the abstract but not triggered by this fixture, so it is not a defect
  here — it does however mean the item-attached path of `PARCEL::Decode`
  is still unpinned for v72 (same as v83; see §4).

Conclusion: Ruling D was applied correctly and not over-applied — there is
no gated arm in this family, so no arm needed independent v72 derivation
beyond what was done (each arm's `ida=0x...` marker cites a v72-native
decompile address, and the report/export prose ties the byte shape back to
this IDB directly, not merely to v83 equality by fiat).

## 2. Marker `ida=0x...` vs export entries

Cross-checked every `packet-audit:verify ... ida=0x...` marker in both new
test files against the corresponding spliced export entry's `"address"`
field (`docs/packets/ida-exports/gms_v72.json`):

- All 21 PARCEL clientbound markers (SendEnableActions 0x65fe59 …
  Open 0x65fd7f) match their `CParcelDlg::OnPacket#<Arm>` export entry
  addresses exactly.
- All 4 DUEY_ACTION serverbound markers (ActionSend 0x65d940, ActionReceive
  0x65af41, ActionDiscard 0x65b061, ActionClose 0x65f8e1) match their
  export entries and the four evidence YAMLs' `ida.address` fields
  identically.

No mismatch found.

## 3. Coverage — all 21 PARCEL + 4 DUEY_ACTION arms carry a marker

`yq -r '.operations[].key' docs/packets/dispatchers/parcel.yaml` → 21 keys;
`... duey_action.yaml` → 4 keys (SEND/RECEIVE/DISCARD/CLOSE). Enumerated
every marker in both v72 test files and confirmed a 1:1 set match against
both key lists (case-mapped: e.g. `ParcelSendEnableActions` ↔
`SEND_ENABLE_ACTIONS`). All 25 arms covered, none missing, none extra.

## 4. Fixture bytes traceability (client read order vs `Encode()`-derived)

The `Open`/`ParcelArrived` v72 fixtures derive part of `want` from
`p.Encode(l, ctx)(nil)` (`v72_test.go:126,168`) — the same
`Encode()`-derived pattern flagged non-blocking in v83
(`v83_test.go:110-124,152-168`, byte-for-byte identical construction).
This is an **inherited limitation, not a new regression**: the v72 fixture
does not add any new circularity beyond what v83 already carries, and
since no item is attached in either version's test, the risk is
symmetric. Still non-blocking, same disposition as batch 1.

All the other 19 clientbound arms and all 4 serverbound arms use literal
byte slices (`[]byte{tc.mode}`, hand-built `want`, or round-trip
Decode→Encode against a hand-authored `buf`) — these are traced to the
decompile prose in the export/report, not to `Encode()` output.

## 5. Export splice — additions only, duplicate key preserved

`git diff 90c44a7a3 238faead6 -- docs/packets/ida-exports/gms_v72.json` is
a single hunk (`@@ -3,6 +3,352 @@`), confirming the splice is a pure
insertion immediately after the `"functions": {` line and touches nothing
else in the file — independently reproduces the reported
**346 insertions, 0 deletions**.

Confirmed both pre-existing `"CPet::OnNameChanged"` occurrences survive
untouched (outside the diff hunk): line 412 is the annotated entry
(`"direction": "clientbound"`, per-field comments e.g. `"comment": "name"`
on `DecodeStr`), line 7525 is the bare stub (`"direction": ""`, empty
comments). Neither was replaced; the round-trip-loss risk the implementer
flagged was correctly avoided by doing the splice as a raw-text insert
rather than a JSON load/dump.

## 6. Evidence records — exactly the 4 DUEY_ACTION arms, hand-added `verifies:`

`ls docs/packets/evidence/gms_v72/parcel.*` → exactly
`parcel.serverbound.{ParcelActionSend,ParcelActionReceive,ParcelActionDiscard,ParcelActionClose}.yaml`.
No `parcel.clientbound.*` evidence file exists (correct per Ruling A — tier1:false
clientbound gets none). Each of the 4 YAMLs carries `direction: serverbound`,
`ida.address` matching its marker/export entry, and a `verifies:` list
pointing at its `v72_test.go#Test...V72` function. Matches RULING A exactly.

## Bonus finding — `CTabQuickSend::SendQuickDelivery` claim

Verified the report's claim about the extra v72-named call site
`CTabQuickSend::SendQuickDelivery @0x65c090`. It is **not** spliced as an
independent top-level export entry with its own address-verified call
list — it is documented only in prose inside the `CTabSend::SendParcel`
export entry's `"notes"` field and inside one call's `"comment"` field
(`docs/packets/ida-exports/gms_v72.json:286,314`). This is not a shortcut
specific to this batch: `gms_v83.json:16404,16432` does the identical
thing for its own quick-send call site (`sub_6F1DF5`), collapsing both
call sites' field lists into the single `CTabSend::SendParcel`/
`ActionSend` codec entry. The v72 test file's `"quick"` subtest
(`serverbound/v72_test.go:55-65`) round-trips a hand-built `sendQuickBytes()`
buffer through `ActionSend.Decode`/`Encode`, which is the actual
byte-level pin for the quick path — the marker just cites the NPC-path
address (0x65d940) as the arm's canonical `ida=`, matching the pattern
used by every other asymmetric-body arm in this family. No divergence
found; the "field shape matches with no divergence" claim is consistent
with both this IDB's decompile prose and the mirrored v83 pattern.

## Gate commands — reran independently, all exit 0

- `go run ./tools/packet-audit matrix --check` → exit 0
- `go run ./tools/packet-audit dispatcher-lint` → exit 0 (`dispatcher-lint: clean`)
- `go run ./tools/packet-audit fname-doc --check` → exit 0
- `go run ./tools/packet-audit operations --check` → exit 0
- `grep -n PARCEL docs/packets/dispatcher-lint-baseline.yaml` → exit 1 (absent, confirmed)
- `go test -count=1 ./libs/atlas-packet/parcel/...` → all pass
- `gofmt -l` on both new test files → clean
- `go vet ./libs/atlas-packet/parcel/...` → clean

## status.json cell states — read directly from the committed file

- `PARCEL` × `gms_v72`: `{"state": "verified", "opcode": 288}` (was `incomplete`/"no audit report")
- `DUEY_ACTION` × `gms_v72`: `{"state": "verified", "opcode": 64}` (was `incomplete`/"no audit report")

Matches the report's before/after claim.

## Findings

None blocking. One non-blocking, inherited from batch 1 (not introduced
here):

- **Non-blocking** — `libs/atlas-packet/parcel/clientbound/v72_test.go:120-135,162-179`
  (`TestParcelArrivedV72`, `TestParcelOpenV72`): `want` is partly derived
  from `parcel.Parcel.Encode()` rather than a fully independent literal.
  This is the same limitation flagged non-blocking on v83's equivalent
  tests; it does not regress anything new in this batch, and the
  item-attached path of `PARCEL::Decode` (which would actually exercise
  the `model.Asset` version gates) remains unpinned for both versions.

## Not evaluable

None. Everything in the diff's surface (fixtures, export splice, evidence,
STATUS.md/status.json, addresses) was checked against either the live
committed export/dispatcher files or independently rerun tool output.

## Verdict rationale

All six of the requested checks pass: the gate enumeration confirms no
over-application of Ruling D, every marker traces to a correct export
entry, all 25 arms are covered, the fixture-circularity issue is inherited
and symmetric with v83 (not a new defect), the export splice is
additions-only with the duplicate key intact, and evidence exists for
exactly the 4 DUEY_ACTION arms with hand-added `verifies:`. The bonus
`SendQuickDelivery` claim checks out against both this IDB's prose and the
v83 precedent for the same collapsing pattern.
