# Review: 8c6cdac73 — PARCEL + DUEY_ACTION verified on gms_v95 (Task 28 batch 7/8)

**Verdict: APPROVE**

IDA session used: `ecc757f4` (`GMS_v95.0_U_DEVM.exe.i64`), confirmed live via
`idb_list` before any decompile — filename matches exactly.

## Priority 1 — independent re-derivation of `GW_ItemSlotEquip::RawDecode`

Decompiled `0x4f8360` directly on `ecc757f4` (not taken from the report).
Read order after `GW_ItemSlotBase::RawDecode`:

`Decode1(nRUC) → Decode1(nCUC) → 15× Decode2 (niSTR/niDEX/niINT/niLUK/
niMaxHP/niMaxMP/niPAD/niMAD/niPDD/niMDD/niACC/niEVA/niCraft/niSpeed/niJump)
→ DecodeStr(sTitle,13) → Decode2(nAttribute) → Decode1(nLevelUpType) →
Decode1(nLevel) → Decode4(nEXP) → Decode4(nDurability) → Decode4(nIUC) →
Decode1(nGrade) → Decode1(nCHUC) → Decode2(nOption1) → Decode2(nOption2) →
Decode2(nOption3) → Decode2(nSocket1) → Decode2(nSocket2) → conditional
DecodeBuffer(liSN,8) → DecodeBuffer(ftEquipped,8) → Decode4(nPrevBonusExpRate)`.

This is byte-for-byte the same shape and order as the report's account and
as v92's already-verified fixture (which was the actual fix target of the
batch-6 revert). Cross-checked against `model.Asset.encodeEquipmentStats`
(`libs/atlas-packet/model/asset.go:530-545`): it writes exactly 15
`WriteShort` calls in the identical STR/DEX/INT/LUK/HP/MP/PAD/MAD/PDD/MDD/
ACC/EVA/hands/Speed/Jump order — a 1:1 match to the 15 `Decode2` reads.
`wantEquipItemBytesV95` (`v95_test.go:46`) encodes this as `make([]byte, 30)`
(30 bytes = 15 shorts), which is correct. The 12-byte potential trailer
(nGrade/nCHUC + 5×Decode2) is present at `v95_test.go:54-55`, gated behind
`MajorAtLeast(92)` in `asset.go:276-292`, matching the decompile exactly.
This batch does **not** repeat the batch-6 failure mode: the read order was
independently re-derived on this IDB, not copied from a neighbor version.

## Priority 2 — the garbled doc comment

Confirmed: the comment above `wantEquipItemBytesV95` (`v95_test.go:22-23`)
reads:

> `Decode1(nRUC)+Decode1(nCUC)+7x Decode2(niSTR/niDEX/niINT/niLUK/niMaxHP/ niMaxMP -- wait, actually 15 total Decode2 stat shorts)`

This is a genuine, uncleaned mid-sentence self-correction left in the
committed file: it opens with "7x" while naming only 6 fields, then
corrects itself to "15 total" without removing the wrong fragment. My
independent decompile above confirms the **true count is 15** — i.e. the
committed bytes (`make([]byte, 30)`, 30 bytes = 15 shorts) match the
corrected, true figure, not the mistaken "7x" one. So the fixture is
byte-correct, but the comment as committed is sloppy and must be flagged:
it should read cleanly as "15x Decode2 stat shorts (niSTR/niDEX/.../
niJump)" with the false-start and the parenthetical "wait" removed. This is
a documentation defect, not a correctness defect — reported per the task
brief, not fixed (this review is read-only).

## Priority 3 — promotion checks

- **Every `packet-audit:verify` marker address** in both
  `parcel/clientbound/v95_test.go` and `parcel/serverbound/v95_test.go` was
  cross-checked against a fresh decompile on `ecc757f4`:
  - `CParcelDlg::OnPacket` @0x692970 — mode read @0x6929a8, switch cases
    8/23/24/25/26/27, default → `NoticeResult`, `v1==18` gate @0x692ec1 —
    matches report and matches every marker address that cites this
    function's case bodies (0x692a73 OPEN, 0x692a15 OPEN_QUICK, 0x692aca
    PARCEL_REMOVED, 0x692c46 PARCEL_ARRIVED, 0x692cb6 ALARM_NAMED, 0x692dde
    ALARM_GENERIC, 0x692ec1 SUCCESSFULLY_SENT).
  - `CParcelDlg::NoticeResult` @0x68efd0 — switch @0x68efe0 with explicit
    cases 10-19(&28)/21/22, each case's Notice-string address matches every
    remaining clientbound marker exactly (0x68f001, 0x68f01b, 0x68f034,
    0x68f04e, 0x68f068, 0x68f081, 0x68f098, 0x68efe7, 0x68f0c5, 0x68f0dc,
    0x68f0f3).
  - `PARCEL::Decode` @0x4f88a0 — `DecodeBuffer(0xEA)` then `Decode1(hasItem)`
    + optional `GW_ItemSlotBase::Decode` — matches report.
  - All four DUEY_ACTION send sites decompiled directly: `CTabSend::
    SendParcel` @0x690140 (`COutPacket(&pkt,70)` @0x6902b8, `Encode1(2)` mode
    @0x6902c8), `CTabReceive::ReceiveParcel` @0x68f470 (`Encode1(4)`
    @0x68f548), `CTabReceive::DiscardParcel` @0x68f5e0 (`Encode1(5)`
    @0x68f672), `CParcelDlg::CloseParcelDlg` @0x68ef40 (`Encode1(7)`
    @0x68ef7f) — all four confirm opcode 70/0x046 via the `COutPacket`
    constructor argument, matching status.json's recorded opcode and the
    markers/evidence exactly.
- **`decompile_sha256` recomputed** for all four `ParcelAction*` evidence
  records by calling `evidence.FunctionHash` (the tool's own canonicalizer)
  against the committed `docs/packets/ida-exports/gms_v95.json` from a
  throwaway `tools/packet-audit/cmd/zzhashcheck` program (deleted after use,
  tree left clean): all four hashes matched the pinned YAML values exactly
  (`CParcelDlg::CloseParcelDlg` → `1631c2d8…`, `CTabReceive::DiscardParcel` →
  `ca9d93e5…`, `CTabReceive::ReceiveParcel` → `35a33bc3…`, `CTabSend::
  SendParcel` → `60862e3a…`).
- **Export splice additivity**: `git diff --numstat 8c6cdac73^ 8c6cdac73 --
  docs/packets/ida-exports/gms_v95.json` → `385 0`, i.e. 385 insertions, 0
  deletions — matches the report's claim exactly; confirmed no line-removal
  diff hunks present.
- **Non-vacuous fixture check**: flipped one byte of `wantEquipItemBytesV95`'s
  template-id encoding (`0xf0`→`0xf1`) and re-ran
  `TestParcelArrivedV95WithItem` — it failed with a clear byte diff at the
  mutated position, confirming the fixture is a real, discriminating
  assertion, not a tautology. Byte restored; `git status` confirmed clean
  afterward (no test-file diff persisted).
- **Gates**: re-ran all five independently from a clean tree —
  `matrix --check` (exit 0, same two pre-existing n-a notes as the report),
  `dispatcher-lint` ("clean"), `fname-doc --check` ("OK, 269 structs..."),
  `operations --check` ("OK, 0 absent-writer notes"), and
  `go test -count=1 ./libs/atlas-packet/parcel/...` (251 tests passed). All
  match the report's claimed output.
- **STATUS.md diff**: confirmed the commit's diff flips exactly the two
  target cells — `PARCEL` row's `0x17D` column ❌→✅ and `DUEY_ACTION` row's
  `0x046` column ❌→✅ — and touches no other version column in either row.

## Known open items observed (not fixed, per instruction)

- `docs/packets/dispatchers/duey_action.yaml`'s call-site roster (the header
  comment and the `fname:` field) still omits `CTabQuickSend::
  SendQuickDelivery` for every version, including gms_v95, even though this
  batch's own report and fixture comments correctly document the quick-send
  path (`CTabQuickSend::SendQuickDelivery` @0x68fa00, named directly in this
  IDB) as a second call site sharing opcode 70/mode 2. `STATUS.md`'s
  generated call-site column for this row does list `SendQuickDelivery`
  (sourced elsewhere), but the dispatcher YAML that is meant to be the
  source of truth does not carry it. Confirmed present, unresolved by this
  batch — as expected, since fixing it is out of this batch's scope.
- Not independently re-tested in this review, but consistent with the
  brief: `packet-audit export -ida-database <session>` is broken on
  GUI-adopted sessions (this batch correctly used the surgical hand-splice
  path instead), and `packet-audit seed-fname --write` reorders keys across
  seed templates (untouched by this commit).

## Tree state

`git status` is clean at the end of this review except for a pre-existing,
unrelated modification to `docs/tasks/task-241-duey-parcel-delivery/
agent-ledger.tsv` (a dispatch-log append made before this review started,
not touched by any action in this review).
