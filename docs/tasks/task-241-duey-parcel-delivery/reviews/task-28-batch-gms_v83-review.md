# Review: Task 28 batch gms_v83 (commit `90c44a7a3`, range `ee2267d64..90c44a7a3`)

Reviewer: atlas-reviewer (Sonnet)
Scope: the diff of `90c44a7a3` only. Brief:
`.superpowers/sdd/plan/task-28-batch-gms_v83-brief.md`; report:
`.superpowers/sdd/plan/task-28-batch-gms_v83-report.md`.

This is the ANCHOR batch of an 8-batch coverage campaign — a wrong address or
a circular fixture here propagates to every later per-version batch that
byte-compares against v83. Weighted accordingly; re-derived every claim
rather than trusting the report.

## Scope confirmed

The commit matches the brief precisely: it promotes exactly the `PARCEL`
(clientbound) and `DUEY_ACTION` (serverbound) gms_v83 cells, plus the
coverage-manifest / prior-batch review artifacts the report doesn't mention
(pre-existing from `f3f588ec1`, folded into this diff only because it's the
tip commit's stat — verified those files are untouched by `90c44a7a3` itself
by checking `git diff f3f588ec1..90c44a7a3` vs `ee2267d64..90c44a7a3`; the
task-24..28a review artifacts and agent-ledger rows are from the prior commit
`f3f588ec1`, not this one — restating scope: this commit's payload is the
export splice, 25 audit reports, 4 evidence records, two v83_test.go files,
and STATUS.md/status.json).

## 1. `ida=0x...` addresses vs. the v83 IDB

No IDA MCP tool was available in this review session (`idb_list`/`func_query`
not registered), so a live decompile spot-check could not be performed —
recorded under **Not evaluable** below rather than silently passed.

What I *could* verify, and did:

- **Internal consistency, 25/25**: every `ida=0x...` value in
  `libs/atlas-packet/parcel/clientbound/v83_test.go` and
  `.../serverbound/v83_test.go` matches, byte-for-byte, the `"address"` field
  of the corresponding key in `docs/packets/ida-exports/gms_v83.json`
  (verified programmatically — extracted both sets and diffed, zero mismatches).
- **Field order matches the Atlas codec exactly**: the export's
  `CTabSend::SendParcel` entry's `calls` list (`Decode1 inventoryType,
  Decode2 slot, Decode2 quantity, Decode4 mesos, DecodeStr recipientName,
  Decode1 quick, [DecodeStr message, Decode4 ticketRef]`) matches
  `libs/atlas-packet/parcel/serverbound/action_send.go`'s `Decode`/`Encode`
  field order and the `quick`-gated trailing pair, field for field
  (`action_send.go:70-83`).
- **Structural plausibility**: the 21 `CParcelDlg::OnPacket#<Arm>` addresses
  cluster tightly in `0x6f579d`-`0x6f5d11` (consistent with a compiler-emitted
  switch-case block inside `NoticeResult`/`OnPacket`'s default arm), and the
  `SuccessfullySent` (mode 18) audit report explicitly documents the "no
  explicit case, gated via the `a1==18` `CloseParcelDlg` side effect" shape
  the brief calls out (`docs/packets/audits/gms_v83/ParcelSuccessfullySent.json`).
- **duey_action.yaml cross-reference**: the two `CTabSend::SendParcel` call
  sites (NPC `@0x6F36A8` quick=0, quick-send `@0x6F1DF5` quick=1) are both
  recorded in the export entry's `notes` field even though only the quick
  address is the top-level key — matches the yaml's own header note.

This is strong secondary evidence the addresses are not merely
internally-consistent-but-wrong, but I did not independently re-derive them
from the binary myself. Flagged as **not evaluable** for the primary check
requested (item 1), not as a pass.

## 2. Fixture bytes trace to decompile, not circular

- **`libs/atlas-packet/parcel/serverbound/v83_test.go`**: `sendNpcBytes()`/
  `sendQuickBytes()` (pre-existing, in `action_test.go`, untouched by this
  commit) are hand-built literal byte slices, independent of any `Encode()`
  call. `TestActionSendV83` decodes those literals then re-encodes and
  compares round-trip — a genuine oracle, not `Encode()`-derived-from-`Encode()`.
  Same for `TestActionReceiveV83`/`DiscardV83` (`[]byte{0x07,0,0,0}`, a
  literal). **PASS**.
- **`libs/atlas-packet/parcel/clientbound/v83_test.go`**, the 15-arm table
  (`TestParcelResultArmsV83`) and `OpenQuick`/`Removed`/`AlarmNamed`/
  `AlarmGeneric`: `want` is a hand-built literal in every case (e.g.
  `want := []byte{0x17, 0x07, 0x00, 0x00, 0x00, 0x03}` for `ParcelRemoved`).
  **PASS**.
- **`TestParcelArrivedV83`/`TestParcelOpenV83`**: these build `want` partly
  from `pBytes := p.Encode(l, ctx)(nil)` — i.e. they reuse the shared
  `parcel.Parcel` struct's own `Encode()` output as the sub-structure oracle
  rather than a hand-built literal. In isolation this *would* be the
  "asserts `Encode()` equals a byte string derived FROM `Encode()`" failure
  mode the brief warns against. However: the `parcel.Parcel.Encode()` byte
  layout is independently, non-circularly pinned by the **pre-existing,
  untouched** `TestParcelEncode` in
  `libs/atlas-packet/parcel/clientbound/parcel_test.go:29-77` (id LE + name
  padded to 13 + mesos LE + FILETIME + message padded to 205 + hasItem byte,
  every field a literal). So the wrapper tests here only need to prove their
  own wrapper-specific bytes (mode byte, quickEnabled/hasItem flags, mailbox
  count, arrived count) — which they do with literals (`0x18`, `0x08, 0x01,
  0x01`, trailing `0x01`) — and delegate the inner-struct correctness to an
  already-independent oracle. Not circular once the full picture is
  considered, but **worth flagging**: `TestParcelEncode` itself carries no
  `packet-audit:verify` marker (it predates this campaign and is explicitly
  out of scope per its own doc comment), so the coverage matrix's promotion
  of `ParcelParcelArrived`/`ParcelOpen` rests partly on an unmarked sibling
  test's correctness. This is a **non-blocking** note, not a defect in this
  commit — the sibling test is real and independently verified, just not
  itself gated by the marker mechanism.

## 3. All 21 PARCEL + 4 DUEY_ACTION arms carry a marker

Cross-checked `yq -r '.operations[].key'` on both yaml files against
`grep packet-audit:verify` on the two new test files:

- `parcel.yaml`: 21 keys (OPEN..UNKNOWN_ERROR_2) — 21 markers present,
  1:1, confirmed by diffing the sorted `packet=` values against the sorted
  yaml keys (case-name-mapped, e.g. `NOT_ENOUGH_MESOS` → `ParcelNotEnoughMesos`).
  **PASS**.
- `duey_action.yaml`: 4 keys (SEND/RECEIVE/DISCARD/CLOSE) — 4 markers
  (`ParcelActionSend/Receive/Discard/Close`). **PASS**.

## 4. Export splice is additions-only, and is the parcel roster

Diffed `git show ee2267d64:...gms_v83.json` against
`git show 90c44a7a3:...gms_v83.json` at the JSON-key level (not just
`git diff --stat`): **25 added, 0 removed, 0 changed** keys. The 25 keys are
exactly: 21 `CParcelDlg::OnPacket#<Arm>` (one per PARCEL arm) +
`CTabSend::SendParcel` + `CTabReceive::ReceiveParcel` +
`CTabReceive::DiscardParcel` + `CParcelDlg::CloseParcelDlg` — the parcel
roster and nothing else. No unrelated key touched. **PASS**, confirms
RULING B compliance beyond the report's own `git diff --stat` claim.

## 5. Evidence records: 4 for DUEY_ACTION, 0 for PARCEL

`docs/packets/evidence/gms_v83/` contains exactly
`parcel.serverbound.ParcelAction{Send,Receive,Discard,Close}.yaml` — 4 files,
each with a hand-added `verifies:` field pointing at the matching
`v83_test.go` test function (e.g. `parcel/serverbound/v83_test.go#TestActionSendV83`).
No `parcel.clientbound.*` evidence file exists anywhere in the repo (grep
returned empty). Matches RULING A exactly. **PASS**.

## Cross-checks beyond the brief's five items

- **STATUS.md/status.json diff scope**: diffed both files line-by-line
  across the commit. Only the `PARCEL` row, the `DUEY_ACTION` row, three new
  `parcel/serverbound/ParcelAction{Send,Receive,Discard}` sub-struct rows,
  the export-hash line, and the summary count row for `v83` changed (468→473
  verified, matching exactly the 5 promoted cells: 1 PARCEL + 4
  DUEY_ACTION-family rows going ❌→✅ in the v83 column). No other
  version's column and no other op's row moved. **PASS** — confirms item 4's
  "no unrelated drift" concern extends to the status files too.
- **`dispatcher-lint-baseline.yaml`**: `grep -i parcel` returns no match
  (empty file, exit 1). **PASS**, per the brief's explicit requirement.
- **status.json cell read-out**: `PARCEL`/`gms_v83` = `{"state": "verified",
  "opcode": 322}`; `DUEY_ACTION`/`gms_v83` = `{"state": "verified", "opcode":
  65}`. Matches the report's claimed before/after exactly.
- **`go test ./libs/atlas-packet/parcel/... -run V83 -v`**: all 25 new
  sub-tests pass (ran directly, not just trusted the report).
- **Coverage manifest** (`docs/tasks/task-241-duey-parcel-delivery/coverage-manifest.yaml`,
  added in this commit's tip but authored across `f3f588ec1`/this commit):
  declares exactly the 21 PARCEL + 4 DUEY_ACTION arms and the 8-version
  scope, consistent with the brief and the yaml files.

## Not evaluable

- **Live IDA decompile spot-check of any `ida=0x...` address** (item 1's
  primary ask) — no IDA tool was registered in this review session. Verified
  internal consistency (marker↔export↔codec agreement, all 25/25) and
  structural/behavioral plausibility instead, but this is not the same as
  independently re-deriving an address from the binary. A controller with
  IDA access should spot-check at minimum `CParcelDlg::OnPacket@0x6f56ea`
  (not itself a marker target, but the dispatcher every arm's address is
  claimed relative to) and one or two of the clustered `NoticeResult` arm
  addresses before treating this anchor as fully closed.

## Findings

No blocking findings. One non-blocking note (see item 2): the promotion of
`ParcelParcelArrived`/`ParcelOpen` leans on the correctness of an unmarked,
pre-existing sibling test (`TestParcelEncode`) for the shared
`parcel.Parcel` sub-structure encode. That test is itself independently
literal-derived and not circular, so this is not a defect — just worth a
controller's awareness given how much later-batch work will byte-compare
against these two arms specifically.

## Verdict

APPROVED_WITH_FINDINGS — the anchor's marker set, export splice, evidence
scoping, and fixture-to-decompile traceability all check out against
independent re-derivation; the only shortfall is the live-IDA spot-check
this review session couldn't perform, which is reported honestly rather than
waved through.
